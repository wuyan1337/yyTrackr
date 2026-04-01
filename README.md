# yyTrackr

`yyTrackr` 是一个自部署的订阅追踪工具，当前版本基于 Go + Gin + SQLite + HTMX，适合个人长期记录和管理各类订阅项目。

当前项目特点：

- Go + Gin 后端
- SQLite 本地数据存储
- HTMX + 服务端模板页面
- 支持账号注册、登录和多用户隔离
- 支持 Dashboard / Subscriptions / Analytics / Calendar / Settings
- 支持 Email / Telegram / Pushover / Webhook 通知
- 支持 iCal 导出、CSV / JSON 导出
- 适合跑在单机 VPS 或个人服务器上

## 当前运行方式

默认运行端口：

```text
8080
```

本地开发：

```bash
go mod tidy
go run .
```

生产环境建议：

- 使用 systemd 保活
- 使用反向代理提供 HTTPS
- 定期备份 `data/` 目录

## 配置

常用环境变量：

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `PORT` | 服务端口 | `8080` |
| `DATABASE_PATH` | SQLite 数据库路径 | `./data/subtrackr.db` |
| `DB_PATH` | `DATABASE_PATH` 的别名 | `./data/subtrackr.db` |
| `GIN_MODE` | Gin 模式 | `debug` |
| `DEV_PREVIEW` | 是否启用开发预览 | `false` |

示例：

```bash
PORT=8080
DATABASE_PATH=./data/subtrackr.db
GIN_MODE=release
DEV_PREVIEW=false
```

## 目录结构

```text
.
├─ internal/
├─ templates/
├─ web/
├─ data/
├─ main.go
├─ go.mod
└─ README.md
```

## 生产部署建议

- 先编译：

```bash
go build -o ./bin/subtrackr .
```

- 再通过 systemd 启动
- 反向代理指向本地 `127.0.0.1:8080`
- 如果用 HTTPS，推荐由 Caddy / Nginx 处理证书

## 说明

本仓库是当前可运行版本的代码仓库，README 以当前实际可部署状态为准，不附带演示截图。
