# 知乐 构建与对应源码

本版本基于 go-music-dl v1.1.3。2026-10-30 的品牌、界面、下载页和打包修改详见 git 历史与 NOTICE。
Go module/import 名称保留上游路径，以保持内部包兼容；公开发行仓库为 isbenny/zhile。

## Web / 本地预览

需要 Go 1.27。

```sh
go mod download
go build -o music-dl ./cmd/music-dl
./music-dl web --port 8080
```

浏览器打开控制台打印的地址并按应用提示登录。
`--desktop` 仅用于回环地址的桌面预览；公网服务不要使用该选项。
修改版应用页底部有上游、AGPL 与对应源码入口。

## Android

完整构建过程在 `.github/workflows/android.yml`。

1. Linux + Android NDK 27.0.12077973：为每个 ABI 运行 `.github/scripts/build_ffmpeg_android.sh`。
2. Windows + Go 1.27 + JDK 17 + Android SDK 33/build-tools 34 + NDK 27：把音频工具放入 `desktop_app/ffmpeg/android/<abi>/`。
3. 设置 `ANDROID_KEYSTORE_FILE`、`ANDROID_KEYSTORE_PASSWORD`、`MUSIC_DL_BUNDLE_ANDROID_FFMPEG=1`，运行 `package_app.bat`。
4. `package_app.bat` 通过本仓库 `tools/gio-cmd` 编译并签名，再注入 FFmpeg/FFprobe 并重新签名。最后运行 `.github/scripts/verify_android_ffmpeg_apk.ps1`。

公开源码不包含私有签名密钥。任何人都可以生成自己的密钥、改应用 ID 编译并安装衍生版。
包名 `io.github.isbenny.zhile`，应用名 知乐，Android 版本 0.1.0.1，产品版本 0.1.0。
最低 Android API 24（7.0）。更新同一应用须沿用原签名，应用版本号须递增。

## 依赖源码与许可

发行页提供 `zhile-source.zip`：本仓库指定提交、Go 依赖源码 zip 及 `.mod/.info` 文件、FFmpeg 8.0.1 源码、锁定版本和构建脚本。
`dependency-sources/manifest.json` 记录模块路径、版本、校验和与本地文件。模块目录结构保持 Go module proxy 的 `/<escaped module>/@v/` 格式，可作为 file:// 代理使用；两套 Go 模块均包含依赖。

```sh
# 在解压后的 zhile/ 中，路径请换成你自己的绝对路径
GOPROXY=file:///absolute/path/zhile/dependency-sources go mod download
```

FFmpeg 源码来自 https://ffmpeg.org/releases/ffmpeg-8.0.1.tar.xz；构建脚本固定 SHA256。
未启用 GPL/nonfree、外部库及网络协议，只在本地文件上工作。
支持常见音频元数据写入；不包含 x264/libmp3lame 等外部编码器，因此部分视频生成/转码功能不可用。
FFmpeg 适用 LGPL-2.1-or-later，原许可证保留在源码中。前端第三方许可证在 `internal/web/templates/static/vendor/licenses/`；Go 模块各自的许可证保留在源码归档中。

## 自己的服务器

将发布包 `zhile-website.zip` 解压，把里面的文件整体上传到网站目录即可。
网页默认按钮下载同目录 `downloads/` 的 APK、源码和校验文件，完全静态，不需要运行 Go 服务。
保留 `LICENSE.txt`、`NOTICE.txt`、源码下载及其入口。不要只上传 APK 而删除源码。
GitHub 中的 `website/dist/` 默认指向同版本 GitHub Release；服务器发布包会自动替换成本地相对链接。

校验 APK：对照 `SHA256SUMS.txt`；Mac 使用 `shasum -a 256 文件.apk`，Windows 使用 `Get-FileHash 文件.apk -Algorithm SHA256`。
