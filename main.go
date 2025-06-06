package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/akm/godocker/cmd"
	"github.com/akm/godocker/container"
)

func main() {
	// 设置命令行子命令
	flag.Parse()
	args := flag.Args()

	// 特殊处理init命令，该命令仅在新容器内部由godocker自己调用
	if len(args) > 0 && args[0] == "init" {
		runInit()
		return
	}

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	// 处理其他命令
	switch args[0] {
	case "run":
		cmd.Run(args[1:])
	case "ps":
		cmd.Ps()
	case "images":
		cmd.Images()
	case "pull":
		if len(args) < 2 {
			fmt.Println("请指定要拉取的镜像，例如: godocker pull ubuntu:latest")
			os.Exit(1)
		}
		cmd.Pull(args[1])
	case "stop":
		if len(args) < 2 {
			fmt.Println("请指定要停止的容器ID，例如: godocker stop [container-id]")
			os.Exit(1)
		}
		cmd.Stop(args[1])
	case "rm":
		if len(args) < 2 {
			fmt.Println("请指定要删除的容器ID，例如: godocker rm [container-id]")
			os.Exit(1)
		}
		cmd.Remove(args[1])
	default:
		fmt.Printf("未知命令: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

// runInit 在容器内部执行初始化
func runInit() {
	if err := container.InitContainer(); err != nil {
		fmt.Printf("容器初始化失败: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("GoDocker - 用于学习的简易Docker实现")
	fmt.Println("\n用法:")
	fmt.Println("  godocker [命令] [参数]")
	fmt.Println("\n可用命令:")
	fmt.Println("  run      运行一个容器")
	fmt.Println("  ps       列出正在运行的容器")
	fmt.Println("  images   列出本地镜像")
	fmt.Println("  pull     拉取镜像")
	fmt.Println("  stop     停止容器")
	fmt.Println("  rm       删除容器")
	fmt.Println("\n示例:")
	fmt.Println("  godocker run -it ubuntu:latest /bin/bash")
}
