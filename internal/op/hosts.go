package op

import (
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type updateHostTask struct {
	hostsFile  string
	needUpdate bool
}

func NewUpdateHostTask() *updateHostTask {
	hostsFile := "/etc/hosts"
	if runtime.GOOS == "windows" {
		hostsFile = "C:/Windows/System32/drivers/etc/hosts"
	}
	return &updateHostTask{
		hostsFile:  hostsFile,
		needUpdate: true,
	}
}

func (t *updateHostTask) UpdateImmediately() {
	t.needUpdate = true
}

func (t *updateHostTask) writeSystemHosts(content []string) error {
	file, err := os.OpenFile(t.hostsFile, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strings.Join(content, "\n"))
	return err
}

func (t *updateHostTask) updateSystemHosts() {
	files, err := CollectProfileFiles()
	if err != nil || len(files) == 0 {
		return
	}
	hosts := []string{}
	for _, file := range files {
		content, err := ReadProfile(file)
		if err != nil {
			continue
		}
		hosts = append(hosts, content...)
	}
	err = t.writeSystemHosts(hosts)
	if err != nil {
		log.Println("更新系统 hosts 失败")
	} else {
		log.Println("更新系统 hosts 成功")
	}
}

func (t *updateHostTask) Start() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			if t.needUpdate {
				t.needUpdate = false
				t.updateSystemHosts()
			}
		}
	}
}
