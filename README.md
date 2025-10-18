# RTSP协议实现 (Go语言)

这是一个使用Go语言实现的完整RTSP (Real Time Streaming Protocol) 协议栈，包含协议层、客户端和服务端实现。

## 项目结构

```
rtsp-go/
├── rtspProtocon/          # RTSP协议核心实现
│   ├── protocol.go        # RTSP协议消息处理
│   ├── sdp.go            # SDP (Session Description Protocol) 支持
│   └── rtp.go            # RTP (Real-time Transport Protocol) 支持
├── server/               # RTSP服务端实现
│   └── server.go         # 服务端主程序
├── client/               # RTSP客户端实现
│   └── client.go         # 客户端主程序
├── examples.go           # 使用示例
└── README.md            # 项目说明文档
```

## 功能特性

### RTSP协议层 (rtspProtocon)
- ✅ 完整的RTSP消息解析和生成
- ✅ 支持所有标准RTSP方法 (OPTIONS, DESCRIBE, SETUP, PLAY, PAUSE, TEARDOWN)
- ✅ 会话管理和状态跟踪
- ✅ 传输协议支持 (RTP/AVP, UDP, TCP)
- ✅ 消息头处理和验证
- ✅ **动态端口分配和管理**
- ✅ **端口冲突检测和重试机制**
- ✅ **TCP Interleaved传输模式支持**
- ✅ **TCP媒体流传输和接收**

### SDP支持
- ✅ Session Description Protocol 完整实现
- ✅ 媒体描述和RTP映射
- ✅ SDP解析和生成
- ✅ 支持视频和音频媒体类型

### RTP支持
- ✅ Real-time Transport Protocol 实现
- ✅ RTP头部解析和生成
- ✅ 统计信息收集
- ✅ 多种载荷类型支持 (H.264, H.265, VP8, VP9等)

### RTSP服务端
- ✅ 多客户端并发支持
- ✅ 流管理和会话跟踪
- ✅ 完整的RTSP方法实现
- ✅ **动态端口池管理**
- ✅ **智能端口分配算法**
- ✅ **端口使用统计和监控**

### RTSP客户端
- ✅ 完整的RTSP会话流程
- ✅ 自动重连和错误处理
- ✅ 统计信息收集
- ✅ 支持播放控制 (播放/暂停/停止)
- ✅ **动态端口处理**
- ✅ **端口分配信息跟踪**

## 🚀 技术优势
- 高并发支持: 通过端口池管理支持大量并发连接
- 资源优化: 智能端口分配避免资源浪费
- 故障恢复: 重试机制确保服务稳定性
- 监控友好: 详细的统计信息便于运维监控
- 扩展性强: 模块化设计便于功能扩展

## 快速开始

### 1. 编译项目

```bash
# 编译服务端
cd server
go build -o rtsp-server server.go

# 编译客户端
cd ../client
go build -o rtsp-client client.go

# 编译示例程序
cd ..
go build -o examples examples.go
```

### 2. 运行服务端

```bash
./rtsp-server
```

服务端将在 `0.0.0.0:8554` 启动，提供以下测试流：
- `rtsp://localhost:8554/test` - 默认测试流
- `rtsp://localhost:8554/camera1` - 摄像头1流
- `rtsp://localhost:8554/camera2` - 摄像头2流

### 3. 运行客户端

```bash
./rtsp-client
```

客户端将连接到服务端并执行完整的RTSP会话流程。

### 4. 运行示例程序

```bash
./examples
```

查看各种协议组件的使用示例。

## API使用示例

### 创建RTSP消息

```go
import "rtspProtocon"

// 创建OPTIONS请求
request := rtspProtocon.NewRequest(rtspProtocon.MethodOptions, "rtsp://example.com/stream")
request.SetHeader("CSeq", "1")
request.SetHeader("User-Agent", "My-RTSP-Client/1.0")

// 创建响应
response := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
response.SetHeader("CSeq", "1")
response.SetHeader("Public", "OPTIONS, DESCRIBE, SETUP, PLAY, PAUSE, TEARDOWN")
```

### 解析RTSP消息

```go
messageStr := `OPTIONS rtsp://example.com/stream RTSP/1.0
CSeq: 1
User-Agent: Test-Client/1.0

`

message, err := rtspProtocon.ParseMessage(messageStr)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("方法: %s\n", message.Method)
fmt.Printf("URL: %s\n", message.URL)
```

### 创建SDP会话描述

```go
sdp := rtspProtocon.NewSessionDescription()
sdp.Version = 0
sdp.Origin = &rtspProtocon.Origin{
    Username:       "user",
    SessID:         1234567890,
    SessVersion:    1234567890,
    NetType:        "IN",
    AddrType:       "IP4",
    UnicastAddress: "192.168.1.100",
}
sdp.SessionName = "Test Video Stream"

// 添加视频媒体
videoMedia := rtspProtocon.NewMediaDescription(
    rtspProtocon.MediaVideo,
    5004,
    rtspProtocon.ProtocolRTPAVP,
    "96",
)
videoMedia.RTPMap[96] = &rtspProtocon.RTPMap{
    PayloadType: 96,
    Encoding:    "H264",
    ClockRate:   90000,
}
sdp.Media = append(sdp.Media, videoMedia)

fmt.Println(sdp.String())
```

