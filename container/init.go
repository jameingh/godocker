package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// InitContainer 在容器命名空间中运行的初始化函数
// 作为容器的1号进程，负责设置容器环境并执行用户命令
func InitContainer() error {
	// 获取环境变量中的容器配置
	rootfs := os.Getenv("CONTAINER_ROOTFS")
	cmdString := os.Getenv("CONTAINER_CMD")
	containerName := os.Getenv("CONTAINER_NAME")

	if rootfs == "" || cmdString == "" {
		return fmt.Errorf("缺少必要的容器环境配置")
	}

	fmt.Printf("初始化容器: %s (rootfs: %s)\n", containerName, rootfs)

	// 设置主机名
	if err := setHostname(containerName); err != nil {
		return fmt.Errorf("设置主机名失败: %v", err)
	}

	// 挂载文件系统
	if err := setupContainerMounts(rootfs); err != nil {
		return fmt.Errorf("设置容器挂载点失败: %v", err)
	}

	// 切换根目录
	if err := syscall.Chroot(rootfs); err != nil {
		return fmt.Errorf("chroot失败: %v", err)
	}

	// 切换工作目录
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("切换工作目录失败: %v", err)
	}

	// 解析命令
	cmdParts := strings.Split(cmdString, " ")
	if len(cmdParts) == 0 {
		return fmt.Errorf("无效的容器命令")
	}

	// 查找命令路径
	cmdPath, err := exec.LookPath(cmdParts[0])
	if err != nil {
		return fmt.Errorf("找不到命令 %s: %v", cmdParts[0], err)
	}

	fmt.Printf("在容器中执行命令: %s\n", cmdString)

	// 执行命令
	return syscall.Exec(cmdPath, cmdParts, os.Environ())
}
