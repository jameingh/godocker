package cmd

import (
	"fmt"

	"github.com/akm/godocker/container"
	"github.com/akm/godocker/image"
)

// Ps 列出正在运行的容器
func Ps() {
	containers, err := container.ListContainers()
	if err != nil {
		fmt.Printf("获取容器列表失败: %v\n", err)
		return
	}

	// 打印容器列表表头
	fmt.Printf("%-12s %-15s %-20s %-10s %-20s\n", "容器ID", "镜像", "命令", "状态", "创建时间")
	fmt.Println("----------------------------------------------------------------------")

	// 打印容器信息
	for _, c := range containers {
		cmd := ""
		if len(c.Command) > 0 {
			cmd = c.Command[0]
			if len(c.Command) > 1 {
				cmd += "..."
			}
		}
		fmt.Printf("%-12s %-15s %-20s %-10s %-20s\n",
			c.ID[:12],
			c.Image,
			cmd,
			c.Status,
			c.CreateTime.Format("2006-01-02 15:04:05"))
	}
}

// Images 列出本地镜像
func Images() {
	images, err := image.ListImages()
	if err != nil {
		fmt.Printf("获取镜像列表失败: %v\n", err)
		return
	}

	// 打印镜像列表表头
	fmt.Printf("%-20s %-15s %-20s %-15s\n", "镜像ID", "仓库", "标签", "大小")
	fmt.Println("----------------------------------------------------------------------")

	// 打印镜像信息
	for _, img := range images {
		fmt.Printf("%-20s %-15s %-20s %-15s\n",
			img.ID[:12],
			img.Repository,
			img.Tag,
			formatSize(img.Size))
	}
}

// Pull 拉取镜像
func Pull(imageName string) {
	fmt.Printf("开始拉取镜像: %s\n", imageName)

	if err := image.PullImage(imageName); err != nil {
		fmt.Printf("拉取镜像失败: %v\n", err)
		return
	}

	fmt.Printf("成功拉取镜像: %s\n", imageName)
}

// Stop 停止容器
func Stop(containerID string) {
	if err := container.StopContainer(containerID); err != nil {
		fmt.Printf("停止容器失败: %v\n", err)
		return
	}

	fmt.Printf("容器 %s 已停止\n", containerID)
}

// Remove 删除容器
func Remove(containerID string) {
	if err := container.RemoveContainer(containerID); err != nil {
		fmt.Printf("删除容器失败: %v\n", err)
		return
	}

	fmt.Printf("容器 %s 已删除\n", containerID)
}

// 格式化文件大小
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
