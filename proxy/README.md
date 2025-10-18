# RTSP代理服务器

RTSP代理服务器是一个高性能的RTSP流媒体代理，支持RTSP消息转发、媒体流代理、负载均衡等功能。

## 功能特性

- ✅ **RTSP消息代理**: 完整的RTSP协议消息转发
- ✅ **媒体流代理**: 支持RTP/RTCP和TCP Interleaved媒体流代理
- ✅ **负载均衡**: 支持多个上游服务器和路由配置
- ✅ **会话管理**: 完整的代理会话生命周期管理
- ✅ **统计监控**: 详细的代理性能和统计信息
- ✅ **配置管理**: 灵活的配置文件和命令行参数
- ✅ **高并发**: 支持大量并发连接和会话

## 项目结构

```
proxy/
├── main.go           # 主程序入口
├── proxy.go          # 代理核心实现
├── rtsp_io.go        # RTSP消息读写器
├── media_proxy.go    # 媒体流代理实现
├── config.go         # 配置管理
├── proxy_examples.go # 示例代码
└── README.md         # 说明文档
```

## 快速开始

### 1. 编译代理程序

```bash
cd proxy
go build -o rtsp-proxy main.go proxy.go rtsp_io.go media_proxy.go config.go
```

### 2. 运行代理服务器

```bash
# 使用默认配置
./rtsp-proxy

# 指定监听地址和上游服务器
./rtsp-proxy -listen 0.0.0.0:8555 -upstream rtsp://localhost:8554/test

# 使用配置文件
./rtsp-proxy -config proxy.json
```

### 3. 命令行参数

```bash
-listen string     代理监听地址 (默认: "0.0.0.0:8555")
-upstream string   默认上游服务器
-timeout duration  连接超时时间 (默认: 30s)
-max-conn int      最大连接数 (默认: 1000)
-log              启用日志 (默认: true)
-config string    配置文件路径
```

## 配置说明

### 配置文件格式 (JSON)

```json
{
  "listen_address": "0.0.0.0",
  "listen_port": 8555,
  "upstream_servers": {
    "/test": "rtsp://localhost:8554/test",
    "/camera1": "rtsp://localhost:8554/camera1",
    "/camera2": "rtsp://localhost:8554/camera2"
  },
  "default_server": "rtsp://localhost:8554/default",
  "buffer_size": 4096,
  "timeout_seconds": 30,
  "max_connections": 1000,
  "enable_logging": true,
  "log_level": "info",
  "stats_interval": 30
}
```

### 配置参数说明

- `listen_address`: 代理服务器监听地址
- `listen_port`: 代理服务器监听端口
- `upstream_servers`: 上游服务器映射 (路径 -> 服务器URL)
- `default_server`: 默认上游服务器
- `buffer_size`: 缓冲区大小
- `timeout_seconds`: 连接超时时间
- `max_connections`: 最大连接数
- `enable_logging`: 是否启用日志
- `log_level`: 日志级别
- `stats_interval`: 统计信息打印间隔

## 使用示例

### 基本代理

```bash
# 启动代理服务器
./rtsp-proxy -listen 0.0.0.0:8555 -upstream rtsp://192.168.1.100:8554/stream

# 客户端连接代理
vlc rtsp://proxy-server:8555/stream
```

### 多上游服务器

```bash
# 启动代理服务器
./rtsp-proxy -config multi-upstream.json

# 客户端连接不同的流
vlc rtsp://proxy-server:8555/camera1
vlc rtsp://proxy-server:8555/camera2
vlc rtsp://proxy-server:8555/test
```

### 负载均衡

```json
{
  "upstream_servers": {
    "/stream": "rtsp://server1:8554/stream",
    "/stream": "rtsp://server2:8554/stream",
    "/stream": "rtsp://server3:8554/stream"
  }
}
```

## API使用示例

### 创建代理服务器

