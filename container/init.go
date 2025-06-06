package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
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

// 设置容器的挂载点
func setupContainerMounts(rootfs string) error {
	// 创建挂载点目录
	for _, dir := range []string{"/proc", "/sys", "/dev", "/dev/pts", "/tmp"} {
		path := rootfs + dir
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %v", path, err)
		}
	}

	// 挂载 proc 文件系统
	if err := mountFilesystem("proc", rootfs+"/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("挂载 proc 失败: %v", err)
	}

	// 挂载 sysfs 文件系统
	if err := mountFilesystem("sysfs", rootfs+"/sys", "sysfs", 0, ""); err != nil {
		return fmt.Errorf("挂载 sys 失败: %v", err)
	}

	// 挂载 tmpfs 到 /dev
	if err := mountFilesystem("tmpfs", rootfs+"/dev", "tmpfs", 0, ""); err != nil {
		return fmt.Errorf("挂载 dev 失败: %v", err)
	}

	// 确保 /dev/pts 目录存在后再挂载
	ptsDir := rootfs + "/dev/pts"
	if err := os.MkdirAll(ptsDir, 0755); err != nil {
		return fmt.Errorf("创建 /dev/pts 目录失败: %v", err)
	}

	// 挂载 devpts
	// 使用更安全的挂载选项
	if err := mountFilesystem("devpts", rootfs+"/dev/pts", "devpts", 0, "newinstance,ptmxmode=0666,mode=0620"); err != nil {
		return fmt.Errorf("挂载 dev/pts 失败: %v", err)
	}

	// 创建一些基本设备节点
	devNull := rootfs + "/dev/null"
	if err := unix.Mknod(devNull, unix.S_IFCHR|0666, int(unix.Mkdev(1, 3))); err != nil && !os.IsExist(err) {
		return fmt.Errorf("创建 /dev/null 失败: %v", err)
	}

	return nil
}

// 跨平台的文件系统挂载函数
func mountFilesystem(source, target, fstype string, flags int, data string) error {
	// 在Linux系统上使用syscall.Mount
	return syscall.Mount(source, target, fstype, uintptr(flags), data)
}
