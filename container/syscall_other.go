//go:build !linux
// +build !linux

package container

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// setHostname 设置容器主机名（非Linux平台的模拟实现）
func setHostname(hostname string) error {
	fmt.Printf("模拟设置主机名: %s\n", hostname)
	return nil
}

// mountFilesystem 挂载文件系统（非Linux平台的模拟实现）
func mountFilesystem(source, target, fstype string, flags int, data string) error {
	fmt.Printf("模拟挂载 %s 到 %s (类型: %s)\n", source, target, fstype)
	return nil
}

// setNamespaceFlags 设置namespace隔离标志（非Linux平台的模拟实现）
func setNamespaceFlags(attr *syscall.SysProcAttr) {
	// 在非Linux平台上不做任何操作
	fmt.Println("模拟设置namespace隔离（在非Linux平台上不可用）")
}

// setupContainerMounts 设置容器的挂载点（非Linux平台的模拟实现）
func setupContainerMounts(rootfs string) error {
	// 创建挂载点目录
	for _, dir := range []string{"/proc", "/sys", "/dev", "/dev/pts", "/tmp"} {
		path := filepath.Join(rootfs, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %v", path, err)
		}
	}

	// 模拟挂载
	fmt.Println("模拟挂载容器文件系统...")

	// 挂载 proc 文件系统
	mountFilesystem("proc", filepath.Join(rootfs, "/proc"), "proc", 0, "")

	// 挂载 sysfs 文件系统
	mountFilesystem("sysfs", filepath.Join(rootfs, "/sys"), "sysfs", 0, "")

	// 挂载 tmpfs 到 /dev
	mountFilesystem("tmpfs", filepath.Join(rootfs, "/dev"), "tmpfs", 0, "")

	// 挂载 devpts
	mountFilesystem("devpts", filepath.Join(rootfs, "/dev/pts"), "devpts", 0, "")

	// 创建一些基本设备节点
	devNull := filepath.Join(rootfs, "/dev/null")
	fmt.Printf("模拟创建设备节点: %s\n", devNull)

	return nil
}
