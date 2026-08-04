# VoHive QDC507 绿联 NAS Docker 部署项目

这是一个面向绿联 NAS 和其他 amd64 Linux 主机的 VoHive 部署项目，专门整理了百望 EG25G-QDC507 类 USB 4G 模块在 Docker 中运行时需要的镜像、配置、驱动绑定脚本、热插拔恢复逻辑和中文使用说明。

项目的目标是：**把模块插在 NAS 上，导入镜像，创建项目，打开网页即可管理；模块拔出再插回后，容器自动尝试恢复，不需要反复重建项目。**

## 项目能做什么

- 在 Docker 中运行 VoHive 后台和网页管理界面。
- 识别百望 EG25G-QDC507 的 USB 标识 `2ca3:4006`。
- 自动加载或绑定 `qmi_wwan`、`cdc_wdm`、`option` 等 Linux 驱动。
- 映射 `/dev`、`/sys` 和 `/lib/modules`，让 USB 重新枚举后生成的新设备节点继续可见。
- 定时检查 QMI 控制通道；模块重插、NAS 启动较慢或设备节点变化时，自动尝试重新绑定。
- 提供可选的绿联“快速访问”反向代理示例，不改变主容器访问 USB/QMI 所需的主机网络。
- 提供可直接导入绿联 Docker 的 amd64 镜像发布包。

## 适合谁使用

本项目适合已经拥有 QDC507 类模块、绿联 NAS 或 amd64 Linux 主机，并希望在 NAS 上运行 VoHive 的用户。使用者需要能够管理 NAS 的 Docker 应用，并接受特权容器需要访问 USB 设备和主机驱动的事实。

## 最快部署方式

普通绿联 NAS 不需要编译源码，按下面四步即可：

