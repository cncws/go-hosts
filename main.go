package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cncws/hosts-go/cmd/flags"
	"github.com/fsnotify/fsnotify"
)

var hostsFile = ""
var NeedUpdateImmediately = false

func init() {
	flag.StringVar(&flags.DataDir, "data-dir", "", "hosts profile 目录，默认用户目录下的 .hosts")
	flag.DurationVar(&flags.UpdateInterval, "update-interval", time.Hour, "remote profile 更新间隔，单位秒")
	flag.Parse()

	if flags.DataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("获取用户目录失败")
		}
		flags.DataDir = filepath.Join(home, ".hosts")
	}
	if os.MkdirAll(flags.DataDir, 0755) != nil {
		log.Fatal("创建工作目录失败")
	}
}

func init() {
	if os := runtime.GOOS; os == "windows" {
		hostsFile = "C:/Windows/System32/drivers/etc/hosts"
	} else {
		hostsFile = "/etc/hosts"
	}
}

func getProfiles() ([]string, error) {
	files := []string{}
	err := filepath.Walk(flags.DataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".local") || strings.HasSuffix(path, ".remote") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	filesname := make([]string, len(files))
	for i, file := range files {
		filesname[i] = filepath.Base(file)
	}
	log.Printf("工作目录 %v\n", flags.DataDir)
	log.Printf("读取配置 %v\n", filesname)
	return files, nil
}

func readLocal(file string) ([]string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

func readRemoteHistory(file string) ([]string, error) {
	return readLocal(file + ".history")
}

func writeRemoteHistory(file string, data []byte) error {
	return os.WriteFile(file+".history", data, 0644)
}

func readRemote(file string) ([]string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	urlString := strings.Split(string(content), "\n")[0]
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(urlString)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	writeRemoteHistory(file, body)
	return strings.Split(string(body), "\n"), nil
}

func readProfile(file string) ([]string, error) {
	lines := []string{"# profile begin: " + filepath.Base(file)}
	if strings.HasSuffix(file, ".local") {
		content, err := readLocal(file)
		if err == nil {
			log.Printf("本地配置 %s 已读取\n", filepath.Base(file))
			lines = append(lines, content...)
		}
	}

	if strings.HasSuffix(file, ".remote") {
		content, err := readRemote(file)
		if err == nil {
			log.Printf("远程配置 %s 已读取\n", filepath.Base(file))
			lines = append(lines, content...)
		} else {
			content, err = readRemoteHistory(file)
			if err == nil {
				log.Println("远程配置读取失败，延用上一次配置")
				lines = append(lines, content...)
				lines = append(lines, "# 读取远程配置失败，延用上一次配置")
			} else {
				log.Printf("远程配置 %s 读取失败\n", filepath.Base(file))
			}
		}
	}

	lines = append(lines, "# profile end, update at "+time.Now().Format(time.RFC3339), "")
	return lines, nil
}

func writeSystemHosts(content []string) error {
	file, err := os.OpenFile(hostsFile, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strings.Join(content, "\n"))
	return err
}

func updateSystemHosts() {
	files, err := getProfiles()
	if err != nil || len(files) == 0 {
		return
	}
	hosts := []string{}
	for _, file := range files {
		content, err := readProfile(file)
		if err != nil {
			continue
		}
		hosts = append(hosts, content...)
	}
	err = writeSystemHosts(hosts)
	if err != nil {
		log.Println("更新系统 hosts 失败")
	} else {
		log.Println("更新系统 hosts 成功")
	}
}

func watchWorkDir() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	if err = watcher.Add(flags.DataDir); err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				continue
			}
			switch event.Op {
			case fsnotify.Create, fsnotify.Write, fsnotify.Rename, fsnotify.Remove:
				if strings.HasSuffix(event.Name, ".local") || strings.HasSuffix(event.Name, ".remote") {
					NeedUpdateImmediately = true
				}
			}
		}
	}
}

func main() {
	ticker := time.NewTicker(flags.UpdateInterval)
	defer ticker.Stop()
	notify := time.NewTicker(1 * time.Second)
	defer notify.Stop()

	go watchWorkDir()
	updateSystemHosts()
	for {
		select {
		case <-ticker.C: // 定期更新
			updateSystemHosts()
		case <-notify.C: // 文件变更触发更新
			if NeedUpdateImmediately {
				NeedUpdateImmediately = false
				updateSystemHosts()
			}
		}
	}
}