### 创建RTP包

```go
// 创建RTP包
payload := []byte("video data")
packet := rtspProtocon.NewRTPPacket(
    rtspProtocon.PayloadTypeH264,
    1,
    rtspProtocon.GenerateRTPTimestamp(),
    rtspProtocon.GenerateSSRC(),
    payload,
)

// 序列化
packetData := packet.Marshal()

// 解析
parsedPacket, err := rtspProtocon.ParseRTPPacket(packetData)
```

### 使用RTSP客户端

```go
client := rtspProtocon.NewRTSPClient("rtsp://localhost:8554/test")

// 启动完整会话
err := client.StartSession()
if err != nil {
    log.Fatal(err)
}

// 暂停流
client.Pause()

// 恢复播放
client.Play()

// 断开连接
client.Disconnect()
```

### 使用RTSP服务端

```go
server := rtspProtocon.NewRTSPServer("0.0.0.0", 8554)

// 设置自定义端口范围
err := server.SetPortRange(5000, 5100, 5000, 5100)
if err != nil {
    log.Fatal(err)
}

// 添加自定义流
server.AddStream("my-stream", "My Custom Stream", 5006)

// 启动服务端
err := server.Start()
if err != nil {
    log.Fatal(err)
}

// 获取端口使用统计
stats := server.GetStats()
log.Printf("Port stats: %+v", stats["port_stats"])

// 停止服务端
server.Stop()
```

### TCP传输模式

```go
// 创建RTSP客户端并启用TCP传输
client := rtspProtocon.NewRTSPClient("rtsp://localhost:8554/test")
client.SetTCPTransport(true) // 启用TCP传输

// 启动会话
err := client.StartSession()
if err != nil {
    log.Fatal(err)
}

// 获取TCP通道信息
if client.TCPChannel != nil {
    fmt.Printf("TCP通道: RTP=%d, RTCP=%d\n", 
        client.TCPChannel.RTPChannel, client.TCPChannel.RTCPChannel)
}

// 发送RTCP包
rtcpData := []byte("RTCP packet data")
err = client.SendTCPRTCPPacket(rtcpData)
if err != nil {
    log.Printf("发送RTCP包失败: %v", err)
}

// 获取TCP传输统计
stats := client.GetStats()
fmt.Printf("TCP统计: %+v\n", stats["tcp_stats"])
```

### TCP传输管理器

```go
// 创建TCP传输管理器
tm := rtspProtocon.NewTCPTransportManager()

// 分配TCP通道
channel := tm.AllocateChannel("session1")
fmt.Printf("分配通道: RTP=%d, RTCP=%d\n", channel.RTPChannel, channel.RTCPChannel)

// 创建TCP Interleaved帧
rtpPacket := rtspProtocon.NewRTPPacket(
    rtspProtocon.PayloadTypeH264,
    1,
    rtspProtocon.GenerateRTPTimestamp(),
    rtspProtocon.GenerateSSRC(),
    []byte("video data"),
)

frame := rtspProtocon.CreateRTPInterleavedFrame(channel.RTPChannel, rtpPacket)

// 序列化帧
frameData := frame.Marshal()

// 解析帧
parsedFrame, err := rtspProtocon.ParseTCPInterleavedFrame(frameData)
if err != nil {
    log.Fatal(err)
}

// 从帧中解析RTP包
rtpPacket, err = rtspProtocon.ParseRTPFromInterleavedFrame(parsedFrame)
if err != nil {
    log.Fatal(err)
}

// 释放通道
tm.ReleaseChannel("session1")
```

## 协议支持

### RTSP方法
- `OPTIONS` - 查询服务器支持的方法
- `DESCRIBE` - 获取媒体描述 (SDP)
- `SETUP` - 建立传输连接
- `PLAY` - 开始播放
- `PAUSE` - 暂停播放
- `TEARDOWN` - 结束会话

### 媒体格式
- H.264 (Payload Type 96)
- H.265 (Payload Type 97)
- VP8 (Payload Type 98)
- VP9 (Payload Type 99)
- AV1 (Payload Type 100)

### 传输协议
- RTP/AVP (UDP)
- RTP/AVPF (UDP with feedback)
- RTP/AVP/TCP (TCP interleaved)
- TCP (interleaved)

## 测试

使用VLC或其他RTSP客户端测试：

