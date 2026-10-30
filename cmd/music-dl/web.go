package main

import (
	"fmt"
	"github.com/guohuiyuan/go-music-dl/internal/web"
	"github.com/spf13/cobra"
)

var port string
var noBrowser bool
var desktopMode bool
var basePath string
var listenHost string
var privateMode bool
var secureCookies bool

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "启动 Web 服务模式",
	RunE: func(cmd *cobra.Command, args []string) error {
		if desktopMode {
			if privateMode || secureCookies {
				return fmt.Errorf("--desktop cannot be combined with --private or --secure-cookies")
			}
			web.StartDesktop(port)
			return nil
		}
		web.StartWithOptions(port, web.StartOptions{ShouldOpenBrowser: !noBrowser, BasePath: basePath, ListenHost: listenHost, RequireAuthForAll: privateMode, SecureCookies: secureCookies})
		return nil
	},
}

func init() {
	webCmd.Flags().StringVarP(&port, "port", "p", "8080", "服务端口")
	webCmd.Flags().StringVar(&basePath, "base-path", web.DefaultRoutePrefix, "Web 端基础路径")
	webCmd.Flags().BoolVar(&noBrowser, "no-browser", false, "不自动打开浏览器")
	webCmd.Flags().BoolVar(&desktopMode, "desktop", false, "桌面内嵌模式")
	webCmd.Flags().StringVar(&listenHost, "host", "", "监听地址；反向代理部署请使用 127.0.0.1")
	webCmd.Flags().BoolVar(&privateMode, "private", false, "个人私有模式：所有音乐与数据接口均需登录")
	webCmd.Flags().BoolVar(&secureCookies, "secure-cookies", false, "仅通过 HTTPS 发送登录 Cookie（用于 HTTPS 反向代理）")
	_ = webCmd.Flags().MarkHidden("desktop")
	rootCmd.AddCommand(webCmd)
}
