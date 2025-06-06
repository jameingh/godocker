# GoDocker - 教学版Docker实现

GoDocker是一个使用Go语言实现的简易Docker，主要用于学习和理解Docker的底层原理。此项目通过直接使用Linux系统调用（而非Docker SDK），实现了容器的基本功能。

## 功能特性

GoDocker实现了以下核心功能：

1. **容器生命周期管理**
   - 创建、启动、停止和删除容器
   - 支持交互式运行和后台运行

2. **镜像管理**
   - 拉取镜像（简化版）
   - 列出本地镜像
   - 加载镜像到容器

3. **资源隔离**
   - 使用Linux namespace实现进程隔离
   - 使用cgroups实现资源限制（CPU/内存）
   - 基本的文件系统隔离

4. **网络管理**
   - 支持bridge/host/none网络模式
   - 容器之间的网络通信
   - 从主机访问容器网络

## 技术实现

GoDocker使用了以下Linux特性和技术：

- **Namespace**：进程隔离（PID、UTS、MNT、NET）
- **Cgroups**：资源限制（CPU、内存）
- **挂载点**：文件系统隔离
- **虚拟网络设备**：网络隔离和通信

## 项目结构

```
godocker/
├── cmd/           # 命令行接口
├── container/     # 容器管理核心
├── image/         # 镜像管理
├── network/       # 网络管理
├── resources/     # 资源限制
└── main.go        # 程序入口
```

## 使用方法

### 编译

```bash
go build -o godocker
```

### 运行容器

```bash
# 交互式运行容器
sudo ./godocker run -it ubuntu:latest /bin/bash

# 后台运行容器
sudo ./godocker run -d nginx:latest

# 限制资源运行容器
sudo ./godocker run -m 100m --cpuset 0,1 ubuntu:latest
```

### 镜像管理

```bash
# 拉取镜像
sudo ./godocker pull ubuntu:latest

# 列出镜像
sudo ./godocker images
```

### 容器管理

```bash
# 列出运行中的容器
sudo ./godocker ps

# 停止容器
sudo ./godocker stop <container-id>

# 删除容器
sudo ./godocker rm <container-id>
```

## 开发

```bash
docker-compose down
docker-compose up -d
docker exec -it godocker-dev sh
go build -o godocker

mkdir -p /var/lib/godocker/images
mkdir -p /sys/fs/cgroup/memory/godocker
mkdir -p /sys/fs/cgroup/cpu/godocker
mkdir -p /sys/fs/cgroup/cpuset/godocker

./godocker pull alpine:latest
./godocker run -it alpine:latest /bin/sh
```

## 教学价值

GoDocker适合学习以下内容：

1. Linux容器的基本原理和实现方式
2. Go语言如何与系统底层交互
3. Docker核心功能的工作流程
4. 容器化技术中的资源隔离机制

## 注意事项

- 此项目为教学目的设计，不适合生产环境使用
- 需要在Linux系统上运行，并需要root权限
- 部分功能是简化实现，不包含错误处理等生产级功能

## 许可证

MIT License 