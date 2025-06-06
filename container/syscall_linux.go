//go:build linux
// +build linux

package container

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// setHostname 设置容器主机名
func setHostname(hostname string) error {
	return syscall.Sethostname([]byte(hostname))
}

// mountFilesystem 挂载文件系统
func mountFilesystem(source, target, fstype string, flags int, data string) error {
	return syscall.Mount(source, target, fstype, uintptr(flags), data)
}

// setNamespaceFlags 设置Linux特定的namespace隔离标志
func setNamespaceFlags(attr *syscall.SysProcAttr) {
	attr.Cloneflags = syscall.CLONE_NEWUTS | // 隔离主机名
		syscall.CLONE_NEWPID | // 隔离进程ID
		syscall.CLONE_NEWNS | // 隔离挂载点
		syscall.CLONE_NEWNET // 隔离网络
}

// setupContainerMounts 设置容器的挂载点
func setupContainerMounts(rootfs string) error {
	// 创建挂载点目录
	for _, dir := range []string{"/proc", "/sys", "/dev", "/dev/pts", "/tmp"} {
		path := filepath.Join(rootfs, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %v", path, err)
		}
	}

	// 挂载 proc 文件系统
	if err := mountFilesystem("proc", filepath.Join(rootfs, "/proc"), "proc", 0, ""); err != nil {
		return fmt.Errorf("挂载 proc 失败: %v", err)
	}

	// 挂载 sysfs 文件系统
	if err := mountFilesystem("sysfs", filepath.Join(rootfs, "/sys"), "sysfs", 0, ""); err != nil {
		return fmt.Errorf("挂载 sys 失败: %v", err)
	}

	// 挂载 tmpfs 到 /dev
	if err := mountFilesystem("tmpfs", filepath.Join(rootfs, "/dev"), "tmpfs", 0, ""); err != nil {
		return fmt.Errorf("挂载 dev 失败: %v", err)
	}

	// 确保 /dev/pts 目录存在后再挂载
	ptsDir := filepath.Join(rootfs, "/dev/pts")
	if err := os.MkdirAll(ptsDir, 0755); err != nil {
		return fmt.Errorf("创建 /dev/pts 目录失败: %v", err)
	}

	// 挂载 devpts
	// 使用更安全的挂载选项
	if err := mountFilesystem("devpts", filepath.Join(rootfs, "/dev/pts"), "devpts", 0, "newinstance,ptmxmode=0666,mode=0620"); err != nil {
		return fmt.Errorf("挂载 dev/pts 失败: %v", err)
	}

	// 创建一些基本设备节点
	devNull := filepath.Join(rootfs, "/dev/null")
	if err := unix.Mknod(devNull, unix.S_IFCHR|0666, int(unix.Mkdev(1, 3))); err != nil && !os.IsExist(err) {
		return fmt.Errorf("创建 /dev/null 失败: %v", err)
	}

	return nil
}
