package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"xiaohuAdmin/global"
	ipmiddlewares "xiaohuAdmin/middlewares"
	middlewares "xiaohuAdmin/middlewares/core"
	"xiaohuAdmin/routers"

	"github.com/gin-gonic/gin"
)

// Windows下DETACHED_PROCESS常量兼容
var DETACHED_PROCESS uint32 = 0x00000008

var (
	jobPidFile    = "runtime/job.pid"
	daemonPidFile = "runtime/daemon.pid"
)

// program 结构体，封装 HTTP 服务和控制通道
// 用于优雅启动和关闭主服务
type program struct {
	Srv      *http.Server
	doneChan chan struct{}
}

// Run 主程序入口，初始化配置、日志、数据库并启动服务
func Run() {
	if err := global.InitConfig(); err != nil {
		fmt.Printf("配置初始化失败: %v\n", err)
		os.Exit(1)
	}
	global.InitLogger()
	if err := global.ValidateConfig(); err != nil {
		global.ZapLog.Fatal("配置验证失败", global.LogError(err))
		os.Exit(1)
	}
	if err := runDirect(); err != nil {
		global.ZapLog.Fatal("服务启动失败", global.LogError(err))
		os.Exit(1)
	}
}

// runDirect 直接运行模式（前台运行），支持优雅关闭
func runDirect() error {
	global.ZapLog.Info("启动直接运行模式")
	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	prog := &program{}
	go func() {
		if err := prog.StartServer(); err != nil {
			global.ZapLog.Fatal("服务启动失败", global.LogError(err))
		}
	}()
	<-sigChan
	global.ZapLog.Info("收到停止信号，正在优雅关闭...")
	if err := prog.StopServer(); err != nil {
		global.ZapLog.Error("服务停止失败", global.LogError(err))
	}
	global.ZapLog.Info("服务已优雅停止")
	return nil
}

// StartServer 启动 HTTP 服务，初始化路由和中间件
func (p *program) StartServer() error {
	global.ZapLog.Info("开始初始化企业级定时任务系统")
	if err := global.InitDB(); err != nil {
		global.ZapLog.Fatal("数据库初始化失败", global.LogError(err))
		return err
	}
	gin.DisableConsoleColor()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.SetFuncMap(global.GetViewFuncMap())
	router.Use(middlewares.CustomRecovery())
	router.Use(middlewares.CORSMiddleware())
	router.Use(middlewares.GzipMiddleware())
	router.Use(middlewares.MemoryGuard(1024))
	router.Use(global.GinLogger())
	router.Use(ipmiddlewares.IPControl())
	routers.InitGlobal(router)
	global.InitJobs()
	config := global.GetGlobalConfig()
	port := config.Server.Port
	if port == "" {
		port = "8080"
	}
	p.Srv = &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	p.doneChan = make(chan struct{})
	global.ZapLog.Warn("企业级定时任务系统启动成功",
		global.LogField("port", port),
		global.LogField("os", runtime.GOOS),
		global.LogField("pid", os.Getpid()))
	global.ZapLog.Info("准备启动HTTP服务器", global.LogField("port", port))
	if err := p.Srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		global.ZapLog.Error("服务器启动失败", global.LogError(err))
		return err
	}
	global.ZapLog.Info("HTTP服务器已正常退出")
	close(p.doneChan)
	return nil
}

// StopServer 优雅关闭 HTTP 服务，释放资源
func (p *program) StopServer() error {
	global.ZapLog.Warn("正在优雅停止服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := p.Srv.Shutdown(ctx); err != nil {
		global.ZapLog.Error("服务关闭失败", global.LogError(err))
		return err
	}
	global.StopTimer()
	global.CloseDB()
	global.CloseAllFileHandles()
	global.ZapLog.Warn("服务已优雅停止")
	return nil
}

// RunBackground 后台模式运行，自动写入PID文件
func RunBackground() {
	if isRunning() {
		fmt.Println("系统已在后台运行")
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("获取可执行文件路径失败: %v\n", err)
		os.Exit(1)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command(execPath, "start")
	} else {
		cmd = exec.Command("nohup", execPath, "start", "&")
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("启动后台进程失败: %v\n", err)
		os.Exit(1)
	}

	writeJobPIDFile(cmd.Process.Pid)
	fmt.Printf("系统已在后台启动，PID: %d\n", cmd.Process.Pid)
}

