package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/akm/godocker/network"
	"github.com/akm/godocker/resources"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
	
)

// Config 容器配置
type Config struct {
	Name     string                   // 容器名称
	Image    string                   // 镜像名称
	Command  []string                 // 容器启动命令
	Tty      bool                     // 是否启用tty
	Detach   bool                     // 是否后台运行
	Network  string                   // 网络模式
	Volumes  []VolumeMapping          // 卷映射
	Resource resources.ResourceConfig // 资源限制
}

// VolumeMapping 卷映射
type VolumeMapping struct {
	HostPath      string // 主机路径
	ContainerPath string // 容器内路径
}

// ContainerInfo 容器信息
type ContainerInfo struct {
	ID         string    // 容器ID
	Name       string    // 容器名称
	Pid        int       // 容器主进程ID
	Image      string    // 容器镜像
	Command    []string  // 容器启动命令
	Status     string    // 容器状态
	CreateTime time.Time // 容器创建时间
	Config     Config    // 容器配置
}

const (
	DefaultContainerRoot = "/var/lib/godocker"
	StatusRunning        = "运行中"
	StatusStopped        = "已停止"
)

// 运行中的容器映射表
var runningContainers = make(map[string]*ContainerInfo)

// NewContainer 创建并启动一个新的容器
func NewContainer(config *Config) (string, error) {
	// 生成唯一的容器ID
	containerId := generateContainerId()

	// 如果指定了容器名称，检查是否重复
	if config.Name != "" {
		// 检查同名容器是否存在
		for _, c := range runningContainers {
			if c.Name == config.Name {
				return "", fmt.Errorf("已存在同名容器: %s", config.Name)
			}
		}
	} else {
		// 如果未指定名称，使用ID前12位作为名称
		config.Name = containerId[:12]
	}

	// 准备容器文件系统
	containerRoot, err := prepareRootfs(containerId, config.Image)
	if err != nil {
		return "", fmt.Errorf("准备容器文件系统失败: %v", err)
	}

	// 创建容器记录
	container := &ContainerInfo{
		ID:         containerId,
		Name:       config.Name,
		Image:      config.Image,
		Command:    config.Command,
		Status:     StatusRunning,
		CreateTime: time.Now(),
		Config:     *config,
	}

	// 启动容器进程
	process, err := startContainer(container, containerRoot)
	if err != nil {
		return "", fmt.Errorf("启动容器进程失败: %v", err)
	}

	// 记录进程ID
	container.Pid = process.Pid

	// 保存容器信息
	runningContainers[containerId] = container

	// 应用资源限制
	if err := resources.ApplyResourceLimits(process.Pid, config.Resource); err != nil {
		fmt.Printf("警告: 应用资源限制失败: %v\n", err)
	}

	if config.Network != "" && config.Network != "none" {
		_, err := network.SetupNetwork(config.Network, containerId, container.Pid)
		if err != nil {
			fmt.Printf("容器网络配置失败: %v\n", err)
		}
	}
	return containerId, nil
}

// StopContainer 停止容器
func StopContainer(containerId string) error {
	container, exists := runningContainers[containerId]
	if !exists {
		return fmt.Errorf("找不到容器: %s", containerId)
	}

	// 如果容器已停止，直接返回
	if container.Status == StatusStopped {
		return nil
	}

	// 向容器主进程发送SIGTERM信号
	process, err := os.FindProcess(container.Pid)
	if err != nil {
		return fmt.Errorf("查找容器进程失败: %v", err)
	}

	// 先尝试优雅停止
	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("发送SIGTERM信号失败，尝试强制终止: %v\n", err)
		// 如果SIGTERM失败，强制终止
		if err := process.Kill(); err != nil {
			return fmt.Errorf("终止容器进程失败: %v", err)
		}
	}

	// 更新容器状态
	container.Status = StatusStopped

	return nil
}

// RemoveContainer 删除容器
func RemoveContainer(containerId string) error {
	container, exists := runningContainers[containerId]
	if !exists {
		return fmt.Errorf("找不到容器: %s", containerId)
	}

	// 如果容器仍在运行，先停止
	if container.Status == StatusRunning {
		if err := StopContainer(containerId); err != nil {
			return fmt.Errorf("停止容器失败: %v", err)
		}
	}

	// 清理容器文件系统
	containerRoot := filepath.Join(DefaultContainerRoot, containerId)
	if err := os.RemoveAll(containerRoot); err != nil {
		fmt.Printf("警告: 清理容器文件系统失败: %v\n", err)
	}

	// 从运行列表中删除
	delete(runningContainers, containerId)

	return nil
}

