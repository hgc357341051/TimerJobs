# 小胡定时任务系统（企业级）

一个高可用、可扩展、支持多种执行模式的企业级定时任务管理系统，适用于自动化运维、定时数据处理、批量任务调度等场景。

---

## 目录结构

```
jobs/
├── cmd/                # 命令行工具
├── config/             # 配置文件（支持多环境）
├── controller/         # 控制器（业务逻辑）
│   ├── admins/         # 管理员相关接口
│   ├── index/          # 首页及通用接口
│   └── jobs/           # 任务相关接口
├── core/               # 核心服务启动与守护
├── docs/               # Swagger/OpenAPI 文档
├── function/           # 公共函数库（如JWT、参数处理等）
├── global/             # 全局配置、日志、数据库、任务调度
├── middlewares/        # 中间件（认证、IP控制、CORS、限流等）
├── models/             # 数据模型
├── routers/            # 路由注册
├── runtime/            # 运行时日志、任务输出
├── scripts/            # 运维脚本
├── main.go             # 主程序入口
└── Dockerfile          # Docker部署文件
```

---

## 业务流程概览

1. **管理员登录** → 获取JWT → 访问API
2. **任务管理**：增删改查任务 → 配置执行模式（HTTP/命令/函数） → 定时调度
3. **任务执行**：按cron表达式自动触发 → 记录执行日志 → 支持手动触发/停止/重启
4. **日志管理**：系统日志、任务日志分离，支持查询与下载
5. **IP控制**：支持白名单、黑名单，灵活配置
6. **系统监控**：健康检查、任务状态、API文档自带

---

## 快速开始

### 环境要求
- Go 1.20+
- MySQL 5.7+/SQLite 3.x
- Windows/Linux/macOS

### 安装与运行
```bash
# 克隆项目
 git clone <repo-url>
 cd jobs
# 安装依赖
go mod tidy
# 编译
 go build -o main main.go
# 运行
 ./main
```

### 配置说明
- 配置文件位于 `config/config.yaml`，支持 MySQL/SQLite 切换、日志、IP控制等参数。
- 支持热更新：修改配置后可通过API或重启服务生效。

### 主要API入口
- Swagger文档：http://localhost:8080/swagger/index.html
- 健康检查：http://localhost:8080/jobs/health
- 任务状态：http://localhost:8080/jobs/jobStatus

---

## 主要功能与接口

### 管理员相关
- `/admin/login` 登录
- `/admin/register` 注册
- `/admin/profile` 获取/更新个人信息
- `/admin/list` 管理员列表
- `/admin/status` 修改状态
- `/admin/delete` 删除管理员

### 任务管理
- `/jobs/add` 新增任务
- `/jobs/edit` 编辑任务
- `/jobs/del` 删除任务
- `/jobs/list` 任务列表（分页）
- `/jobs/read` 查询任务详情
- `/jobs/run` 手动运行
- `/jobs/stop` 停止任务
- `/jobs/restart` 重启任务
- `/jobs/logs` 查询任务日志

### 日志与系统
- `/jobs/zapLogs` 系统日志
- `/jobs/health` 健康检查
- `/jobs/jobStatus` 任务调度状态

### IP控制
- `/jobs/ip-control/status` 查询IP控制状态
- `/jobs/ip-control/whitelist/add|remove` 白名单管理
- `/jobs/ip-control/blacklist/add|remove` 黑名单管理

---

## 业务开发规范

- **控制器**：所有业务逻辑集中在 `controller/`，每个模块独立。
- **模型**：数据结构定义在 `models/`，与数据库表结构一一对应。
- **中间件**：统一放在 `middlewares/`，如认证、IP控制、CORS、限流等。
- **全局变量与配置**：统一在 `global/`，包括日志、数据库、调度器等。
- **路由注册**：所有路由集中在 `routers/`，分模块注册。
- **公共函数**：如参数校验、JWT、通用响应等在 `function/`。
- **日志**：系统日志与任务日志分离，日志文件在 `runtime/`。
- **Swagger注释**：所有接口、结构体均需补全注释，自动生成API文档。

---

## 二次开发与扩展

1. **新增业务模块**：
   - 在 `controller/`、`models/`、`routers/` 下分别添加对应文件
   - 在 `routers/` 注册新模块路由
2. **自定义任务执行模式**：
   - 在 `global/jobFunc.go` 中实现新函数
   - 在任务配置中选择 `mode: func` 并指定函数名
3. **中间件扩展**：
   - 在 `middlewares/` 新增中间件并在 `core/run.go` 注册
4. **API文档维护**：
   - 按照 swagger 注释规范补全接口和结构体注释
   - 运行 `swag init` 自动生成文档

---

## 部署与运维

### Docker 部署
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
EXPOSE 8080
CMD ["./main"]
```

### Systemd 服务
```ini
[Unit]
Description=小胡定时任务系统
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/xiaohu/jobs
ExecStart=/opt/xiaohu/jobs/main start -d -f  # -d: 后台运行，-f: 前台日志输出
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## 常见问题与支持

- **数据库连接失败**：检查配置、服务状态、账号密码
- **任务执行失败**：检查cron表达式、命令/URL、日志
- **API认证失败**：检查JWT token格式与有效期
- **日志查看**：`runtime/` 目录下日志文件

---

## 贡献与联系方式

- Fork 项目，提交 PR
- 作者：小胡
- QQ：357341051
- 邮箱：357341051@qq.com

---

## License

MIT 