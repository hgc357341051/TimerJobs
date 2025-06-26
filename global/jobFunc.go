package global

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// JobFunction 任务函数类型定义
type JobFunction func(args []string) (string, error)

// FuncMap 函数映射表，统一管理所有任务函数
var FuncMap = make(map[string]JobFunction)

// 初始化函数映射
func init() {
	// 注册所有任务函数
	FuncMap["Dayin"] = Dayin
	FuncMap["Test"] = Test
	FuncMap["Hello"] = Hello
	FuncMap["Time"] = Time
	FuncMap["Echo"] = Echo
	FuncMap["Math"] = Math
	FuncMap["File"] = File
	FuncMap["Database"] = Database
	FuncMap["Email"] = Email
	FuncMap["SMS"] = SMS
	FuncMap["Webhook"] = Webhook
	FuncMap["Backup"] = Backup
	FuncMap["Cleanup"] = Cleanup
	FuncMap["Monitor"] = Monitor
	FuncMap["Report"] = Report
}

// GetFunction 获取函数
func GetFunction(name string) (JobFunction, bool) {
	fn, exists := FuncMap[name]
	return fn, exists
}

// ListFunctions 列出所有可用函数
func ListFunctions() []string {
	var functions []string
	for name := range FuncMap {
		functions = append(functions, name)
	}
	return functions
}

// Dayin 示例函数 - 打印任务信息
func Dayin(args []string) (string, error) {

	result := fmt.Sprintf("Dayin函数执行成功，参数: %v", args)

	// 根据参数执行不同逻辑
	if len(args) > 0 {
		firstArg := args[0]
		if firstArg == "1" {
			result += " - 参数1处理"
		}
	}

	if len(args) > 1 {
		secondArg := args[1]
		result += fmt.Sprintf(" - 字符串参数: %s", secondArg)
	}

	if len(args) > 2 {
		thirdArg := args[2]
		if thirdArg == "true" {
			result += " - 布尔参数为真"
		}
	}

	return result, nil
}

// Test 测试函数
func Test(args []string) (string, error) {

	result := fmt.Sprintf("Test函数执行成功，参数: %v", args)
	return result, nil
}

// Hello 问候函数
func Hello(args []string) (string, error) {

	name := "World"
	if len(args) > 0 {
		name = args[0]
	}

	result := fmt.Sprintf("Hello, %s!", name)
	return result, nil
}

// Time 时间函数
func Time(args []string) (string, error) {

	format := "2006-01-02 15:04:05"
	if len(args) > 0 {
		format = args[0]
	}

	result := fmt.Sprintf("当前时间: %s", time.Now().Format(format))
	return result, nil
}

// Echo 回显函数
func Echo(args []string) (string, error) {

	result := strings.Join(args, " ")
	return result, nil
}

// Math 数学计算函数
func Math(args []string) (string, error) {

	if len(args) < 3 {
		return "", fmt.Errorf("Math函数需要至少3个参数: 操作符 数字1 数字2")
	}

	op := args[0]
	num1, err1 := strconv.ParseFloat(args[1], 64)
	num2, err2 := strconv.ParseFloat(args[2], 64)

	if err1 != nil || err2 != nil {
		return "", fmt.Errorf("参数必须是数字")
	}

	var result float64
	switch op {
	case "+":
		result = num1 + num2
	case "-":
		result = num1 - num2
	case "*":
		result = num1 * num2
	case "/":
		if num2 == 0 {
			return "", fmt.Errorf("除数不能为零")
		}
		result = num1 / num2
	default:
		return "", fmt.Errorf("不支持的操作符: %s", op)
	}

	return fmt.Sprintf("%.2f", result), nil
}

// File 文件操作函数
func File(args []string) (string, error) {

	if len(args) < 2 {
		return "", fmt.Errorf("File函数需要至少2个参数: 操作 文件路径")
	}

	operation := args[0]
	filePath := args[1]

	switch operation {
	case "read":
		// 这里实现文件读取逻辑
		return fmt.Sprintf("读取文件: %s", filePath), nil
	case "write":
		// 这里实现文件写入逻辑
		return fmt.Sprintf("写入文件: %s", filePath), nil
	case "delete":
		// 这里实现文件删除逻辑
		return fmt.Sprintf("删除文件: %s", filePath), nil
	default:
		return "", fmt.Errorf("不支持的文件操作: %s", operation)
	}
}

// Database 数据库操作函数
func Database(args []string) (string, error) {

	if len(args) < 2 {
		return "", fmt.Errorf("Database函数需要至少2个参数: 操作 SQL语句")
	}

	operation := args[0]
	sql := args[1]

	switch operation {
	case "query":
		// 这里实现数据库查询逻辑
		return fmt.Sprintf("执行查询: %s", sql), nil
	case "execute":
		// 这里实现数据库执行逻辑
		return fmt.Sprintf("执行SQL: %s", sql), nil
	default:
		return "", fmt.Errorf("不支持的数据库操作: %s", operation)
	}
}

// Email 邮件发送函数（仅示例，未实现实际发送）
func Email(args []string) (string, error) {
	if len(args) < 3 {
		return "", fmt.Errorf("Email函数需要至少3个参数: 收件人 主题 内容")
	}
	to := args[0]
	subject := args[1]
	// args[2] 为内容参数，暂未用
	return fmt.Sprintf("发送邮件到: %s, 主题: %s", to, subject), nil
}

// SMS 短信发送函数（仅示例，未实现实际发送）
func SMS(args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("SMS函数需要至少2个参数: 手机号 内容")
	}
	phone := args[0]
	// args[1] 为内容参数，暂未用
	return fmt.Sprintf("发送短信到: %s", phone), nil
}

// Webhook Webhook调用函数（仅示例，未实现实际调用）
func Webhook(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("Webhook函数需要至少1个参数: URL")
	}
	url := args[0]
	return fmt.Sprintf("调用Webhook: %s", url), nil
}

// Backup 备份函数
func Backup(args []string) (string, error) {

	source := "."
	if len(args) > 0 {
		source = args[0]
	}

	// 这里实现备份逻辑
	result := fmt.Sprintf("备份目录: %s", source)
	return result, nil
}

// Cleanup 清理函数
func Cleanup(args []string) (string, error) {

	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// 这里实现清理逻辑
	result := fmt.Sprintf("清理目录: %s", path)
	return result, nil
}

// Monitor 监控函数
func Monitor(args []string) (string, error) {

	target := "system"
	if len(args) > 0 {
		target = args[0]
	}

	// 这里实现监控逻辑
	result := fmt.Sprintf("监控目标: %s", target)
	return result, nil
}

// Report 报告函数
func Report(args []string) (string, error) {

	reportType := "daily"
	if len(args) > 0 {
		reportType = args[0]
	}

	// 这里实现报告生成逻辑
	result := fmt.Sprintf("生成报告: %s", reportType)
	return result, nil
}
