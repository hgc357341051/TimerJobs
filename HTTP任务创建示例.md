# 任务创建与更新示例文档

## 支持的3种任务类型

系统现在支持以下3种任务类型，每种类型都有详细的配置参数：

1. **HTTP任务** (`mode="http"`) - 执行HTTP请求
2. **命令任务** (`mode="command"`) - 执行系统命令
3. **函数任务** (`mode="func"`) - 调用系统函数

## 创建任务示例

### 1. HTTP任务

#### 基本HTTP GET请求
```json
{
  "name": "健康检查",
  "mode": "http",
  "url": "https://api.example.com/health",
  "cron_expr": "*/5 * * * *",
  "desc": "每5分钟检查API健康状态"
}
```

#### 带代理的HTTP POST请求
```json
{
  "name": "数据同步",
  "mode": "http",
  "url": "https://api.example.com/sync",
  "http_mode": "POST",
  "proxy": "http://proxy.example.com:8080",
  "headers": "Content-Type:application/json|||Authorization:Bearer token123",
  "data": "{\"sync\":true}",
  "timeout": 120,
  "cron_expr": "0 */6 * * *",
  "desc": "每6小时同步一次数据"
}
```

#### SOCKS5代理配置
```json
{
  "name": "外部API调用",
  "mode": "http",
  "url": "https://external-api.com/data",
  "proxy": "socks5://127.0.0.1:1080",
  "times": 3,
  "interval": 30,
  "desc": "通过SOCKS5代理访问外部API"
}
```

### 2. 命令任务

#### 基本命令执行
```json
{
  "name": "备份数据库",
  "mode": "command",
  "cmd": "mysqldump -u root -p123456 mydb > backup.sql",
  "workdir": "/opt/backups",
  "cron_expr": "0 2 * * *",
  "desc": "每天凌晨2点备份数据库"
}
```

#### 带环境变量的命令
```json
{
  "name": "部署应用",
  "mode": "command",
  "cmd": "npm run build && pm2 restart all",
  "workdir": "/var/www/myapp",
  "env": "NODE_ENV=production|||PORT=3000",
  "times": 1,
  "timeout": 300,
  "desc": "构建并重启应用"
}
```

### 3. 函数任务

#### 调用系统函数
```json
{
  "name": "清理日志",
  "mode": "func",
  "func_name": "clean_logs",
  "arg": "30d",
  "cron_expr": "0 3 * * 0",
  "desc": "每周日凌晨3点清理30天前的日志"
}
```

#### 带参数的函数调用
```json
{
  "name": "系统监控",
  "mode": "func",
  "func_name": "system_check",
  "arg": "--cpu --memory --disk",
  "interval": 60,
  "times": 1440,
  "desc": "每分钟检查一次系统资源"
}
```

## 命令行使用示例

### 使用create_job工具

#### 创建HTTP任务
```bash
# 基本HTTP GET
create_job --name "网站监控" --mode http --url "https://example.com" --cron_expr "*/10 * * * *"

# 带代理的HTTP POST
create_job --name "数据推送" --mode http --url "https://api.example.com/data" --http_mode POST --proxy "http://proxy.example.com:8080" --data '{"key":"value"}' --headers "Content-Type:application/json|||Authorization:Bearer token"

# 带SOCKS5代理
create_job --name "外部访问" --mode http --url "https://external.com" --proxy "socks5://127.0.0.1:1080" --timeout 120
```

#### 创建命令任务
```bash
# 基本命令
create_job --name "日志清理" --mode command --cmd "find /var/log -name '*.log' -mtime +7 -delete" --cron_expr "0 4 * * *"

# 带工作目录和环境变量
create_job --name "构建项目" --mode command --cmd "make build" --workdir "/opt/project" --env "GOOS=linux|||GOARCH=amd64" --timeout 600
```

#### 创建函数任务
```bash
# 基本函数调用
create_job --name "清理缓存" --mode func --func_name "clear_cache" --arg "all" --cron_expr "0 6 * * *"

# 带参数和间隔
create_job --name "健康检查" --mode func --func_name "health_check" --arg "--verbose" --interval 30 --times 100
```

