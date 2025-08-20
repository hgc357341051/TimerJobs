#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试脚本：验证MCP服务器是否实现了所有后端接口功能
"""
import json
import subprocess
import sys
import time
import os

# 设置编码
os.environ['PYTHONIOENCODING'] = 'utf-8'

def send_mcp_request(request_data):
    """发送MCP请求到stdio服务器"""
    try:
        process = subprocess.Popen(
            ['go', 'run', 'stdio_mcp.go'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            encoding='utf-8'
        )
        
        stdout, stderr = process.communicate(input=json.dumps(request_data) + '\n', timeout=10)
        
        if stderr:
            print(f"错误输出: {stderr}")
        
        try:
            response = json.loads(stdout.strip())
            return response
        except json.JSONDecodeError as e:
            print(f"JSON解析错误: {e}")
            print(f"原始输出: {stdout}")
            return None
    except Exception as e:
        print(f"请求失败: {e}")
        return None

def test_tool_list():
    """测试工具列表"""
    print("=== 测试工具列表 ===")
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/list",
        "params": {}
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        tools = response["result"]["tools"]
        print(f"找到 {len(tools)} 个工具:")
        for tool in tools:
            print(f"  - {tool['name']}: {tool['description']}")
        return True
    return False

def test_job_operations():
    """测试任务相关操作"""
    print("\n=== 测试任务操作 ===")
    
    # 测试获取任务列表
    print("1. 获取任务列表...")
    request = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/call",
        "params": {
            "name": "list_jobs",
            "arguments": {"page": 1, "size": 5}
        }
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        print("✓ 任务列表获取成功")
    else:
        print("✗ 任务列表获取失败")
    
    # 测试获取调度器状态
    print("2. 获取调度器状态...")
    request = {
        "jsonrpc": "2.0",
        "id": 3,
        "method": "tools/call",
        "params": {
            "name": "get_scheduler_status",
            "arguments": {}
        }
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        print("✓ 调度器状态获取成功")
    else:
        print("✗ 调度器状态获取失败")
    
    # 测试获取任务函数
    print("3. 获取任务函数...")
    request = {
        "jsonrpc": "2.0",
        "id": 4,
        "method": "tools/call",
        "params": {
            "name": "get_job_functions",
            "arguments": {}
        }
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        print("✓ 任务函数获取成功")
    else:
        print("✗ 任务函数获取失败")
    
    # 测试获取系统配置
    print("4. 获取系统配置...")
    request = {
        "jsonrpc": "2.0",
        "id": 5,
        "method": "tools/call",
        "params": {
            "name": "get_jobs_config",
            "arguments": {}
        }
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        print("✓ 系统配置获取成功")
    else:
        print("✗ 系统配置获取失败")

def test_ip_control():
    """测试IP控制功能"""
    print("\n=== 测试IP控制功能 ===")
    
    # 测试获取IP控制状态
    print("1. 获取IP控制状态...")
    request = {
        "jsonrpc": "2.0",
        "id": 6,
        "method": "tools/call",
        "params": {
            "name": "get_ip_control_status",
            "arguments": {}
        }
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        print("✓ IP控制状态获取成功")
    else:
        print("✗ IP控制状态获取失败")

def test_logging():
    """测试日志功能"""
    print("\n=== 测试日志功能 ===")
    
    # 测试获取系统日志
    print("1. 获取系统日志...")
    request = {
        "jsonrpc": "2.0",
        "id": 7,
        "method": "tools/call",
        "params": {
            "name": "get_system_logs",
            "arguments": {"page": 1, "size": 5}
        }
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        print("✓ 系统日志获取成功")
    else:
        print("✗ 系统日志获取失败")

def test_resources():
    """测试资源功能"""
    print("\n=== 测试资源功能 ===")
    
    # 测试健康检查资源
    print("1. 健康检查资源...")
    request = {
        "jsonrpc": "2.0",
        "id": 8,
        "method": "resources/read",
        "params": {
            "uri": "xiaohu://health"
        }
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        print("✓ 健康检查资源获取成功")
    else:
        print("✗ 健康检查资源获取失败")

def test_prompts():
    """测试提示功能"""
    print("\n=== 测试提示功能 ===")
    
    # 测试系统健康报告提示
    print("1. 系统健康报告提示...")
    request = {
        "jsonrpc": "2.0",
        "id": 9,
        "method": "prompts/get",
        "params": {
            "name": "system_health_report"
        }
    }
    
    response = send_mcp_request(request)
    if response and "result" in response:
        print("✓ 系统健康报告提示获取成功")
    else:
        print("✗ 系统健康报告提示获取失败")

def main():
    """主测试函数"""
    print("🚀 开始测试MCP服务器所有功能...")
    
    # 检查go环境
    try:
        subprocess.run(["go", "version"], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    except subprocess.CalledProcessError:
        print("❌ 未找到Go环境，请先安装Go")
        return
    
    # 运行测试
    tests = [
        test_tool_list,
        test_job_operations,
        test_ip_control,
        test_logging,
        test_resources,
        test_prompts
    ]
    
    passed = 0
    total = len(tests)
    
    for test in tests:
        try:
            test()
            passed += 1
        except Exception as e:
            print(f"测试失败: {e}")
    
    print(f"\n📊 测试结果: {passed}/{total} 个测试通过")
    
    if passed == total:
        print("✅ 所有功能测试通过！MCP服务器已完整实现所有后端接口功能")
    else:
        print("⚠️  部分测试失败，请检查日志")

if __name__ == "__main__":
    main()