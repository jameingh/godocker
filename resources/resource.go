package resources

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ResourceConfig 定义资源限制配置
type ResourceConfig struct {
	MemoryLimit string // 内存限制，例如 "100m"
	CpuSet      string // CPU核心设置，例如 "0,1"
	CpuShare    int    // CPU共享权重
}

const (
	// cgroup挂载点路径
	cgroupMemoryPath = "/sys/fs/cgroup/memory"
	cgroupCpuPath    = "/sys/fs/cgroup/cpu"
	cgroupCpusetPath = "/sys/fs/cgroup/cpuset"
)

// ApplyResourceLimits 应用资源限制到指定进程
func ApplyResourceLimits(pid int, config ResourceConfig) error {
	// 如果没有设置任何资源限制，直接返回
	if config.MemoryLimit == "" && config.CpuSet == "" && config.CpuShare == 0 {
		return nil
	}

	// 创建cgroup子系统
	cgroupName := "godocker-" + strconv.Itoa(pid)

	// 应用内存限制
	if config.MemoryLimit != "" {
		if err := setupMemoryLimit(cgroupName, pid, config.MemoryLimit); err != nil {
			return fmt.Errorf("设置内存限制失败: %v", err)
		}
	}

	// 应用CPU核心限制
	if config.CpuSet != "" {
		if err := setupCpuSet(cgroupName, pid, config.CpuSet); err != nil {
			return fmt.Errorf("设置CPU核心限制失败: %v", err)
		}
	}

	// 应用CPU共享限制
	if config.CpuShare > 0 {
		if err := setupCpuShare(cgroupName, pid, config.CpuShare); err != nil {
			return fmt.Errorf("设置CPU共享限制失败: %v", err)
		}
	}

	return nil
}

// 设置内存限制
func setupMemoryLimit(cgroupName string, pid int, memoryLimit string) error {
	// 转换内存限制为字节
	memoryBytes, err := parseMemoryLimit(memoryLimit)
	if err != nil {
		return err
	}

	// 创建内存cgroup子系统
	memoryPath := filepath.Join(cgroupMemoryPath, cgroupName)
	if err := os.MkdirAll(memoryPath, 0755); err != nil {
		return err
	}

	// 设置内存限制
	if err := ioutil.WriteFile(
		filepath.Join(memoryPath, "memory.limit_in_bytes"),
		[]byte(strconv.FormatInt(memoryBytes, 10)),
		0644); err != nil {
		return err
	}

	// 禁用交换内存
	if err := ioutil.WriteFile(
		filepath.Join(memoryPath, "memory.swappiness"),
		[]byte("0"),
		0644); err != nil {
		return err
	}

	// 将进程加入到cgroup
	if err := ioutil.WriteFile(
		filepath.Join(memoryPath, "tasks"),
		[]byte(strconv.Itoa(pid)),
		0644); err != nil {
		return err
	}

	return nil
}

// 设置CPU核心限制
func setupCpuSet(cgroupName string, pid int, cpuSet string) error {
	// 创建cpuset cgroup子系统
	cpusetPath := filepath.Join(cgroupCpusetPath, cgroupName)
	if err := os.MkdirAll(cpusetPath, 0755); err != nil {
		return err
	}

	// 设置CPU核心
	if err := ioutil.WriteFile(
		filepath.Join(cpusetPath, "cpuset.cpus"),
		[]byte(cpuSet),
		0644); err != nil {
		return err
	}

	// 设置内存节点
	// 在实际环境中，应该根据系统的NUMA节点配置来设置
	if err := ioutil.WriteFile(
		filepath.Join(cpusetPath, "cpuset.mems"),
		[]byte("0"),
		0644); err != nil {
		return err
	}

	// 将进程加入到cgroup
	if err := ioutil.WriteFile(
		filepath.Join(cpusetPath, "tasks"),
		[]byte(strconv.Itoa(pid)),
		0644); err != nil {
		return err
	}

	return nil
}

// 设置CPU共享限制
func setupCpuShare(cgroupName string, pid int, cpuShare int) error {
	// 创建cpu cgroup子系统
	cpuPath := filepath.Join(cgroupCpuPath, cgroupName)
	if err := os.MkdirAll(cpuPath, 0755); err != nil {
		return err
	}

	// 设置CPU共享值
	if err := ioutil.WriteFile(
		filepath.Join(cpuPath, "cpu.shares"),
		[]byte(strconv.Itoa(cpuShare)),
		0644); err != nil {
		return err
	}

	// 将进程加入到cgroup
	if err := ioutil.WriteFile(
		filepath.Join(cpuPath, "tasks"),
		[]byte(strconv.Itoa(pid)),
		0644); err != nil {
		return err
	}

	return nil
}

// parseMemoryLimit 将内存限制字符串转换为字节数
func parseMemoryLimit(memoryLimit string) (int64, error) {
	memoryLimit = strings.ToLower(memoryLimit)
	var multiplier int64 = 1

	if strings.HasSuffix(memoryLimit, "k") {
		multiplier = 1024
		memoryLimit = strings.TrimSuffix(memoryLimit, "k")
	} else if strings.HasSuffix(memoryLimit, "m") {
		multiplier = 1024 * 1024
		memoryLimit = strings.TrimSuffix(memoryLimit, "m")
	} else if strings.HasSuffix(memoryLimit, "g") {
		multiplier = 1024 * 1024 * 1024
		memoryLimit = strings.TrimSuffix(memoryLimit, "g")
	}

	value, err := strconv.ParseInt(memoryLimit, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的内存限制格式: %s", memoryLimit)
	}

	return value * multiplier, nil
}
