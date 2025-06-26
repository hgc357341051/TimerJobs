// @title 小胡定时任务系统 API
// @version 1.0.0
// @description 企业级定时任务管理系统，提供完整的任务调度、执行、监控功能。
// @termsOfService http://swagger.io/terms/

// @contact.name 小胡QQ357341051
// @contact.url http://www.swagger.io/support
// @contact.email 357341051@qq.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:36363
// @BasePath /

package main

import (
	"fmt"
	"os"

	"xiaohuAdmin/core"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		// 默认前台模式运行
		core.Run()
		return
	}

	switch args[0] {
	case "start":
		if len(args) > 1 {
			if args[1] == "-d" {
				if len(args) > 2 && args[2] == "-f" {
					// start -d -f 守护模式运行
					fmt.Println("启动守护模式...")
					core.RunDaemon()
				} else {
					// start -d 后台模式运行
					fmt.Println("启动后台模式...")
					core.RunBackground()
				}
			} else {
				fmt.Println("无效的参数，使用: start [前台模式] | start -d [后台模式] | start -d -f [守护模式]")
			}
		} else {
			// start 前台模式运行
			fmt.Println("启动前台模式...")
			core.Run()
		}
	case "stop":
		if len(args) > 1 && args[1] == "-f" {
			// stop -f 停止守护模式(后台进程和守护进程都退出)
			fmt.Println("停止守护模式...")
			core.StopDaemon()
			os.Exit(0)
		} else {
			// stop 停止后台模式
			fmt.Println("停止后台模式...")
			core.StopBackground()
			os.Exit(0)
		}
	case "status":
		// 查看状态
		if core.IsRunning() {
			fmt.Println("系统正在运行")
		} else {
			fmt.Println("系统未运行")
		}
	case "daemon":
		core.DaemonLoop()
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Printf("未知命令: %s\n", args[0])
		printHelp()
	}
}

func printHelp() {
	fmt.Println("小胡定时任务系统 - 使用说明")
	fmt.Println("")
	fmt.Println("命令格式:")
	fmt.Println("  start           - 前台模式运行")
	fmt.Println("  start -d        - 后台模式运行")
	fmt.Println("  start -d -f     - 守护模式运行")
	fmt.Println("  stop            - 停止后台模式")
	fmt.Println("  stop -f         - 停止守护模式(后台进程和守护进程都退出)")
	fmt.Println("  status          - 查看系统运行状态")
	fmt.Println("  daemon          - 进入守护模式")
	fmt.Println("  help            - 显示帮助信息")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  ./jobs start              # 前台运行")
	fmt.Println("  ./jobs start -d           # 后台运行")
	fmt.Println("  ./jobs start -d -f        # 守护进程运行")
	fmt.Println("  ./jobs stop               # 停止后台进程")
	fmt.Println("  ./jobs stop -f            # 停止所有相关进程")
	fmt.Println("  ./jobs status             # 查看运行状态")
	fmt.Println("  ./jobs daemon             # 进入守护模式")
}
