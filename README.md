# 小胡定时任务系统（企业级）

一个高可用、可扩展、支持多种执行模式的企业级定时任务管理系统，适用于自动化运维、定时数据处理、批量任务调度等场景。

---

## 目录结构

```
jobs/
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
 git clone https://github.com/hgc357341051/TimerJobs.git
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
- Swagger文档：http://127.0.0.1:36363/swagger/index.html
- 健康检查：http://127.0.0.1:36363/jobs/health
- 任务状态：http://127.0.0.1:36363/jobs/jobStatus

---

## 主要功能与接口

### 任务管理

#### 创建任务 API (`POST /jobs/add`)

系统支持三种执行模式：**HTTP请求**、**系统命令**、**内置函数**。每种模式都有不同的参数配置。

##### 通用参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| `name` | string | 是 | 任务名称，唯一标识 | `"数据备份任务"` |
| `desc` | string | 否 | 任务描述 | `"每日凌晨备份数据库"` |
| `cron_expr` | string | 是 | Cron表达式，定义执行时间 | `"0 2 * * *"` |
| `mode` | string | 是 | 执行模式：`http`/`command`/`func` | `"http"` |
| `command` | string | 是 | 执行内容（根据mode不同而不同） | 见下方详细说明 |
| `state` | int | 否 | 任务状态：0=等待，1=执行中，2=停止 | `0` |
| `allow_mode` | int | 否 | 执行模式：0=并行，1=串行，2=立即执行 | `0` |
| `max_run_count` | int | 否 | 最大执行次数，0=无限制 | `0` |

##### 1. HTTP 模式 (`mode: "http"`)

用于调用外部 HTTP API 接口。

**command 格式说明：**
```
【url】URL地址
【mode】请求方式
【headers】请求头1:值1|||请求头2:值2
【data】POST数据
【cookies】Cookie字符串
【proxy】代理地址
【times】执行次数
【result】自定义结果判断字符串
```

**详细示例：**

1. **简单GET请求**
```json
{
  "name": "健康检查",
  "desc": "检查服务健康状态",
  "cron_expr": "0 */2 * * * *",
  "mode": "http",
  "command": "【url】https://api.example.com/health\n【mode】GET"
}
```

2. **POST请求带JSON数据**
```json
{
  "name": "数据同步",
  "desc": "同步用户数据",
  "cron_expr": "0 0 2 * * *",
  "mode": "http",
  "command": "【url】https://api.example.com/sync\n【mode】POST\n【headers】Content-Type:application/json\n【data】{\"action\":\"sync\",\"timestamp\":\"2024-01-01\"}"
}
```


4. **使用代理的请求**
```json
{
  "name": "代理请求",
  "desc": "通过代理访问API",
  "cron_expr": "0 */5 * * * *",
  "mode": "http",
  "command": "【url】https://api.example.com/data\n【mode】GET\n【proxy】http://proxy.example.com:8080"
}
```

5. **带Cookie的请求**
```json
{
  "name": "会话请求",
  "desc": "保持会话的API调用",
  "cron_expr": "0 0 */1 * * *",
  "mode": "http",
  "command": "【url】https://api.example.com/user/profile\n【mode】GET\n【cookies】sessionid=abc123; userid=456"
}
```

**配置参数说明：**

| 参数 | 说明 | 示例 |
|------|------|------|
| `【url】` | 请求的URL地址（必填） | `【url】https://api.example.com/endpoint` |
| `【mode】` | 请求方式，默认GET | `【mode】POST` |
| `【headers】` | 请求头，多个用`|||`分隔 | `【headers】Content-Type:application/json|||Authorization:Bearer token` |
| `【data】` | POST请求的数据 | `【data】{"key":"value"}` |
| `【cookies】` | Cookie字符串 | `【cookies】sessionid=123; userid=456` |
| `【proxy】` | 代理服务器地址 | `【proxy】http://proxy.example.com:8080` |
| `【times】` | 执行次数，0=无限制 | `【times】3` |
| `【result】` | 自定义成功判断字符串 | `【result】success` |

##### 2. 命令模式 (`mode: "command"`)

用于执行系统命令或脚本。

**command 格式说明：**
```
【command】要执行的命令
【workdir】工作目录
【env】环境变量1|||环境变量2
【timeout】超时时间(秒)
```

**详细示例：**

1. **简单命令**
```json
{
  "name": "磁盘清理",
  "desc": "清理临时文件",
  "cron_expr": "0 0 4 * * *",
  "mode": "command",
  "command": "find /tmp -name '*.tmp' -mtime +7 -delete"
}
```

2. **带工作目录的命令**
```json
{
  "name": "备份脚本",
  "desc": "执行数据库备份脚本",
  "cron_expr": "0 0 2 * * *",
  "mode": "command",
  "command": "【command】./backup.sh\n【workdir】/opt/scripts"
}
```

3. **带环境变量的命令**
```json
{
  "name": "环境变量命令",
  "desc": "使用特定环境变量执行命令",
  "cron_expr": "0 0 6 * * *",
  "mode": "command",
  "command": "【command】echo $CUSTOM_VAR\n【env】CUSTOM_VAR=test_value|||DEBUG=true"
}
```

4. **带超时的命令**
```json
{
  "name": "超时命令",
  "desc": "设置超时时间的命令",
  "cron_expr": "0 */10 * * * *",
  "mode": "command",
  "command": "【command】long-running-script.sh\n【timeout】60"
}
```