1. 打开 [v0.2.2 发布页](https://github.com/hei85/vohive-qdc507-nas-docker/releases/tag/v0.2.2)。
2. 下载 `vohive-qdc507-0.2.2-amd64.tar`，在绿联 Docker 的“镜像 → 导入”中导入。
3. 下载同一发布页中的 `compose.import.yml`，在“项目 → 创建项目”中导入；如果界面没有文件导入按钮，就把文件内容粘贴到项目编辑框。
4. 项目显示运行后，在浏览器打开 `http://NAS_IP:7575`。首次登录账号是 `admin`，密码是 `vohive`；登录后立即修改密码。

完整的文件对照、点击路径、设备检查和热插拔测试见 [中文逐步部署指南](docs/使用指南.md)。

## 文件用途对照

| 文件 | 用途 | 普通绿联部署是否需要 |
|---|---|---|
| `vohive-qdc507-0.2.2-amd64.tar` | 已经构建好的 Docker 镜像 | 需要，导入“镜像” |
| `compose.import.yml` | 使用已导入镜像创建项目 | 需要，创建“项目” |
| `SHA256SUMS.txt` | 检查镜像下载是否完整 | 可选，不导入 |
| `compose.yml` | 从源码构建镜像 | 不需要 |
| `compose.ugreenlink.yml` | 可选“快速访问”代理 | 只有需要时才使用 |
| `Dockerfile` | 镜像构建规则 | 不需要逐个导入 |
| `docker-entrypoint.sh` | 启动、驱动绑定和热插拔恢复脚本 | 已经打包进镜像 |
| `scripts/verify-qmi.sh` | 检查 QMI 设备和驱动 | 已经打包进镜像 |
| `third_party/` | 可分发的上游源码快照 | 源码构建时使用 |
| `docs/使用指南.md` | 中文详细说明 | 建议先阅读 |

最容易混淆的地方是：**TAR 导入“镜像”，`compose.import.yml` 创建“项目”；两者不是同一种导入。**

## 运行原理

主容器使用 `network_mode: host`（主机网络），因为 QMI 网络设备和 NAS 主机的 USB 驱动属于主机侧资源。Compose 配置还保留以下映射：

- `/dev:/dev`：让容器看到 USB 串口和 QMI 控制节点。
- `/sys:/sys:rw`：允许恢复脚本重新绑定指定的 QDC507 USB 接口。
- `/lib/modules:/lib/modules:ro`：读取主机已有的内核模块。
- `runtime/data`：保存 VoHive 运行配置。
- `runtime/logs`：保存日志和驱动恢复记录。

容器不会替 NAS 主机安装缺失的内核驱动。如果主机内核没有 `qmi_wwan`、`cdc_wdm` 或 `option`，需要先处理 NAS 内核支持问题。

## 设备识别与热插拔

容器启动时会尝试加载 QMI 相关驱动，并针对 QDC507 的 USB 接口进行绑定。随后每隔约 5 秒检查一次设备状态。模块拔出再插回同一个 USB 口后，通常等待 10～30 秒即可恢复；不同 NAS 的供电、USB 控制器和内核行为可能导致时间不同。

进入容器终端执行下面的检查：

```sh
verify-qmi
```

正常时至少应该能看到 `/dev/cdc-wdm0` 或其他编号的 `cdc-wdm` 节点，同时能看到 `qmi_wwan`、`cdc_wdm` 和 `option` 驱动。

## 远程访问

主容器必须保留主机网络。部分绿联版本的“快速访问”只代理桥接网络容器，因此仓库提供了一个可选的 `compose.ugreenlink.yml`，用 Caddy 在 NAS 的 `37953` 端口代理到主服务的 `7575` 端口。

局域网使用不需要这个可选代理。无论使用哪种方式，都不要把管理后台直接暴露到公网；请使用强密码和可信的远程访问方式。

## 从源码构建

如果使用者有一台具备 Docker 构建能力的 Linux 主机，可以执行：

```sh
docker compose -f compose.yml up -d --build
```

构建完成后，可执行下面的脚本导出镜像，再将生成的 TAR 导入绿联 NAS：

```sh
./scripts/export-image.sh vohive-qdc507-0.2.2-amd64.tar
```

普通绿联 NAS 直接使用发布页的预构建镜像即可，不需要执行源码构建。

## 发布新版本（维护者）

镜像压缩包由 GitHub Actions 自动构建。打一个 `v` 开头的 tag 并推送，会自动构建 amd64 镜像、导出 TAR、计算 SHA256，并发布到该 tag 对应的 GitHub Release：

```sh
git tag v0.2.2
git push origin v0.2.2
```

Release 资产里会出现 `vohive-qdc507-0.2.2-amd64.tar` 和 `SHA256SUMS.txt`。tag 名会去掉前导 `v` 作为镜像版本号（`v0.2.2` → `0.2.2`），所以导入后镜像名是 `local/vohive-qdc507:0.2.2`。发版前记得同步更新 `compose.import.yml`、`scripts/export-image.sh` 和 `Dockerfile` 里的版本号，避免 tar 与 compose 引用对不上。

需要在不发版的情况下测试构建时，去 Actions → “Build Docker release image” → Run workflow 手动触发；手动触发的产物只作为 Actions 构件保留 30 天，不会发布到 Release。

## 安全、隐私和许可证

- 发布包不包含任何个人 IP、账号密码、IMEI、日志、SIM 配置或 NAS 运行数据。
- 发布包不包含 DJI、百望或 Quectel 模块固件；本项目运行 VoHive 不需要给模块刷写固件。
- 主容器使用特权模式并挂载主机设备，只应在自己完全信任的 NAS 上运行。
- VoHive 源码保留原始 PolyForm 非商业许可证；VoWiFi Go、SWU Go 和本项目脚本/文档的许可证见 `NOTICE.md`、`LICENSE` 和各源码目录中的许可证文件。
- 本项目与 DJI、百望、绿联、VoHive 原作者没有官方隶属关系，也不保证适配所有 NAS 内核或所有运营商网络。
- 使用者应遵守飞行安全、蜂窝通信、频谱、SIM 服务条款和当地法律，不得将本项目用于规避设备限制或进行未授权通信。

## 项目状态

v0.2.2 修复大疆一代 4G 模块（BAIWANG EG25G-QDC507，USB ID `2ca3:4006`）在多个接口都被 `qmi_wwan` 绑定时错误选择 `.1/cdc-wdm0` 的问题；设备发现现优先使用已验证可用的 `.4/cdc-wdm3`。

v0.2.1 在 v0.2.0 基础上修复了免责声明每次登录重复弹窗的问题（改为服务端持久化：任意设备首次同意后，所有设备不再重复提示），并新增打 `v` tag 自动构建并发 GitHub Release 的工作流；amd64 镜像构建、归档导入与网页启动流程不变。硬件识别与热插拔恢复仍取决于具体 NAS 的 USB 供电、内核驱动和模块状态；遇到设备未识别时，请先按中文指南执行 `verify-qmi` 并查看容器日志。