// RunDaemon 守护进程模式运行，自动写入PID文件
func RunDaemon() {
	if isRunning() {
		fmt.Println("系统已在守护进程模式下运行")
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("获取可执行文件路径失败: %v\n", err)
		os.Exit(1)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "start", "/B", execPath, "daemon")
	} else {
		cmd = exec.Command("nohup", execPath, "daemon", "&")
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("启动守护进程失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("守护进程已启动，PID: %d\n", cmd.Process.Pid)
}

// writeJobPIDFile 写入业务进程PID到文件
func writeJobPIDFile(pid int) {
	dir := filepath.Dir(jobPidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建PID文件目录失败: %v\n", err)
		return
	}
	if err := os.WriteFile(jobPidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Printf("写入PID文件失败: %v\n", err)
	}
}

// readJobPIDFile 读取业务进程PID
func readJobPIDFile() int {
	data, err := os.ReadFile(jobPidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// removeJobPIDFile 删除业务进程PID文件
func removeJobPIDFile() { os.Remove(jobPidFile) }

// writeDaemonPIDFile 写入守护进程PID到文件
func writeDaemonPIDFile(pid int) {
	dir := filepath.Dir(daemonPidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建守护进程PID文件目录失败: %v\n", err)
		return
	}
	if err := os.WriteFile(daemonPidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Printf("写入守护进程PID文件失败: %v\n", err)
	}
}

// readDaemonPIDFile 读取守护进程PID
func readDaemonPIDFile() int {
	data, err := os.ReadFile(daemonPidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// removeDaemonPIDFile 删除守护进程PID文件
func removeDaemonPIDFile() { os.Remove(daemonPidFile) }

// DaemonLoop 守护进程主循环，支持自动重启
func DaemonLoop() {
	writeDaemonPIDFile(os.Getpid())
	defer removeDaemonPIDFile()
	maxRestarts := 100
	restartDelay := 3 * time.Second
	restarts := 0
	for {
		cmd := exec.Command(os.Args[0], "start")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Println("[守护] 启动业务子进程...")
		if err := cmd.Start(); err != nil {
			fmt.Println("[守护] 启动子进程失败:", err)
			time.Sleep(restartDelay)
			continue
		}
		pid := cmd.Process.Pid
		writeJobPIDFile(pid)
		fmt.Printf("[守护] 子进程PID: %d\n", pid)
		err := cmd.Wait()
		removeJobPIDFile()
		if err != nil {
			fmt.Println("[守护] 子进程异常退出:", err)
		} else {
			fmt.Println("[守护] 子进程正常退出")
		}
		restarts++
		if restarts >= maxRestarts {
			fmt.Println("[守护] 达到最大重启次数，守护进程退出")
			break
		}
		fmt.Printf("[守护] %d 秒后重启子进程...\n", int(restartDelay.Seconds()))
		time.Sleep(restartDelay)
	}
}

// StopBackground 停止后台运行进程
func StopBackground() {
	pid := readJobPIDFile()
	if pid == 0 {
		fmt.Println("未找到运行中的进程")
		return
	}

	if err := killProcess(pid); err != nil {
		fmt.Printf("停止进程失败: %v\n", err)
		os.Exit(1)
	}

	removeJobPIDFile()
	fmt.Printf("已停止进程 PID: %d\n", pid)
}

// StopDaemon 停止守护进程
func StopDaemon() {
	jobPid := readJobPIDFile()
	if jobPid != 0 {
		if err := killProcess(jobPid); err != nil {
			fmt.Printf("停止业务进程失败: %v\n", err)
		}
		removeJobPIDFile()
	}
	daemonPid := readDaemonPIDFile()
	if daemonPid != 0 {
		if err := killProcess(daemonPid); err != nil {
			fmt.Printf("停止守护进程失败: %v\n", err)
		}
		removeDaemonPIDFile()
	}
	fmt.Println("已优雅停止守护进程和业务进程")
}

// isRunning 检查主进程或守护进程是否已运行
func isRunning() bool {
	pid := readJobPIDFile()
	if pid == 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), strconv.Itoa(pid))
	} else {
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return false
		}
		return true
	}
}

// IsRunning 外部接口，检查系统是否运行中
func IsRunning() bool {
	return isRunning()
}

// killProcess 通过PID杀死进程
func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		// 优先尝试优雅关闭（不带 /F）
		cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid))
		if err := cmd.Run(); err != nil {
			// 如果失败再强杀
			cmd = exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F")
			return cmd.Run()
		}
		return nil
	} else {
		// Linux/Unix 优雅关闭
		if err := process.Signal(syscall.SIGTERM); err != nil {
			// 如果失败再强杀
			return process.Kill()
		}
		return nil
	}
}
