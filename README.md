# VoHive QDC507 NAS Docker Toolkit

> Unofficial deployment helpers for using a Baiwang EG25G-QDC507-based DJI 4G module with VoHive on an amd64 Linux or UGREEN NAS host.

这个仓库提供 Docker、QMI 驱动绑定和热插拔恢复脚本。它不包含 VoHive 可执行程序、DJI/百望固件、SIM 配置、设备标识、账号密码、日志或 NAS 配置。

## 功能

- 针对 `2ca3:4006`（BAIWANG EG25G-QDC507）注册 `qmi_wwan` 与 `option` 驱动。
- 保留 `/dev`、`/sys` 和内核模块映射，使 USB 热插拔后生成的新设备节点可被容器使用。
- 每 5 秒检查一次 QMI 控制节点；模块重插或 NAS 启动较慢时，会自动将 QMI 接口重新绑定，无需手动重启项目。
- 提供一个可选的 Caddy 反向代理示例，用于让 UGREEN NAS 的 Quick Access 访问后台，而不改变主容器的硬件网络模式。

## 不包含什么

VoHive 本体与任何 DJI/百望固件均不属于本仓库。请自行确认来源、许可和当地适用规则，并将**合法取得的 Linux amd64 VoHive 可执行文件**放到 `vendor/vohive` 后再构建。本仓库的 MIT 许可证只覆盖本仓库原创的部署脚本和文档。

本仓库不会修改模块固件、IMEI 或序列号；它只在 NAS 主机上重新绑定 Linux USB 网络驱动。

## 前提条件

- amd64 Linux/UGREEN NAS 主机，内核包含 `qmi_wwan`、`cdc_wdm` 与 `option` 模块。
- Docker Compose v2 或兼容的 NAS Docker 项目功能。
- 模块直接插在 NAS 的 USB 口。若保留虚拟机备份，请确保虚拟机已关机，避免 USB 被两个系统同时占用。

## 构建与部署

1. 复制 `config.example.yaml` 为本地 `config.yaml`，设置强密码；不要提交该文件。
2. 将合法取得的应用程序放至 `vendor/vohive` 并赋予可执行权限。
3. 执行 `docker compose -f compose.yml up -d --build`。
4. 打开 `http://NAS_IP:7575`，使用你在本地配置中设置的账号登录。

UGREEN NAS 不能直接在项目中构建镜像时，可在另一台 Linux 主机执行：

```sh
./scripts/export-image.sh vohive-qdc507-image.tar
```

然后从 NAS Docker 的“镜像导入”导入该 TAR；把 `compose.yml` 中的 `build: .` 移除后，保留 `image: local/vohive-qdc507:local` 再部署。

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

## 安全与免责声明

该项目以特权容器方式挂载 `/dev` 和可写 `/sys`，仅应在你完全信任的个人 NAS 上运行。不要把此容器或后台直接暴露给互联网。该项目与 DJI、BAIWANG、UGREEN 和 VoHive 作者没有隶属关系或官方支持关系。

用户应自行遵守飞行安全、蜂窝通信、频谱、SIM 卡服务条款和当地法律。请勿将此工具用于规避设备限制、监管要求或未授权通信。
