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

	"github.com/cncws/go-hosts/cmd/flags"
)

type profileReader func(string) ([]string, error)

var remoteHistorySuffix = ".history"
var readerMap = map[string]profileReader{
	".local":  readLocal,
	".remote": readRemote,
}

func SupportProfile(file string) bool {
	return readerMap[strings.ToLower(filepath.Ext(file))] != nil
}

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
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			if body, err := io.ReadAll(resp.Body); err == nil {
				writeRemoteHistory(file, body)
				return strings.Split(string(body), "\n"), nil
			}
		} else {
			err = fmt.Errorf("Status %d", resp.StatusCode)
		}
	}
	// 读取历史
	if lines, hisErr := readRemoteHistory(file); hisErr == nil {
		log.Printf("配置 %s 更新失败【%v】延用上一次配置", filepath.Base(file), err)
		lines = append(lines, "# 配置更新失败，延用上一次配置")
		return lines, nil
	}
	return nil, err
}

func ReadProfile(file string) ([]string, error) {
	reader, ok := readerMap[strings.ToLower(filepath.Ext(file))]
	if !ok {
		return nil, fmt.Errorf("不支持的文件类型 %s", filepath.Ext(file))
	}
	content, err := reader(file)
	lines := []string{"# profile begin: " + filepath.Base(file)}
	if err == nil {
		lines = append(lines, content...)
		log.Printf("配置 %s 加载成功\n", filepath.Base(file))
	} else {
		lines = append(lines, fmt.Sprintf("# 配置 %s 加载失败", filepath.Base(file)))
		log.Printf("配置 %s 加载失败【%v】\n", filepath.Base(file), err)
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
		if !info.IsDir() && SupportProfile(path) {
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
	log.Printf("加载配置 %v\n", filesname)
	return files, nil
}