### 使用update_job工具

#### 更新HTTP任务
```bash
# 更新代理设置
update_job --job_id 123 --proxy "http://new-proxy.com:8080"

# 更新URL和超时时间
update_job --job_id 123 --url "https://new-api.com/endpoint" --timeout 180
```

#### 更新命令任务
```bash
# 更新命令和工作目录
update_job --job_id 456 --cmd "python3 new_script.py" --workdir "/opt/new_project"

# 更新环境变量
update_job --job_id 456 --env "ENV=production|||DEBUG=false"
```

#### 更新函数任务
```bash
# 更新函数参数
update_job --job_id 789 --arg "--mode=fast --verbose"

# 更新函数名
update_job --job_id 789 --func_name "new_function"
```

## 参数说明表

### 通用参数
| 参数名 | 类型 | 说明 | 适用模式 |
|--------|------|------|----------|
| name | string | 任务名称 | 所有模式 |
| command | string | 命令/URL/配置字符串 | 所有模式 |
| mode | string | 任务类型: http/command/func | 所有模式 |
| cron_expr | string | cron表达式 | 所有模式 |
| max_run_count | number | 最大运行次数 | 所有模式 |
| allow_mode | number | 并发模式 | 所有模式 |
| desc | string | 任务描述 | 所有模式 |
| times | number | 执行次数 | http/command/func |
| interval | number | 执行间隔(秒) | http/command/func |

### HTTP任务专用参数
| 参数名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| url | string | HTTP URL | "https://api.example.com" |
| http_mode | string | HTTP方法 | "GET", "POST", "PUT", "DELETE" |
| proxy | string | 代理地址 | "http://proxy.com:8080", "socks5://127.0.0.1:1080" |
| headers | string | HTTP头 | "key1:value1|||key2:value2" |
| data | string | POST数据 | '{"key":"value"}' |
| cookies | string | cookies | "session=abc123; token=xyz789" |
| result | string | 成功匹配字符串 | "success" |
| timeout | number | 超时时间(秒) | 60 |

### 命令任务专用参数
| 参数名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| cmd | string | 命令字符串 | "ls -la" |
| workdir | string | 工作目录 | "/opt/project" |
| env | string | 环境变量 | "key1=value1|||key2=value2" |
| timeout | number | 超时时间(秒) | 30 |

### 函数任务专用参数
| 参数名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| func_name | string | 函数名 | "clean_logs" |
| arg | string | 函数参数 | "30d" |

## 注意事项

1. **参数优先级**：特定模式的专用参数（如url、cmd、func_name）会覆盖command参数
2. **代理支持**：HTTP模式支持HTTP代理和SOCKS5代理两种格式
3. **格式规范**：
   - headers/env格式：`key1:value1|||key2:value2`
   - cookies格式：`name=value; name2=value2`
4. **向后兼容**：所有旧版本的使用方式仍然有效
5. **任务类型**：确保mode参数与专用参数匹配，否则可能无法正常工作

## 实际使用场景示例

### 场景1：网站监控
```json
{
  "name": "网站可用性监控",
  "mode": "http",
  "url": "https://mywebsite.com/health",
  "proxy": "http://company-proxy:8080",
  "result": "OK",
  "timeout": 30,
  "cron_expr": "*/5 * * * *",
  "desc": "监控网站健康状态"
}
```

### 场景2：定时备份
```json
{
  "name": "数据库备份",
  "mode": "command",
  "cmd": "pg_dump -U postgres mydb > /backups/$(date +%Y%m%d_%H%M%S).sql",
  "workdir": "/opt/backups",
  "env": "PGPASSWORD=mypassword",
  "cron_expr": "0 2 * * *",
  "desc": "每天凌晨2点备份数据库"
}
```

### 场景3：系统清理
```json
{
  "name": "临时文件清理",
  "mode": "func",
  "func_name": "cleanup_temp",
  "arg": "--older-than=7d --path=/tmp",
  "cron_expr": "0 3 * * 0",
  "desc": "每周清理临时文件"
}
```