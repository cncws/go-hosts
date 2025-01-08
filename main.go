package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cncws/hosts-go/cmd/flags"
	"github.com/cncws/hosts-go/internal/op"
	"github.com/fsnotify/fsnotify"
)

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

func main() {
	var task = op.NewUpdateHostTask()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	if err = watcher.Add(flags.DataDir); err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(flags.UpdateInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			// 触发机制1: 文件变动
			case event, ok := <-watcher.Events:
				if !ok {
					continue
				}
				switch event.Op {
				case fsnotify.Create, fsnotify.Write, fsnotify.Rename, fsnotify.Remove:
					if op.SupportProfile(event.Name) {
						task.UpdateImmediately()
					}
				}
			// 触发机制2: 定时
			case <-ticker.C:
				task.UpdateImmediately()
			}
		}
	}()

	task.Start()
}
