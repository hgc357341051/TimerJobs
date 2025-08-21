# Xiaohu Jobs MCP SSE Server

这是一个将 Xiaohu Jobs MCP 从 stdio 模式改为 SSE (Server-Sent Events) 模式的实现。

## 功能特性

- **SSE 协议支持**: 使用 Server-Sent Events 实现实时通信
- **HTTP 服务器**: 基于标准 HTTP 协议，便于集成和调试
- **完整功能**: 保持原有所有 MCP 工具和功能
- **跨平台**: 支持所有支持 HTTP 的客户端

## 快速开始

### 1. 启动 SSE 服务器

```bash
cd d:\1\app\jobs\mcp
go run SSE_MCP_server.go
```

服务器将在端口 `8080` 启动，输出类似：
```
Starting SSE MCP server on port :8080...
SSE endpoint: http://localhost:8080/sse
```

### 2. 连接方式

#### 使用 Claude Desktop

在 `claude_desktop_config.json` 中添加：

```json
{
  "mcpServers": {
    "xiaohu-jobs-sse": {
      "url": "http://localhost:8080/mcp",
      "transport": "sse"
    }
  }
}
```

配置文件位置：
- **Windows**: `%APPDATA%/Claude/claude_desktop_config.json`
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

#### 使用 Cursor

在 Cursor 设置中添加 MCP 服务器：

1. 打开 Cursor 设置 (`Ctrl/Cmd + ,`)
2. 搜索 "MCP"
3. 点击 "Add MCP Server"
4. 选择 "HTTP" 类型
5. 输入 URL: `http://localhost:8080/mcp`

或者手动编辑配置文件：

**Windows**: `%APPDATA%/Cursor/User/settings.json`
**macOS**: `~/Library/Application Support/Cursor/User/settings.json`
**Linux**: `~/.config/Cursor/User/settings.json`

```json
{
  "mcp": {
    "servers": {
      "xiaohu-jobs": {
        "url": "http://localhost:8080/mcp",
        "type": "sse"
      }
    }
  }
}
```

#### 使用 Windsurf

在 Windsurf 设置中添加：

1. 打开设置 (`Ctrl/Cmd + ,`)
2. 搜索 "MCP"
3. 添加新的 MCP 服务器
4. 输入 URL: `http://localhost:8080/mcp`

#### 使用 VS Code (with MCP extension)

安装 MCP 扩展后，在 `settings.json` 中添加：

```json
{
  "mcp": {
    "servers": {
      "xiaohu-jobs": {
        "url": "http://localhost:8080/mcp",
        "transport": "sse"
      }
    }
  }
}
```

#### 使用 Zed

在 Zed 设置中添加：

```json
{
  "assistant": {
    "mcp": {
      "servers": {
        "xiaohu-jobs": {
          "url": "http://localhost:8080/mcp",
          "transport": "sse"
        }
      }
    }
  }
}
```

#### 使用 Cline (VS Code extension)

在 Cline 的 MCP 设置中添加：

```json
{
  "mcpServers": {
    "xiaohu-jobs": {
      "url": "http://localhost:8080/mcp",
      "transport": "sse"
    }
  }
}
```

#### 使用其他 MCP 客户端

- **SSE 端点**: `http://localhost:8080/mcp`
- **健康检查**: `http://localhost:8080/health`
- **Web 界面**: `http://localhost:8080/`

### 3. 可用工具

服务器提供以下 MCP 工具：

- `list_jobs` - 列出所有任务
- `get_job` - 获取特定任务详情
- `create_job` - 创建新任务
- `update_job` - 更新任务
- `delete_job` - 删除任务
- `start_job` - 启动任务
- `stop_job` - 停止任务
- `get_job_logs` - 获取任务日志
- `run_job` - 手动运行任务
- `restart_job` - 重启任务
- `get_system_logs` - 获取系统日志
- `get_scheduler_status` - 获取调度器状态
- `start_all_jobs` - 启动所有任务
- `stop_all_jobs` - 停止所有任务
- `get_job_functions` - 获取可用函数
- `get_jobs_config` - 获取系统配置
- `reload_config` - 重载配置
- `clear_job_logs` - 清除任务日志
- `calibrate_job_list` - 校准任务列表
- IP 控制相关工具（白名单/黑名单管理）

## API 端点

| 端点 | 描述 |
|------|------|
| `GET /` | Web 界面，显示服务器信息和工具列表 |
| `GET /sse` | SSE 端点，用于 MCP 客户端连接 |

## 配置选项

### 修改服务器端口

可以通过以下方式配置服务器端口：

1. **使用环境变量**（推荐）：
```bash
# Windows
set SERVER_PORT=:8081

# Linux/Mac
export SERVER_PORT=:8081
```

2. **修改代码默认值**（在 `SSE_MCP_server.go` 中）：
```go
// 在文件顶部修改默认值
var (
    ServerPort = getEnvWithDefault("SERVER_PORT", ":8080") // 修改为你想要的端口
)
```

默认端口为：`:8080`

### 修改 API 地址

如果需要连接到不同的 Xiaohu Jobs API 地址，可以通过以下方式配置：

1. **使用环境变量**（推荐）：
```bash
# Windows
set API_BASE_URL=http://your-api-server:36363

# Linux/Mac
export API_BASE_URL=http://your-api-server:36363
```

2. **修改代码默认值**（在 `SSE_MCP_server.go` 中）：
```go
// 在文件顶部修改默认值
var (
    APIBaseURL = getEnvWithDefault("API_BASE_URL", "http://your-api-server:36363")
)
```

默认地址为：`http://127.0.0.1:36363`

## 开发说明

### 文件结构

- `SSE_MCP_server.go` - SSE 服务器主文件
- `SSE_MCP.go` - 原始的 stdio 版本（保留作为参考）
- `README_SSE.md` - 本文档

### 与原始版本的区别

1. **通信方式**: 从 stdio 改为 HTTP SSE
2. **启动方式**: 从命令行参数改为 HTTP 服务器
3. **连接方式**: 支持多个客户端同时连接
4. **调试友好**: 提供 Web 界面和 HTTP 接口

### 添加新功能

要添加新的工具或资源，请：

1. 在 `addTools()` 函数中添加新工具定义
2. 实现对应的工具处理函数
3. 在 `addResources()` 函数中添加新资源（如需要）

## 调试技巧

### 使用 curl 测试 SSE

```bash
curl -N -H "Accept: text/event-stream" http://localhost:8080/sse
```

### 浏览器测试

1. 打开 `http://localhost:8080/` 查看服务器信息
2. 使用浏览器开发者工具的 Network 标签查看 SSE 连接

### 日志调试

服务器会在控制台输出启动信息。如需更详细的日志，可以添加日志中间件。

## 故障排除

### 端口占用

如果端口 8080 被占用：

1. 修改 `ServerPort` 常量
2. 或者关闭占用端口的程序

### 连接问题

- 确保 Xiaohu Jobs API 服务正在运行
- 检查 `APIBaseURL` 配置是否正确
- 使用浏览器访问 `http://localhost:8080/` 验证服务器是否启动

### CORS 问题

服务器已配置允许所有来源的 CORS。如需限制，请修改 `handleSSE` 函数中的 CORS 头设置。

## 许可证

与原项目保持一致。