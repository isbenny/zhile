<p align="center"><img src="brand/icon.png" width="112" alt="知乐 circular wave-Z logo"></p>
<h1 align="center">知乐 · ZHILE</h1>
<p align="center">听见喜欢，乐在其中。</p>

黑绿配色的开源音乐工具，支持多来源搜索、在线试听、收藏与下载。
基于 [go-music-dl](https://github.com/guohuiyuan/go-music-dl) v1.1.3 二次开发，保留 AGPL-3.0 许可和原作者署名。

- [安卓安装包与对应源码](https://github.com/isbenny/zhile/releases)
- 下载页源码：`website/dist/`（可以直接上传自己的静态服务器）
- [构建与源码说明](docs/BUILD.md)
- [修改与第三方声明](NOTICE) · [许可证](LICENSE) · [上游使用文档](docs/UPSTREAM-README.md)

## Android

大多数手机选择 `zhile-arm64.apk`，旧设备可选 `zhile-universal.apk`。
Android 7.0 及以上；系统 WebView 需保持更新。浏览器和构建检查通过不代表所有安卓设备已实测。

首次安装按系统提示允许当前浏览器安装此应用。不要关闭系统的整体安全防护。
这是独立应用 `io.github.isbenny.zhile`，与原版 MusicDL 可以同时安装。

音频工具由 FFmpeg 8.0.1 源码构建，支持原音频复制及常见元数据写入。
不包含 x264、libmp3lame 等外部编码器；部分高级视频生成/转码功能不受支持。

## 许可与来源

可以修改、再分发及商业使用代码，但发布衍生版本须遵守 AGPL-3.0、保留版权与许可证、标注修改并提供对应源码；修改版通过网络供用户交互时也须提供源码入口。
本仓库的独立名称不改变上游代码的版权归属。音乐内容权利另行适用。

## 当前版本与使用边界

v0.2.1 为正常功能预览版，保留在线搜索、推荐、播放、换源和下载，以及本地音乐、歌单管理。第三方来源可能失效，软件许可不代表内容授权。仅访问和保存依法有权使用的内容。

iPhone／iPad 可连接个人 Web 实例，通过 Safari 添加到主屏幕；当前没有原生 iOS 安装包。见 [iOS 使用说明](website/dist/ios.html)。网站静态下载页不会运行音乐服务。

演示内容仅供学习与技术交流；“侵删”不代替授权，也不额外限制 AGPL 源码权利。版权及隐私联系：csuyangxing@gmail.com。

[发布前风险核验](docs/LEGAL_REVIEW.md) · [大陆服务器部署与统计](docs/DEPLOYMENT.md)

## v0.2.0 界面与发现

新增首次进入的场景入口、本地音乐入口、首页/搜索/我的音乐底部导航，以及大封面播放详情。歌词默认收起，点迷你播放器歌名可展开详情。下载页提供可暂停动效。

[运营节奏与 AI 方向](docs/OPERATIONS.md)（AI 尚未接入）。
