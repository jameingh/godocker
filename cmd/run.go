package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/akm/godocker/container"
	"github.com/akm/godocker/resources"
)

// Run 实现容器的运行命令
func Run(args []string) {
	// 解析run命令的参数
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	// 定义run命令参数
	tty := runCmd.Bool("it", false, "启用交互式终端")
	memory := runCmd.String("m", "", "内存限制 (如 '100m')")
	cpuShare := runCmd.String("cpuset", "", "CPU核心使用限制 (如 '0,1')")
	volume := runCmd.String("v", "", "数据卷映射 (如 '/host:/container')")
	name := runCmd.String("name", "", "指定容器名称")
	network := runCmd.String("net", "bridge", "指定网络模式")
	detach := runCmd.Bool("d", false, "后台运行容器")

	if err := runCmd.Parse(args); err != nil {
		fmt.Println("解析参数错误:", err)
		os.Exit(1)
	}

	// 获取剩余参数，第一个是镜像名，后面是要执行的命令
	cmdArgs := runCmd.Args()
	if len(cmdArgs) < 1 {
		fmt.Println("请指定容器镜像，例如: godocker run ubuntu:latest /bin/bash")
		os.Exit(1)
	}

	imageName := cmdArgs[0]

	// 构建容器配置
	containerConfig := &container.Config{
		Name:     *name,
		Image:    imageName,
		Command:  []string{},
		Tty:      *tty,
		Detach:   *detach,
		Network:  *network,
		Volumes:  parseVolumes(*volume),
		Resource: parseResourceConfig(*memory, *cpuShare),
	}

	// 处理要执行的命令
	if len(cmdArgs) > 1 {
		containerConfig.Command = cmdArgs[1:]
	} else {
		// 如果没有指定命令，使用默认的shell
		containerConfig.Command = []string{"/bin/sh"}
	}

	// 运行容器
	containerId, err := container.NewContainer(containerConfig)
	if err != nil {
		fmt.Printf("创建容器失败: %v\n", err)
		os.Exit(1)
	}

	// 如果是交互式模式，等待容器运行结束
	if !*detach {
		if err := container.WaitContainer(containerId); err != nil {
			fmt.Printf("等待容器结束失败: %v\n", err)
		}
	} else {
		fmt.Printf("容器已在后台启动，ID: %s\n", containerId)
	}
}

// 解析卷映射参数
func parseVolumes(volumeStr string) []container.VolumeMapping {
	if volumeStr == "" {
		return nil
	}

	volumeMappings := []container.VolumeMapping{}
	volumes := strings.Split(volumeStr, ",")

	for _, v := range volumes {
		parts := strings.Split(v, ":")
		if len(parts) == 2 {
			hostPath, _ := filepath.Abs(parts[0])
			containerPath := parts[1]

			volumeMappings = append(volumeMappings, container.VolumeMapping{
				HostPath:      hostPath,
				ContainerPath: containerPath,
			})
		}
	}

	return volumeMappings
}

// 解析资源限制参数
func parseResourceConfig(memoryLimit, cpuSet string) resources.ResourceConfig {
	config := resources.ResourceConfig{}

	if memoryLimit != "" {
		config.MemoryLimit = memoryLimit
	}

	if cpuSet != "" {
		config.CpuSet = cpuSet
	}

	return config
}
