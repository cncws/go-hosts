package op

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cncws/hosts-go/cmd/flags"
)

var remoteHistorySuffix = ".history"

func readLocal(file string) ([]string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

func writeRemoteHistory(file string, data []byte) error {
	return os.WriteFile(file+remoteHistorySuffix, data, 0644)
}

func readRemoteHistory(file string) ([]string, error) {
	return readLocal(file + remoteHistorySuffix)
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
		lines, err := readRemoteHistory(file)
		if err != nil {
			return []string{fmt.Sprintf("# 读取远程配置失败，status code: %d", resp.StatusCode)}, nil
		}
		lines = append(lines, "# 读取远程配置失败，延用上一次配置")
		return lines, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	writeRemoteHistory(file, body)
	return strings.Split(string(body), "\n"), nil
}

func ReadProfile(file string) ([]string, error) {
	lines := []string{"# profile begin: " + filepath.Base(file)}
	var content []string
	var err error
	switch filepath.Ext(file) {
	case ".local":
		if content, err = readLocal(file); err == nil {
			lines = append(lines, content...)
			log.Printf("本地配置 %s 已读取\n", filepath.Base(file))
		}
	case ".remote":
		if content, err = readRemote(file); err == nil {
			lines = append(lines, content...)
			log.Printf("远程配置 %s 已读取\n", filepath.Base(file))
		}
	}
	if err != nil {
		lines = append(lines, "# 读取配置失败："+err.Error())
	}
	lines = append(lines, "# profile end, update at "+time.Now().Format(time.RFC3339), "")
	return lines, nil
}

func CollectProfileFiles() ([]string, error) {
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
