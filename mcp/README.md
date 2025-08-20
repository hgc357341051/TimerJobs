# Xiaohu Jobs MCP 服务器

这是一个基于Golang后端的MCP（Model Context Protocol）服务器，提供任务管理功能。

## 功能特性

- **任务管理**: 创建、读取、更新、删除任务
- **任务控制**: 启动、停止任务
- **系统监控**: 查看系统健康状态和资源使用
- **任务概览**: 获取任务统计信息

## 安装和运行

### 前提条件

1. 已安装Golang后端服务
2. 后端服务运行在 `http://127.0.0.1:36363`

### 编译

```bash
go build -o xiaohu-mcp-stdio.exe stdio_mcp.go
```

### 测试

运行集成测试：
```bash
# Windows
run_tests.bat

# 手动测试
python test_stdio_mcp.py
```

## 使用示例

### 1. 基本使用

```python
import json
import subprocess

# 启动MCP服务器
process = subprocess.Popen([
    './xiaohu-mcp-stdio.exe'
], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True, encoding='utf-8')

# 发送初始化请求
init_request = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "client", "version": "1.0.0"}
    }
}

# 发送请求并接收响应
json_str = json.dumps(init_request) + "\n"
process.stdin.write(json_str)
process.stdin.flush()
response = process.stdout.readline()
```

### 2. 任务管理

#### 列出所有任务
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
        "name": "list_jobs",
        "arguments": {
            "page": 1,
            "size": 10
        }
    }
}
```

#### 创建新任务
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
        "name": "create_job",
        "arguments": {
            "name": "my-task",
            "command": "echo 'Hello World'",
            "cron_expr": "*/5 * * * * *",
            "mode": "command"
        }
    }
}
```

#### 获取任务详情
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
        "name": "get_job",
        "arguments": {
            "job_id": "1"
        }
    }
}
```

### 3. 系统资源访问

#### 健康检查
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "resources/read",
    "params": {
        "uri": "xiaohu://health"
    }
}
```

#### 任务概览
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "resources/read",
    "params": {
        "uri": "xiaohu://jobs/overview"
    }
}
```

## 支持的MCP功能

### 工具 (Tools)
- `list_jobs`: 列出任务
- `get_job`: 获取任务详情
- `create_job`: 创建任务
- `update_job`: 更新任务
- `delete_job`: 删除任务
- `start_job`: 启动任务
- `stop_job`: 停止任务
- `get_job_logs`: 获取任务日志

### 资源 (Resources)
- `xiaohu://health`: 系统健康状态
- `xiaohu://jobs/overview`: 任务概览信息
- `xiaohu://config`: 系统配置

## 注意事项

- 确保Golang后端服务正在运行
- 在Windows系统上，使用UTF-8编码运行测试：
  ```bash
  set PYTHONIOENCODING=utf-8
  python -X utf8 test_stdio_mcp.py
  ```

## 故障排除

### 编码问题
如果遇到编码错误，请：
1. 设置环境变量 `PYTHONIOENCODING=utf-8`
2. 使用 `python -X utf8` 运行Python脚本
3. 确保所有文件使用UTF-8编码

### 连接问题
1. 检查Golang后端是否运行：
   ```bash
   curl http://127.0.0.1:36363/jobs/health
   ```
2. 确认端口36363未被占用

## 项目结构

```
mcp/
├── stdio_mcp.go          # 主MCP服务器代码
├── test_stdio_mcp.py     # 集成测试脚本
├── demo.py              # 使用示例
├── run_tests.bat        # Windows测试脚本
├── mcp_config.json      # MCP配置
└── README.md            # 本文档
```