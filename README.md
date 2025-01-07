
## 初衷

SwitchHosts 占用了 macOS 一个珍贵的菜单栏图标。

## 功能描述

1. 程序需要一个配置目录如 `.hosts`，*配置文件*放在该目录的根目录下，放子文件夹下无效。
2. 配置文件为普通的文本文件，分两种格式：
   - 以 `.local` 结尾的文件内容是静态的 hosts，需要配置什么 hosts 自行编辑
   - 以 `.remote` 结尾的文件在第一行放入 URL，程序会定期使用 GET 请求获取 hosts 并更新
3. 更新时机
   - 配置文件被修改时（使用 `github.com/fsnotify/fsnotify` 监听文件修改事件）
   - 每隔一段时间（如 1h）

## 守护进程

macOS 编辑 `~/Library/LaunchAgents/hosts-go.plist` 添加以下内容（替换 `/path/to` 为实际路径）。

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>Label</key>
		<string>com.example.hosts-go</string>
		<key>ProgramArguments</key>
		<array>
			<string>/path/to/hosts-go</string>
		</array>
		<key>RunAtLoad</key>
		<true/>
		<key>StandardOutPath</key>
		<string>/path/to/.hosts/hosts-go.log</string>
		<key>StandardErrorPath</key>
		<string>/path/to/.hosts/hosts-go.log</string>
	</dict>
</plist>
```

然后执行 `launchctl load ~/Library/LaunchAgents/hosts-go.plist` 加载配置，之后可以使用 `launchctl unload ~/Library/LaunchAgents/hosts-go.plist` 来卸载配置。

windows 可以使用 [nssm](https://nssm.cc/download) 将 `hosts-go.exe` 配置为服务：

1. 下载 nssm 解压到某个文件夹
2. 在 powershell 中切换到 nssm 所在目录下再执行 `.\nssm.exe install hosts-go`
3. 在弹出的窗口中填写 `hosts-go.exe` 路径，可选配置服务的标题、描述、日志路径等，点击 `Install Service` 安装
4. 在服务中找到 `hosts-go` 服务，启动服务
5. 之后可以通过 `.\nssm.exe remove hosts-go` 移除服务
