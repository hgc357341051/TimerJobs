#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试脚本：验证MCP服务器与Golang后端API的集成
"""

import json
import subprocess
import time
import requests
import sys
import os
from typing import Dict, Any

class MCPTester:
    def __init__(self):
        self.server_process = None
        self.api_base = "http://127.0.0.1:36363"
        
    def check_backend_health(self) -> bool:
        """检查Golang后端是否运行"""
        try:
            response = requests.get(f"{self.api_base}/jobs/health", timeout=5)
            return response.status_code == 200
        except requests.exceptions.RequestException:
            return False
            
    def start_server(self):
        """启动MCP服务器"""
        try:
            # 使用UTF-8编码避免Windows编码问题
            self.server_process = subprocess.Popen([
                '.\\xiaohu-mcp-stdio.exe'
            ], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, 
               text=True, encoding='utf-8')
            time.sleep(2)  # 等待服务器启动
            return True
        except Exception as e:
            print(f"启动服务器失败: {e}")
            return False
            
    def send_request(self, method: str, params: Dict[str, Any] = None) -> Dict[str, Any]:
        """发送JSON-RPC请求到MCP服务器"""
        if not params:
            params = {}
            
        request_data = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params
        }
        
        try:
            json_str = json.dumps(request_data, ensure_ascii=False) + "\n"
            self.server_process.stdin.write(json_str)
            self.server_process.stdin.flush()
            
            response = self.server_process.stdout.readline()
            if not response:
                return {"error": "服务器无响应"}
            
            # 强制使用UTF-8解码
            try:
                return json.loads(response)
            except UnicodeDecodeError:
                # 如果解码失败，尝试使用raw_unicode_escape
                response = response.encode('utf-8').decode('utf-8')
                return json.loads(response)
                
        except Exception as e:
            print(f"发送请求失败: {e}")
            return {"error": str(e)}
            
    def test_list_jobs(self) -> bool:
        """测试列出任务"""
        print("测试: 列出所有任务...")
        
        # 先直接测试API
        try:
            api_response = requests.get(f"{self.api_base}/jobs/list?page=1&size=10")
            if api_response.status_code == 200:
                api_data = api_response.json()
                print(f"✓ API响应正常: 找到 {len(api_data.get('data', []))} 个任务")
            else:
                print(f"✗ API响应异常: {api_response.status_code}")
                return False
        except Exception as e:
            print(f"✗ API连接失败: {e}")
            return False
            
        # 测试MCP工具
        response = self.send_request("tools/call", {
            "name": "list_jobs",
            "arguments": {
                "page": 1,
                "size": 10
            }
        })
        
        if "result" in response:
            content = response["result"]["content"]
            if isinstance(content, list) and len(content) > 0:
                jobs_data = json.loads(content[0]["text"])
                print(f"✓ MCP工具正常: 返回 {len(jobs_data.get('jobs', []))} 个任务")
                return True
        
        print(f"✗ MCP工具失败: {response}")
        return False
        
    def test_get_job(self) -> bool:
        """测试获取单个任务详情"""
        print("测试: 获取任务详情...")
        
        # 先获取一个任务ID
        api_response = requests.get(f"{self.api_base}/jobs/list?page=1&size=1")
        if api_response.status_code != 200:
            print("✗ 无法获取任务列表")
            return False
            
        jobs_data = api_response.json()
        if not jobs_data.get("data"):
            print("✗ 没有找到任务")
            return True
            
        job_id = str(jobs_data["data"][0]["id"])
        
        # 测试MCP工具
        response = self.send_request("tools/call", {
            "name": "get_job",
            "arguments": {
                "job_id": job_id
            }
        })
        
        if "result" in response:
            content = response["result"]["content"]
            if isinstance(content, list) and len(content) > 0:
                job_data = json.loads(content[0]["text"])
                print(f"✓ 成功获取任务详情: {job_data.get('name', 'Unknown')}")
                return True
        
        print(f"✗ 获取任务详情失败: {response}")
        return False
        
    def test_create_job(self) -> bool:
        """测试创建新任务"""
        print("测试: 创建新任务...")
        
        test_job_name = f"test-job-{int(time.time())}"
        response = self.send_request("tools/call", {
            "name": "create_job",
            "arguments": {
                "name": test_job_name,
                "command": "echo 'Hello World'",
                "cron_expr": "*/5 * * * * *",
                "mode": "command"
            }
        })
        
        if "result" in response:
            content = response["result"]["content"]
            if isinstance(content, list) and len(content) > 0:
                result_text = content[0]["text"]
                if "successfully" in result_text.lower():
                    print(f"✓ 创建任务成功: {test_job_name}")
                    return True
        
        print(f"✗ 创建任务失败: {response}")
        return False
        
    def test_system_resources(self) -> bool:
        """测试系统资源"""
        print("测试: 系统资源访问...")
        
        # 测试健康检查
        response = self.send_request("resources/read", {
            "uri": "xiaohu://health"
        })
        
        if "contents" in response.get("result", {}):
            print("✓ 健康检查资源正常")
        else:
            print("✗ 健康检查资源失败")
            return False
            
        # 测试任务概览
        response = self.send_request("resources/read", {
            "uri": "xiaohu://jobs/overview"
        })
        
        if "contents" in response.get("result", {}):
            print("✓ 任务概览资源正常")
            return True
        else:
            print("✗ 任务概览资源失败")
            return False
            
    def test_job_operations(self) -> bool:
        """测试任务操作（启动/停止）"""
        print("测试: 任务操作...")
        
        # 获取一个任务进行测试
        api_response = requests.get(f"{self.api_base}/jobs/list?page=1&size=1")
        if api_response.status_code != 200 or not api_response.json().get("data"):
            print("✗ 没有可用的任务进行测试")
            return True
            
        job = api_response.json()["data"][0]
        job_id = str(job["id"])
        
        # 测试停止任务
        response = self.send_request("tools/call", {
            "name": "stop_job",
            "arguments": {
                "job_id": job_id
            }
        })
        
        if "result" in response:
            print("✓ 停止任务操作正常")
            
            # 测试启动任务
            response = self.send_request("tools/call", {
                "name": "start_job",
                "arguments": {
                    "job_id": job_id
                }
            })
            
            if "result" in response:
                print("✓ 启动任务操作正常")
                return True
                
        print("✗ 任务操作测试失败")
        return False
        
    def run_all_tests(self) -> bool:
        """运行所有测试"""
        print("=" * 50)
        print("MCP服务器与Golang后端集成测试")
        print("=" * 50)
        
        # 检查后端
        if not self.check_backend_health():
            print("✗ Golang后端未运行，请先启动后端服务")
            return False
        
        print("✓ Golang后端运行正常")
        
        # 启动MCP服务器
        if not self.start_server():
            return False
            
        try:
            # 初始化连接
            init_response = self.send_request("initialize", {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "test-client", "version": "1.0.0"}
            })
            
            if "error" in init_response:
                print(f"✗ 初始化失败: {init_response['error']}")
                return False
                
            print("✓ 服务器初始化成功")
            
            # 运行各项测试
            tests = [
                self.test_list_jobs,
                self.test_get_job,
                self.test_create_job,
                self.test_system_resources,
                self.test_job_operations
            ]
            
            passed = 0
            total = len(tests)
            
            for test in tests:
                try:
                    if test():
                        passed += 1
                    time.sleep(1)  # 避免请求过快
                except Exception as e:
                    print(f"✗ 测试异常: {e}")
                    
            print("=" * 50)
            print(f"测试结果: {passed}/{total} 通过")
            print("=" * 50)
            
            return passed == total
            
        finally:
            if self.server_process:
                self.server_process.terminate()
                self.server_process.wait()

if __name__ == "__main__":
    tester = MCPTester()
    success = tester.run_all_tests()
    sys.exit(0 if success else 1)