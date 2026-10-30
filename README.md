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