```go
package main

import (
    "log"
    "time"
    "github.com/iotzf/rtsp-go/proxy"
)

func main() {
    // 创建代理配置
    config := proxy.NewProxyConfig()
    config.ListenAddress = "0.0.0.0"
    config.ListenPort = 8555
    config.Timeout = 30 * time.Second
    config.MaxConnections = 1000
    config.EnableLogging = true

    // 添加上游服务器
    config.UpstreamServers["/test"] = "rtsp://localhost:8554/test"
    config.UpstreamServers["/camera1"] = "rtsp://localhost:8554/camera1"

    // 创建代理服务器
    proxy := proxy.NewRTSPProxy(config)

    // 启动代理
    if err := proxy.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### 动态配置管理

```go
// 添加上游服务器
proxy.AddUpstreamServer("/new-stream", "rtsp://new-server:8554/stream")

// 移除上游服务器
proxy.RemoveUpstreamServer("/old-stream")

// 设置默认上游服务器
proxy.SetDefaultUpstreamServer("rtsp://default:8554/default")

// 获取代理统计
stats := proxy.GetStats()
fmt.Printf("Active sessions: %v\n", stats["active_sessions"])
```

### 媒体流代理

```go
// 创建媒体流代理
mediaProxy := proxy.NewMediaStreamProxy(session)

// 设置传输模式
mediaProxy.SetTransportMode(true) // true for TCP, false for UDP

// 启动媒体流代理
if err := mediaProxy.Start(); err != nil {
    log.Printf("Failed to start media proxy: %v", err)
}

// 获取媒体流统计
stats := mediaProxy.GetStats()
fmt.Printf("Media proxy stats: %+v\n", stats)
```

## 代理架构

```
客户端 <---> RTSP代理 <---> 上游服务器
   |              |              |
   |              |              |
   v              v              v
RTSP控制      RTSP消息转发    RTSP控制
RTP/RTCP     媒体流代理      RTP/RTCP
```

### 代理流程

1. **连接建立**: 客户端连接到代理服务器
2. **会话创建**: 代理创建会话并连接到上游服务器
3. **消息转发**: RTSP控制消息在客户端和上游服务器之间转发
4. **媒体代理**: 媒体流通过代理转发
5. **会话清理**: 连接断开时清理会话和资源

## 性能特性

- **高并发**: 支持数千个并发连接
- **低延迟**: 优化的消息转发机制
- **内存效率**: 智能的缓冲区管理
- **CPU优化**: 高效的I/O处理
- **网络优化**: 支持TCP和UDP传输模式

## 监控和统计

### 代理统计信息

- 总会话数
- 活跃会话数
- 总连接数
- 转发消息数
- 转发字节数
- 运行时间

### 会话统计信息

- 转发消息数
- 转发字节数
- 会话持续时间
- 最后活动时间

### 媒体流统计信息

- RTP包转发数
- RTCP包转发数
- TCP帧转发数
- 转发字节数

## 故障排除

### 常见问题

1. **连接失败**
   - 检查上游服务器是否可达
   - 验证网络连接
   - 检查防火墙设置

2. **媒体流中断**
   - 检查RTP/RTCP端口是否被占用
   - 验证传输模式配置
   - 检查网络带宽

3. **性能问题**
   - 调整缓冲区大小
   - 增加最大连接数
   - 优化上游服务器配置

### 日志分析

```bash
# 启用详细日志
./rtsp-proxy -log

# 查看日志输出
tail -f proxy.log
```

## 扩展开发

### 自定义代理逻辑

```go
// 实现自定义代理处理器
type CustomProxyHandler struct {
    *RTSPProxy
}

func (cph *CustomProxyHandler) ProcessRequest(session *ProxySession, request *rtspProtocon.Message) (*rtspProtocon.Message, error) {
    // 自定义请求处理逻辑
    return cph.RTSPProxy.processRequest(session, request)
}
```

### 添加新的传输协议

```go
// 扩展传输协议支持
func (msp *MediaStreamProxy) StartCustomTransport() error {
    // 实现自定义传输协议
    return nil
}
```

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request来改进这个项目。

## 联系方式

如有问题或建议，请通过GitHub Issues联系。