5. **Windows系统命令**
```json
{
  "name": "Windows清理",
  "desc": "清理Windows临时文件",
  "cron_expr": "0 0 5 * * *",
  "mode": "command",
  "command": "del /q /f %TEMP%\\*.tmp"
}
```

**配置参数说明：**

| 参数 | 说明 | 示例 |
|------|------|------|
| `【command】` | 要执行的命令（必填） | `【command】ls -la` |
| `【workdir】` | 工作目录 | `【workdir】/opt/scripts` |
| `【env】` | 环境变量，多个用`|||`分隔 | `【env】PATH=/usr/bin|||DEBUG=true` |
| `【timeout】` | 超时时间（秒），默认30秒 | `【timeout】60` |

##### 3. 函数模式 (`mode: "func"`)

使用系统内置函数，支持参数传递。

**command 格式说明：**
```
【name】函数名
【arg】参数1,参数2,参数3
```

**内置函数列表：**

| 函数名 | 功能 | 参数格式 | 示例 |
|--------|------|----------|------|
| `Dayin` | 打印任务信息 | `参数1,参数2,参数3` | `Dayin 1,hello,true` |
| `Test` | 测试函数 | `任意参数` | `Test test123` |
| `Hello` | 问候函数 | `姓名` | `Hello 张三` |
| `Time` | 时间函数 | `时间格式` | `Time 2006-01-02 15:04:05` |
| `Echo` | 回显函数 | `任意文本` | `Echo Hello World` |
| `Math` | 数学计算 | `操作符,数字1,数字2` | `Math +,10,5` |
| `File` | 文件操作 | `操作,文件路径` | `File read,/path/to/file` |
| `Database` | 数据库操作 | `操作,SQL语句` | `Database query,SELECT * FROM users` |
| `Email` | 邮件发送 | `收件人,主题,内容` | `Email user@example.com,测试,邮件内容` |
| `SMS` | 短信发送 | `手机号,内容` | `SMS 13800138000,测试短信` |
| `Webhook` | Webhook调用 | `URL,数据` | `Webhook https://webhook.site/xxx,{"data":"value"}` |
| `Backup` | 备份操作 | `源路径,目标路径` | `Backup /data,/backup` |
| `Cleanup` | 清理操作 | `路径,天数` | `Cleanup /tmp,7` |
| `Monitor` | 监控检查 | `检查项` | `Monitor disk` |
| `Report` | 报告生成 | `报告类型` | `Report daily` |

**详细示例：**

1. **基础函数调用**
```json
{
  "name": "时间显示",
  "desc": "显示当前时间",
  "cron_expr": "0 */5 * * * *",
  "mode": "func",
  "command": "【name】Time\n【arg】2006-01-02 15:04:05"
}
```

2. **数学计算**
```json
{
  "name": "数学计算",
  "desc": "执行数学运算",
  "cron_expr": "0 */30 * * * *",
  "mode": "func",
  "command": "【name】Math\n【arg】+,100,50"
}
```

3. **文件操作**
```json
{
  "name": "文件检查",
  "desc": "检查文件状态",
  "cron_expr": "0 0 */2 * * *",
  "mode": "func",
  "command": "【name】File\n【arg】read,/var/log/app.log"
}
```

4. **数据库操作**
```json
{
  "name": "数据统计",
  "desc": "统计用户数量",
  "cron_expr": "0 0 1 * * *",
  "mode": "func",
  "command": "【name】Database\n【arg】query,SELECT COUNT(*) FROM users"
}
```

5. **复杂参数函数**
```json
{
  "name": "Dayin测试",
  "desc": "测试Dayin函数",
  "cron_expr": "0 */15 * * * *",
  "mode": "func",
  "command": "【name】Dayin\n【arg】1,hello,true"
}
```

**配置参数说明：**

| 参数 | 说明 | 示例 |
|------|------|------|
| `【name】` | 函数名（必填） | `【name】Time` |
| `【arg】` | 函数参数，用逗号分隔 | `【arg】参数1,参数2,参数3` |

##### Cron表达式说明

| 字段 | 允许值 | 特殊字符 | 说明 |
|------|--------|----------|------|
| 秒 | 0-59 | `* / , -` | 秒数（0-59） |
| 分 | 0-59 | `* / , -` | 分钟（0-59） |
| 时 | 0-23 | `* / , -` | 小时（0-23） |
| 日 | 1-31 | `* / , - ?` | 日期（1-31） |
| 月 | 1-12 | `* / , -` | 月份（1-12） |
| 周 | 0-7 | `* / , - ?` | 星期（0或7=周日） |

**常用Cron表达式示例：**

| 表达式 | 说明 |
|--------|------|
| `* * * * * *` | 每秒执行 |
| `0 * * * * *` | 每分钟执行 |
| `0 0 * * * *` | 每小时执行 |
| `0 0 0 * * *` | 每天0点执行 |
| `0 0 2 * * *` | 每天2点执行 |
| `0 30 9 * * *` | 每天9点30分执行 |
| `0 0 0 * * 1` | 每周一0点执行 |
| `0 0 0 1 * *` | 每月1号0点执行 |

#### 其他任务管理接口

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
- `/jobs/ip-control/status` 获取IP控制状态

### 管理员相关（暂时无用）
- `/admin/login` 登录
- `/admin/register` 注册
- `/admin/profile` 获取/更新个人信息
- `/admin/list` 管理员列表
- `/admin/status` 修改状态
- `/admin/delete` 删除管理员

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
EXPOSE 36363
CMD ["./main","start"]
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