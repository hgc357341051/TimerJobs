@echo off
title MCP服务器测试
chcp 65001

echo ================================================
echo MCP服务器测试脚本
echo ================================================

echo 检查Golang后端...
curl -s http://127.0.0.1:36363/jobs/health > nul
if errorlevel 1 (
    echo 错误：请先启动Golang后端服务！
    echo 运行：go run d:\1\app\jobs\main.go
    pause
    exit /b 1
)
echo ✅ Golang后端运行正常

echo.
echo 检查服务器文件...
if exist xiaohu-mcp-stdio.exe (
    echo ✅ 服务器可执行文件已找到
) else (
    echo ❌ 服务器可执行文件未找到
    echo 正在编译...
    go build -o xiaohu-mcp-stdio.exe stdio_mcp.go
    if errorlevel 1 (
        echo ❌ 编译失败
        pause
        exit /b 1
    )
    echo ✅ 编译成功
)

echo.
echo 运行集成测试...
set PYTHONIOENCODING=utf-8
python -X utf8 test_stdio_mcp.py

echo.
echo 测试完成！
pause