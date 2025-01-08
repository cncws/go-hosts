package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cncws/hosts-go/cmd/flags"
	"github.com/cncws/hosts-go/internal/op"
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
	files, err := op.CollectProfileFiles()
	if err != nil || len(files) == 0 {
		return
	}
	hosts := []string{}
	for _, file := range files {
		content, err := op.ReadProfile(file)
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
