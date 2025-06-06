package network

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// NetworkConfig 网络配置
type NetworkConfig struct {
	Mode      string // 网络模式：bridge, host, none
	IPAddress string // 容器IP地址
	Gateway   string // 网关地址
	Subnet    string // 子网掩码
	MacAddr   string // MAC地址
}

const (
	// 网络模式
	BridgeMode = "bridge"
	HostMode   = "host"
	NoneMode   = "none"

	// 默认的网桥设备名
	DefaultBridge = "godocker0"

	// 默认的网络配置
	DefaultSubnet   = "172.17.0.0/16"
	DefaultGateway  = "172.17.0.1"
	DefaultIPPrefix = "172.17.0."
)

// SetupNetwork 为容器配置网络
func SetupNetwork(netMode string, containerID string, pid int) (*NetworkConfig, error) {
	// 创建网络配置
	netConfig := &NetworkConfig{
		Mode: netMode,
	}

	// 根据网络模式进行配置
	switch netMode {
	case BridgeMode:
		// 创建网桥（如果不存在）
		if err := setupBridge(); err != nil {
			return nil, fmt.Errorf("设置网桥失败: %v", err)
		}

		// 创建虚拟网卡对
		vethName := "veth-" + containerID[:8]
		peerName := "eth0"

		// 分配IP地址
		ipAddr := allocateIP()
		netConfig.IPAddress = ipAddr
		netConfig.Gateway = DefaultGateway
		netConfig.Subnet = DefaultSubnet

		// 创建虚拟网卡
		if err := createVethPair(vethName, peerName); err != nil {
			return nil, fmt.Errorf("创建虚拟网卡对失败: %v", err)
		}

		// 将网卡移入容器命名空间
		if err := setupContainerNetns(vethName, peerName, pid, ipAddr); err != nil {
			return nil, fmt.Errorf("设置容器网络命名空间失败: %v", err)
		}

		// 连接网卡到网桥
		if err := connectVethToBridge(vethName, DefaultBridge); err != nil {
			return nil, fmt.Errorf("连接网卡到网桥失败: %v", err)
		}

		// 设置网络转发和NAT
		if err := setupNAT(DefaultBridge, DefaultSubnet); err != nil {
			return nil, fmt.Errorf("设置NAT失败: %v", err)
		}

	case HostMode:
		// 直接使用主机网络，不需要额外配置
		fmt.Println("容器使用主机网络模式")

	case NoneMode:
		// 不配置网络
		fmt.Println("容器未配置网络")

	default:
		return nil, fmt.Errorf("不支持的网络模式: %s", netMode)
	}

	return netConfig, nil
}

// 设置网桥
func setupBridge() error {
	// 检查网桥是否已存在
	if exists, _ := deviceExists(DefaultBridge); exists {
		fmt.Printf("网桥 %s 已存在\n", DefaultBridge)
		return nil
	}

	// 创建网桥
	fmt.Printf("创建网桥 %s\n", DefaultBridge)

	// 创建网桥设备
	if _, err := exec.Command("ip", "link", "add", "name", DefaultBridge, "type", "bridge").Output(); err != nil {
		return fmt.Errorf("创建网桥失败: %v", err)
	}

	// 设置网桥IP
	if _, err := exec.Command("ip", "addr", "add", DefaultGateway+"/16", "dev", DefaultBridge).Output(); err != nil {
		return fmt.Errorf("设置网桥IP失败: %v", err)
	}

	// 启动网桥
	if _, err := exec.Command("ip", "link", "set", "dev", DefaultBridge, "up").Output(); err != nil {
		return fmt.Errorf("启动网桥失败: %v", err)
	}

	return nil
}

// 创建虚拟网卡对
func createVethPair(vethName, peerName string) error {
	// 创建veth对
	if _, err := exec.Command("ip", "link", "add", vethName, "type", "veth", "peer", "name", peerName).Output(); err != nil {
		return fmt.Errorf("创建veth对失败: %v", err)
	}

	// 启动veth
	if _, err := exec.Command("ip", "link", "set", "dev", vethName, "up").Output(); err != nil {
		return fmt.Errorf("启动veth失败: %v", err)
	}

	return nil
}

// 设置容器网络命名空间
func setupContainerNetns(vethName, peerName string, pid int, ipAddr string) error {
	// 获取容器网络命名空间路径（这里仅作记录，实际使用pid直接操作）
	_ = filepath.Join("/proc", strconv.Itoa(pid), "ns", "net")

	// 将peer移动到容器命名空间
	if _, err := exec.Command("ip", "link", "set", peerName, "netns", strconv.Itoa(pid)).Output(); err != nil {
		return fmt.Errorf("移动peer到容器命名空间失败: %v", err)
	}

	// 在容器命名空间中设置peer
	// 设置网卡名称为eth0
	if _, err := exec.Command("nsenter", "-t", strconv.Itoa(pid), "-n", "ip", "link", "set", "dev", peerName, "name", "eth0").Output(); err != nil {
		return fmt.Errorf("重命名网卡失败: %v", err)
	}

	// 设置IP地址
	if _, err := exec.Command("nsenter", "-t", strconv.Itoa(pid), "-n", "ip", "addr", "add", ipAddr+"/16", "dev", "eth0").Output(); err != nil {
		return fmt.Errorf("设置IP地址失败: %v", err)
	}

	// 启动网卡
	if _, err := exec.Command("nsenter", "-t", strconv.Itoa(pid), "-n", "ip", "link", "set", "dev", "eth0", "up").Output(); err != nil {
		return fmt.Errorf("启动网卡失败: %v", err)
	}

	// 设置默认路由
	if _, err := exec.Command("nsenter", "-t", strconv.Itoa(pid), "-n", "ip", "route", "add", "default", "via", DefaultGateway).Output(); err != nil {
		return fmt.Errorf("设置默认路由失败: %v", err)
	}

	return nil
}

// 连接网卡到网桥
func connectVethToBridge(vethName, bridge string) error {
	if _, err := exec.Command("ip", "link", "set", "dev", vethName, "master", bridge).Output(); err != nil {
		return fmt.Errorf("连接veth到网桥失败: %v", err)
	}
	return nil
}

// 设置NAT
func setupNAT(bridge, subnet string) error {
	// 启用IP转发
	if _, err := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Output(); err != nil {
		return fmt.Errorf("启用IP转发失败: %v", err)
	}

	// 添加NAT规则
	if _, err := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", subnet, "!", "-o", bridge, "-j", "MASQUERADE").Output(); err != nil {
		// 这里不返回错误，因为规则可能已存在
		fmt.Printf("警告: 添加NAT规则可能失败: %v\n", err)
	}

	return nil
}

// 检查设备是否存在
func deviceExists(name string) (bool, error) {
	_, err := net.InterfaceByName(name)
	if err != nil {
		if strings.Contains(err.Error(), "no such network interface") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// 分配IP地址
func allocateIP() string {
	// 简化实现，实际应该有更复杂的IP地址分配机制
	// 这里简单返回一个固定IP段的随机IP
	// 在实际实现中，应该维护已分配IP的列表

	// 简单起见，使用进程ID的后两位作为IP地址的最后部分
	lastOctet := os.Getpid() % 254
	if lastOctet < 2 {
		lastOctet = 100 // 避免使用0和1
	}

	return DefaultIPPrefix + strconv.Itoa(lastOctet)
}
