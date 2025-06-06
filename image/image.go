package image

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ImageInfo 镜像信息
type ImageInfo struct {
	ID         string    // 镜像ID
	Repository string    // 仓库名
	Tag        string    // 标签
	Size       int64     // 大小（字节）
	CreatedAt  time.Time // 创建时间
	Layers     []string  // 层ID列表
}

const (
	// 镜像存储根目录
	DefaultImageRoot = "/var/lib/godocker/images"
)

// PullImage 拉取镜像
func PullImage(imageName string) error {
	// 解析镜像名称和标签
	repository, tag := parseImageName(imageName)
	if tag == "" {
		tag = "latest"
	}

	fmt.Printf("开始拉取镜像 %s:%s\n", repository, tag)

	// 创建镜像存储目录
	imageRoot := filepath.Join(DefaultImageRoot, repository, tag)
	if err := os.MkdirAll(imageRoot, 0755); err != nil {
		return fmt.Errorf("创建镜像目录失败: %v", err)
	}

	// 在实际实现中，这里应该使用Docker Registry API拉取镜像
	// 简化示例使用 tar 命令模拟拉取过程
	// 这部分简化处理，实际拉取需要实现Docker Registry HTTP API交互
	if err := simulatePullImage(repository, tag, imageRoot); err != nil {
		return err
	}

	// 创建镜像元数据
	imageId := generateImageId(repository, tag)
	imageInfo := &ImageInfo{
		ID:         imageId,
		Repository: repository,
		Tag:        tag,
		Size:       calculateImageSize(imageRoot),
		CreatedAt:  time.Now(),
		Layers:     []string{imageId}, // 简化处理，实际应该有多层
	}

	// 保存镜像元数据
	metadataFile := filepath.Join(imageRoot, "metadata.json")
	file, err := os.Create(metadataFile)
	if err != nil {
		return fmt.Errorf("创建元数据文件失败: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(imageInfo); err != nil {
		return fmt.Errorf("保存元数据失败: %v", err)
	}

	fmt.Printf("镜像 %s:%s 已成功拉取，ID: %s\n", repository, tag, imageId[:12])
	return nil
}

// ListImages 列出本地镜像
func ListImages() ([]*ImageInfo, error) {
	var images []*ImageInfo

	// 遍历镜像目录
	if err := filepath.Walk(DefaultImageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只查找metadata.json文件
		if !info.IsDir() && filepath.Base(path) == "metadata.json" {
			// 读取镜像元数据
			file, err := os.Open(path)
			if err != nil {
				fmt.Printf("警告: 无法打开元数据文件 %s: %v\n", path, err)
				return nil
			}
			defer file.Close()

			var imageInfo ImageInfo
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(&imageInfo); err != nil {
				fmt.Printf("警告: 解析元数据文件 %s 失败: %v\n", path, err)
				return nil
			}

			images = append(images, &imageInfo)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("遍历镜像目录失败: %v", err)
	}

	return images, nil
}

// GetImagePath 获取镜像文件系统路径
func GetImagePath(imageName string) (string, error) {
	repository, tag := parseImageName(imageName)
	if tag == "" {
		tag = "latest"
	}

	// 检查镜像是否存在
	imageRoot := filepath.Join(DefaultImageRoot, repository, tag)
	if _, err := os.Stat(imageRoot); os.IsNotExist(err) {
		return "", fmt.Errorf("镜像 %s:%s 不存在", repository, tag)
	}

	// 返回镜像根路径
	return imageRoot, nil
}

// 解析镜像名称
func parseImageName(imageName string) (string, string) {
	parts := strings.Split(imageName, ":")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// 生成镜像ID
func generateImageId(repository, tag string) string {
	// 简化处理，实际应该基于镜像内容生成哈希
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%x", timestamp)
}

// 计算镜像大小
func calculateImageSize(imageRoot string) int64 {
	var size int64

	filepath.Walk(imageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size
}

// 模拟拉取镜像（实际实现中应使用Docker Registry API）
func simulatePullImage(repository, tag, imageRoot string) error {
	// 创建示例rootfs
	rootfsDir := filepath.Join(imageRoot, "rootfs")
	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		return fmt.Errorf("创建rootfs目录失败: %v", err)
	}

	// 创建必要的目录
	for _, dir := range []string{"/bin", "/etc", "/lib", "/usr", "/var", "/proc", "/sys", "/tmp"} {
		if err := os.MkdirAll(filepath.Join(rootfsDir, dir), 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %v", dir, err)
		}
	}

	// 创建一个示例文件
	helloFile := filepath.Join(rootfsDir, "hello.txt")
	if err := os.WriteFile(helloFile, []byte(fmt.Sprintf("Hello from %s:%s", repository, tag)), 0644); err != nil {
		return fmt.Errorf("创建示例文件失败: %v", err)
	}

	// 模拟层信息
	layersDir := filepath.Join(imageRoot, "layers")
	if err := os.MkdirAll(layersDir, 0755); err != nil {
		return fmt.Errorf("创建层目录失败: %v", err)
	}

	// 模拟下载进度
	for i := 1; i <= 5; i++ {
		fmt.Printf("拉取镜像层 %d/5: %d%%\n", i, i*20)
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("下载完成，正在解压镜像...")
	time.Sleep(500 * time.Millisecond)

	return nil
}