```bash
# VLC播放器 (UDP传输)
vlc rtsp://localhost:8554/test

# VLC播放器 (TCP传输)
vlc --rtsp-tcp rtsp://localhost:8554/test

# FFmpeg (UDP传输)
ffmpeg -i rtsp://localhost:8554/test -c copy output.mp4

# FFmpeg (TCP传输)
ffmpeg -rtsp_transport tcp -i rtsp://localhost:8554/test -c copy output.mp4

# GStreamer (UDP传输)
gst-launch-1.0 rtspsrc location=rtsp://localhost:8554/test ! decodebin ! autovideosink

# GStreamer (TCP传输)
gst-launch-1.0 rtspsrc protocols=tcp location=rtsp://localhost:8554/test ! decodebin ! autovideosink
```

## 扩展开发

### 添加新的媒体格式

1. 在 `rtp.go` 中添加新的载荷类型常量
2. 在 `GetPayloadTypeName` 函数中添加名称映射
3. 在SDP创建时使用新的载荷类型

### 添加新的RTSP方法

1. 在 `protocol.go` 中添加方法常量
2. 在服务端和客户端中实现方法处理
3. 更新OPTIONS响应的Public头

### 自定义传输协议

1. 扩展 `TransportType` 枚举
2. 实现新的传输协议解析
3. 在服务端和客户端中添加支持

### 动态端口管理扩展

1. **自定义端口分配策略**:
   ```go
   // 实现自定义端口分配算法
   type CustomPortAllocator struct {
       *rtspProtocon.PortManager
   }
   
   func (cpa *CustomPortAllocator) AllocatePortsWithStrategy(sessionID string) (*rtspProtocon.PortAllocation, error) {
       // 自定义分配逻辑
   }
   ```

2. **端口使用监控**:
   ```go
   // 添加端口使用监控
   func (pm *PortManager) MonitorPortUsage() {
       go func() {
           for {
               stats := pm.GetStats()
               log.Printf("Port usage: %+v", stats)
               time.Sleep(30 * time.Second)
           }
       }()
   }
   ```

3. **端口池预热**:
   ```go
   // 预分配端口池
   func (pm *PortManager) WarmupPortPool(size int) error {
       _, err := pm.PreAllocatePorts(size)
       return err
   }
   ```

### TCP传输扩展

1. **自定义TCP传输策略**:
   ```go
   // 实现自定义TCP传输策略
   type CustomTCPTransport struct {
       *rtspProtocon.TCPTransportManager
   }
   
   func (ctt *CustomTCPTransport) AllocateChannelWithStrategy(sessionID string) *rtspProtocon.TCPChannel {
       // 自定义通道分配逻辑
   }
   ```

2. **TCP传输监控**:
   ```go
   // 添加TCP传输监控
   func (tm *TCPTransportManager) MonitorTCPTransport() {
       go func() {
           for {
               // 监控TCP传输状态
               time.Sleep(30 * time.Second)
           }
       }()
   }
   ```

3. **TCP传输优化**:
   ```go
   // TCP传输优化
   func OptimizeTCPTransport(frame *rtspProtocon.TCPInterleavedFrame) {
       // 实现TCP传输优化逻辑
       // 例如：帧合并、压缩等
   }
   ```

## RTSP代理服务器

本项目还包含一个完整的RTSP代理服务器实现，位于 `proxy/` 目录下。

### 代理功能特性

- ✅ **RTSP消息代理**: 完整的RTSP协议消息转发
- ✅ **媒体流代理**: 支持RTP/RTCP和TCP Interleaved媒体流代理
- ✅ **负载均衡**: 支持多个上游服务器和路由配置
- ✅ **会话管理**: 完整的代理会话生命周期管理
- ✅ **统计监控**: 详细的代理性能和统计信息
- ✅ **配置管理**: 灵活的配置文件和命令行参数
- ✅ **高并发**: 支持大量并发连接和会话

### 代理架构

```
客户端 <---> RTSP代理 <---> 上游服务器
   |              |              |
   |              |              |
   v              v              v
RTSP控制      RTSP消息转发    RTSP控制
RTP/RTCP     媒体流代理      RTP/RTCP
```

### 快速使用代理

```bash
# 编译代理程序
cd proxy
go build -o rtsp-proxy main.go proxy.go rtsp_io.go media_proxy.go config.go

# 启动代理服务器
./rtsp-proxy -listen 0.0.0.0:8555 -upstream rtsp://localhost:8554/test

# 客户端连接代理
vlc rtsp://proxy-server:8555/stream
```

### 代理配置示例

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
  "timeout_seconds": 30,
  "max_connections": 1000,
  "enable_logging": true
}
```

### 代理应用场景

- **负载均衡**: 将客户端请求分发到多个上游服务器
- **协议转换**: 在不同RTSP实现之间进行协议转换
- **访问控制**: 实现RTSP流的访问控制和认证
- **网络优化**: 优化网络传输和减少延迟
- **监控代理**: 监控和分析RTSP流量
- **防火墙穿透**: 帮助RTSP流穿越防火墙和NAT

详细文档请参考 [proxy/README.md](proxy/README.md)。

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request来改进这个项目。

## 联系方式

如有问题或建议，请通过GitHub Issues联系。