// ListContainers 列出所有容器
func ListContainers() ([]*ContainerInfo, error) {
	result := make([]*ContainerInfo, 0, len(runningContainers))

	for _, container := range runningContainers {
		result = append(result, container)
	}

	return result, nil
}

// WaitContainer 等待容器执行结束
func WaitContainer(containerId string) error {
	container, exists := runningContainers[containerId]
	if !exists {
		return fmt.Errorf("找不到容器: %s", containerId)
	}

	// 如果容器已停止，直接返回
	if container.Status == StatusStopped {
		return nil
	}

	// 查找容器进程
	process, err := os.FindProcess(container.Pid)
	if err != nil {
		return fmt.Errorf("查找容器进程失败: %v", err)
	}

	// 等待进程结束
	state, err := process.Wait()
	if err != nil {
		return fmt.Errorf("等待容器进程失败: %v", err)
	}

	// 更新容器状态
	container.Status = StatusStopped

	fmt.Printf("容器 %s 已退出，状态码: %d\n", containerId[:12], state.ExitCode())

	return nil
}

// 生成唯一的容器ID
func generateContainerId() string {
	return uuid.New().String()
}

// 准备容器文件系统
func prepareRootfs(containerId, imageName string) (string, error) {
	// 容器根目录
	containerRoot := filepath.Join(DefaultContainerRoot, containerId)

	// 创建容器目录
	if err := os.MkdirAll(containerRoot, 0755); err != nil {
		return "", err
	}

	// TODO: 实际解压镜像到该目录，这里简化为使用主机的文件系统
	fmt.Printf("准备容器文件系统: %s (使用镜像: %s)\n", containerRoot, imageName)

	// 在实际实现中，这里应该解压镜像到containerRoot目录
	// 简化示例中，我们创建一个简单的文件表示rootfs已准备
	marker := filepath.Join(containerRoot, ".rootfs_ready")
	if err := os.WriteFile(marker, []byte(imageName), 0644); err != nil {
		return "", err
	}

	return containerRoot, nil
}

// 启动容器进程
func startContainer(container *ContainerInfo, rootfs string) (*os.Process, error) {
	// 设置命令
	cmd := exec.Command("/proc/self/exe", "init")

	// 设置容器进程的namespace隔离
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	// 平台特定的namespace设置
	setNamespaceFlags(cmd.SysProcAttr)

	// 设置环境变量
	cmd.Env = []string{
		"PATH=/bin:/usr/bin:/sbin:/usr/sbin",
		"TERM=xterm",
		"CONTAINER_ID=" + container.ID,
		"CONTAINER_NAME=" + container.Name,
	}

	// 传递容器配置
	cmd.Env = append(cmd.Env,
		"CONTAINER_CMD="+strings.Join(container.Command, " "),
		"CONTAINER_ROOTFS="+rootfs,
	)

	// 设置标准输入输出
	if container.Config.Tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	fmt.Printf("容器进程已启动，PID: %d\n", cmd.Process.Pid)

	return cmd.Process, nil
}

// 设置挂载点
func setupMounts(rootfs string) error {
	// 实现文件系统挂载
	// 这里需要挂载proc、sys等文件系统

	// 示例: 挂载proc文件系统
	procPath := filepath.Join(rootfs, "proc")
	if err := os.MkdirAll(procPath, 0755); err != nil {
		return err
	}

	if err := unix.Mount("proc", procPath, "proc", 0, ""); err != nil {
		return err
	}

	return nil
}

// 容器初始化函数，会在容器命名空间中运行
func containerInitProcess() error {
	// 获取环境变量中的容器配置
	rootfs := os.Getenv("CONTAINER_ROOTFS")
	cmdString := os.Getenv("CONTAINER_CMD")

	if rootfs == "" || cmdString == "" {
		return errors.New("缺少必要的容器环境配置")
	}

	// 设置主机名
	if err := unix.Sethostname([]byte(os.Getenv("CONTAINER_NAME"))); err != nil {
		return err
	}

	// 设置挂载点
	if err := setupMounts(rootfs); err != nil {
		return err
	}

	// 切换根目录
	if err := unix.Chroot(rootfs); err != nil {
		return err
	}

	// 切换工作目录
	if err := os.Chdir("/"); err != nil {
		return err
	}

	// 解析命令
	cmdParts := strings.Split(cmdString, " ")
	if len(cmdParts) == 0 {
		return errors.New("无效的容器命令")
	}

	// 查找命令路径
	cmdPath, err := exec.LookPath(cmdParts[0])
	if err != nil {
		return err
	}

	// 执行命令
	return syscall.Exec(cmdPath, cmdParts, os.Environ())
}
