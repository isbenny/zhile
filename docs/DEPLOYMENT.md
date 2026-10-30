# music.cnxwkj.com：部署与运营

2026-10-30：用户确认打包与发布 v0.2.1 正常功能版。网站提供静态下载与 iOS 连接个人 Web 实例的说明；不是共享音乐服务。第三方内容使用权须另行确认。

## 宝塔 / 1Panel

1. 在面板新建 `music.cnxwkj.com` 静态网站，配置域名解析与 HTTPS。大陆主机上线前，通过接入商核实并办理网站 ICP 备案；App 按实际服务办理 APP 备案。备案不等于获得音乐版权授权。
2. 用户确认发布、完成发布前核验后，再构建签名 APK 和对应源码。运行打包脚本生成的网站含 `index.html`、`about.html`、`legal.html`、`ios.html`、`assets/`、`downloads/`、`update.json`、许可证与声明。
3. 上传网站内容；不要把源码仓库、SQLite 数据库、Cookie、签名密钥或配置目录直接作为网站根目录。不要反向代理个人应用的 `/music` 接口。网站只分发审核过的安装包和源码，App 的音乐与歌单留在用户设备中。
4. 先确认安装包和源码完整可下载，最后上传 `update.json`。固定 URL 不使用长期缓存；建议文件先上传为临时名，再原子替换。域名及 CDN 都应对 `update.json` 和固定名称 APK 使用 `Cache-Control: no-cache`。
5. 用真实安卓设备验证首次安装、覆盖升级、导入本地文件、播放及收藏，检查权限和实际联网行为后再向公众推广。

## 更新方式

Android 公开版主动点击“检查更新”后，优先读取 `https://music.cnxwkj.com/update.json`；该文件不可用或不合法时回退 GitHub。自动检查遵循应用设置。它不会静默安装，用户仍需通过 Android 系统确认安装。

每次发布保持应用 ID `io.github.isbenny.zhile` 和同一签名密钥，递增版本及 Android versionCode。网站 HTML 更新不会自动替换 APK 内置界面，App 功能改动需要发新 APK。

`scripts/update_manifest.py` 从真实安装包计算大小、SHA256，并检查对应源码是否在场。`scripts/package_release.py` 会调用它，并生成配套部署文件。Feed 中 SHA256 可供运维校验；当前客户端只打开下载链接，**没有实现下载后的 SHA256 自动验证**。安装签名校验由系统执行。

旧版 App 仍依赖 GitHub 更新渠道，未来发布时应同时维护 GitHub Release。已经下载出去的旧 APK 不会因为网站修改而自动失去原来的功能，也无法凭静态网站撤回。

## 下载统计

`deployment/nginx-download-stats.conf.example` 是可审阅配置示例。`log_format` 和 `map` 必须放在 Nginx/OpenResty 的 `http` 段，站点里只放对应 `access_log`。请按面板实际路径调整，使用面板检查配置通过后再重载，避免把完整示例直接粘入站点配置。

统计日志只记录时间、APK 路径、状态和字节，不记录 IP、Cookie、查询参数、User-Agent 或指纹。已有面板/CDN 安全日志是另一套日志，需另行确认用途、保留期和访问权限，不能据此宣称整个网站“不收集 IP”。下载统计计划按日轮转保留 30 天；示例轮转配置中的用户组、路径和日志重开命令需要面板适配。

查看汇总：

```sh
python3 scripts/download_stats.py /实际日志路径/zhile-downloads.jsonl
```

显示的是 **200/206 成功响应数与传输流量**。断点续传、重复点击和重试会增加响应数，它不等于完整下载人数、独立用户数、安装量或活跃用户数。不要把这些数对外标为“用户数”。目前没有启用线上日志、行为追踪或第三方统计 SDK。

## 用户管理

现阶段不建立云账户；音乐、收藏和历史在设备上。既有 Web 管理登录不能当作多用户系统，它没有逐用户隔离音乐库。静态下载网站无法管理每个安装者，也不能远程禁用其离线文件。

以后若确有云同步或授权服务需求，再单独设计账户、身份验证、数据隔离、注销、隐私告知和权限。不要先收集手机号或身份证来“统计用户”。AI 服务也应在功能与数据处理范围明确后另行接入。

## 依据与发布前事项

- [工信部 APP 备案解读](https://www.miit.gov.cn/jgsj/xgj/hlwgl/art/2023/art_564bf0759d7e41d5b4aa8ce4996b9e84.html)
- [Android 发布](https://developer.android.com/studio/publish)、[应用更新](https://developer.android.com/google/play/app-updates)
- [Nginx 日志模块](https://nginx.org/en/docs/http/ngx_http_log_module.html)
- 详见 [LEGAL_REVIEW.md](LEGAL_REVIEW.md)。示例配置未部署，不代表备案、授权、隐私或法律审查已经完成。
