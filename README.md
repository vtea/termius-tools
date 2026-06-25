# Termius Tools

用于在 macOS、Windows 和 Linux 上备份与恢复 [Termius](https://termius.com/) SSH 客户端数据的图形界面工具。

基于 jimmyalcala 的 CLI 工具 [termius-backup](https://github.com/jimmyalcala/termius-backup)。

## 功能

- **备份** — 将主机、SSH 密钥、代码片段、分组和设置导出为可移植的 `.tbk` 文件
- **恢复** — 从备份导入数据，并将加密密钥写回系统钥匙串
- **查看** — 在不执行恢复的情况下查看备份元数据与文件列表
- **状态面板** — 实时显示 Termius 运行状态与数据目录检测结果

## 环境要求

- [Go](https://go.dev/dl/) 1.22+
- [Wails v2](https://wails.io/docs/gettingstarted/installation) CLI

## 开发

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式运行
make dev
# 或
wails dev

# 构建生产版本
make build
# 或
wails build
```

构建产物位于 `build/bin/`。

## 安全说明

备份文件（`.tbk`）包含你的 Termius 数据**以及加密密钥**。请像保管密码导出文件一样妥善保管，使用后及时删除。

## 许可证

MIT
