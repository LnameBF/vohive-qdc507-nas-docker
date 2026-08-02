# VoHive QDC507 NAS Docker Toolkit

> Unofficial deployment helpers for using a Baiwang EG25G-QDC507-based DJI 4G module with VoHive on an amd64 Linux or UGREEN NAS host.

这个仓库提供完整的非商业源码构建、Docker 镜像导入、QMI 驱动绑定和热插拔恢复脚本。Release 会提供可直接导入 NAS Docker 的 amd64 镜像；SIM 配置、设备标识、账号密码、日志和 NAS 配置均不包含在仓库中。

## 功能

- 针对 `2ca3:4006`（BAIWANG EG25G-QDC507）注册 `qmi_wwan` 与 `option` 驱动。
- 保留 `/dev`、`/sys` 和内核模块映射，使 USB 热插拔后生成的新设备节点可被容器使用。
- 每 5 秒检查一次 QMI 控制节点；模块重插或 NAS 启动较慢时，会自动将 QMI 接口重新绑定，无需手动重启项目。
- 提供一个可选的 Caddy 反向代理示例，用于让 UGREEN NAS 的 Quick Access 访问后台，而不改变主容器的硬件网络模式。

## 许可与边界

仓库内含的 VoHive 与 VoWiFi Go 源码均保留原始许可证和署名，详情见 [NOTICE.md](NOTICE.md)。VoHive 部分仅限非商业用途。本仓库的原创部署脚本和文档使用 MIT 许可证，见 `LICENSES/MIT.txt`。

本仓库不包含也不需要 DJI/百望/Quectel 的模块固件。它不会修改模块固件、IMEI 或序列号；只在 NAS 主机上重新绑定 Linux USB 网络驱动。

## 前提条件

- amd64 Linux/UGREEN NAS 主机，内核包含 `qmi_wwan`、`cdc_wdm` 与 `option` 模块。
- Docker Compose v2 或兼容的 NAS Docker 项目功能。
- 模块直接插在 NAS 的 USB 口。若保留虚拟机备份，请确保虚拟机已关机，避免 USB 被两个系统同时占用。

## 构建与部署

最简单的绿联 NAS 流程是：

1. 从 [v0.2.0 Release](https://github.com/hei85/vohive-qdc507-nas-docker/releases/tag/v0.2.0) 下载 `vohive-qdc507-0.2.0-amd64.tar`。
2. 在 Docker → 镜像 → 导入中导入这个 TAR。
3. 在 Docker → 项目中新建项目，使用仓库中的 `compose.import.yml`。
4. 打开 `http://NAS_IP:7575`；首次登录为 `admin` / `vohive`，登录后立即修改密码。

不要把 TAR 粘贴到项目编辑框，也不要在导入现成镜像时使用 `compose.yml`。文件用途和逐屏操作见 [docs/使用指南.md](docs/使用指南.md)。

UGREEN NAS 不能直接在项目中构建镜像时，可在另一台 Linux 主机执行：

```sh
./scripts/export-image.sh vohive-qdc507-image.tar
```

然后从 NAS Docker 的“镜像导入”导入该 TAR，并用 `compose.import.yml` 创建项目。

## UGREENlink / Quick Access（可选）

主服务需要 `network_mode: host` 才能可靠访问 USB/QMI 硬件。许多 NAS 的“快速访问”只代理 bridge 网络容器，因此可单独部署：

```sh
docker compose -f compose.ugreenlink.yml up -d
```

它在 NAS 的 `37953` 端口提供反向代理；再从 UGREEN Docker 的容器快速访问入口打开该代理。不要把管理后台直接暴露到公网；务必设置强密码，并仅通过可信的远程访问方式使用。

## 验证与故障排查

在主容器中运行：

```sh
verify-qmi
```

正常时应看到至少一个 `/dev/cdc-wdm*` 节点，以及 `qmi_wwan`、`cdc_wdm`、`option` 内核模块。若没有 `cdc-wdm`，检查主机内核是否提供所需驱动；容器本身不能补齐主机缺失的内核模块。

热插拔测试：保持容器运行，拔掉模块，等待约 10 秒后插回原 USB 口；最多约 30 秒应恢复。若超过一分钟仍未恢复，先检查模块供电和主机 USB 枚举情况，再查看 `runtime/logs/driver-init.log`。

完整操作步骤见 [docs/使用指南.md](docs/使用指南.md)。

## 安全与免责声明

该项目以特权容器方式挂载 `/dev` 和可写 `/sys`，仅应在你完全信任的个人 NAS 上运行。不要把此容器或后台直接暴露给互联网。该项目与 DJI、BAIWANG、UGREEN 和 VoHive 作者没有隶属关系或官方支持关系。

用户应自行遵守飞行安全、蜂窝通信、频谱、SIM 卡服务条款和当地法律。请勿将此工具用于规避设备限制、监管要求或未授权通信。
