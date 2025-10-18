package main

import (
	"fmt"
	"log"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// 示例：创建自定义RTSP消息
func createCustomMessage() {
	fmt.Println("=== 创建自定义RTSP消息示例 ===")
	
	// 创建OPTIONS请求
	optionsReq := rtspProtocon.NewRequest(rtspProtocon.MethodOptions, "rtsp://example.com/stream")
	optionsReq.SetHeader("CSeq", "1")
	optionsReq.SetHeader("User-Agent", "Custom-RTSP-Client/1.0")
	
	fmt.Println("OPTIONS请求:")
	fmt.Println(optionsReq.String())
	
	// 创建200 OK响应
	okResp := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
	okResp.SetHeader("CSeq", "1")
	okResp.SetHeader("Public", "OPTIONS, DESCRIBE, SETUP, PLAY, PAUSE, TEARDOWN")
	okResp.SetHeader("Server", "Custom-RTSP-Server/1.0")
	
	fmt.Println("\n200 OK响应:")
	fmt.Println(okResp.String())
}

// 示例：解析RTSP消息
func parseMessageExample() {
	fmt.Println("\n=== 解析RTSP消息示例 ===")
	
	messageStr := `OPTIONS rtsp://example.com/stream RTSP/1.0
CSeq: 1
User-Agent: Test-Client/1.0

`
	
	message, err := rtspProtocon.ParseMessage(messageStr)
	if err != nil {
		log.Printf("解析消息失败: %v", err)
		return
	}
	
	fmt.Printf("消息类型: %v\n", message.Type)
	fmt.Printf("方法: %s\n", message.Method)
	fmt.Printf("URL: %s\n", message.URL)
	fmt.Printf("版本: %s\n", message.Version)
	fmt.Printf("CSeq: %s\n", message.GetHeader("CSeq"))
	fmt.Printf("User-Agent: %s\n", message.GetHeader("User-Agent"))
}

// 示例：创建SDP会话描述
func createSDPExample() {
	fmt.Println("\n=== 创建SDP会话描述示例 ===")
	
	// 创建SDP
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
	sdp.Connection = &rtspProtocon.Connection{
		NetType:        "IN",
		AddrType:       "IP4",
		ConnectionAddr: "192.168.1.100",
	}
	sdp.Timing = &rtspProtocon.Timing{
		StartTime: 0,
		StopTime:  0,
	}
	
	// 添加视频媒体
	videoMedia := rtspProtocon.NewMediaDescription(
		rtspProtocon.MediaVideo,
		5004,
		rtspProtocon.ProtocolRTPAVP,
		"96",
	)
	videoMedia.Connection = &rtspProtocon.Connection{
		NetType:        "IN",
		AddrType:       "IP4",
		ConnectionAddr: "192.168.1.100",
	}
	videoMedia.RTPMap[96] = &rtspProtocon.RTPMap{
		PayloadType: 96,
		Encoding:    "H264",
		ClockRate:   90000,
		Parameters:  "",
	}
	videoMedia.Attributes["control"] = "track1"
	
	sdp.Media = append(sdp.Media, videoMedia)
	
	fmt.Println("SDP内容:")
	fmt.Println(sdp.String())
}

// 示例：解析SDP
func parseSDPExample() {
	fmt.Println("\n=== 解析SDP示例 ===")
	
	sdpStr := `v=0
o=user 1234567890 1234567890 IN IP4 192.168.1.100
s=Test Video Stream
c=IN IP4 192.168.1.100
t=0 0
m=video 5004 RTP/AVP 96
a=rtpmap:96 H264/90000
a=control:track1
`
	
	sdp, err := rtspProtocon.ParseSDP(sdpStr)
	if err != nil {
		log.Printf("解析SDP失败: %v", err)
		return
	}
	
	fmt.Printf("版本: %d\n", sdp.Version)
	fmt.Printf("会话名称: %s\n", sdp.SessionName)
	if sdp.Origin != nil {
		fmt.Printf("Origin: %s %d %d %s %s %s\n",
			sdp.Origin.Username,
			sdp.Origin.SessID,
			sdp.Origin.SessVersion,
			sdp.Origin.NetType,
			sdp.Origin.AddrType,
			sdp.Origin.UnicastAddress)
	}
	fmt.Printf("媒体数量: %d\n", len(sdp.Media))
	
	for i, media := range sdp.Media {
		fmt.Printf("媒体 %d: %s %d %s %s\n",
			i+1,
			string(media.MediaType),
			media.Port,
			string(media.Protocol),
			media.Format)
	}
}

// 示例：创建RTP包
func createRTPExample() {
	fmt.Println("\n=== 创建RTP包示例 ===")
	
	// 创建RTP头部
	header := rtspProtocon.NewRTPHeader()
	header.PayloadType = rtspProtocon.PayloadTypeH264
	header.SequenceNum = 1
	header.Timestamp = rtspProtocon.GenerateRTPTimestamp()
	header.SSRC = rtspProtocon.GenerateSSRC()
	header.Marker = false
	
	// 创建RTP包
	payload := []byte("test video data")
	packet := rtspProtocon.NewRTPPacket(
		rtspProtocon.PayloadTypeH264,
		1,
		rtspProtocon.GenerateRTPTimestamp(),
		rtspProtocon.GenerateSSRC(),
		payload,
	)
	
	// 序列化RTP包
	packetData := packet.Marshal()
	fmt.Printf("RTP包大小: %d 字节\n", len(packetData))
	fmt.Printf("载荷类型: %s\n", rtspProtocon.GetPayloadTypeName(packet.Header.PayloadType))
	fmt.Printf("序列号: %d\n", packet.Header.SequenceNum)
	fmt.Printf("时间戳: %d\n", packet.Header.Timestamp)
	fmt.Printf("SSRC: %d\n", packet.Header.SSRC)
	
	// 解析RTP包
	parsedPacket, err := rtspProtocon.ParseRTPPacket(packetData)
	if err != nil {
		log.Printf("解析RTP包失败: %v", err)
		return
	}
	
	fmt.Printf("解析后的载荷大小: %d 字节\n", len(parsedPacket.Payload))
}

// 示例：端口管理功能
func portManagementExample() {
	fmt.Println("\n=== 端口管理示例 ===")
	
	// 创建端口管理器
	pm := rtspProtocon.NewPortManager(5000, 5100, 5000, 5100)
	
	// 验证端口范围
	if err := pm.ValidatePortRange(); err != nil {
		log.Printf("端口范围验证失败: %v", err)
		return
	}
	fmt.Println("端口范围验证通过")
	
	// 分配端口
	allocation1, err := pm.AllocatePorts("session1")
	if err != nil {
		log.Printf("端口分配失败: %v", err)
		return
	}
	fmt.Printf("分配端口1: RTP=%d, RTCP=%d\n", allocation1.RTPPort, allocation1.RTCPPort)
	
	// 再次分配端口
	allocation2, err := pm.AllocatePorts("session2")
	if err != nil {
		log.Printf("端口分配失败: %v", err)
		return
	}
	fmt.Printf("分配端口2: RTP=%d, RTCP=%d\n", allocation2.RTPPort, allocation2.RTCPPort)
	
	// 检查端口冲突
	conflict := pm.CheckPortConflict(allocation1.RTPPort)
	fmt.Printf("端口 %d 冲突检查: %v\n", allocation1.RTPPort, conflict)
	
	// 获取端口统计
	stats := pm.GetStats()
	fmt.Printf("端口统计: %+v\n", stats)
	
	// 释放端口
	pm.ReleasePorts(allocation1)
	pm.ReleasePorts(allocation2)
	fmt.Println("端口已释放")
	
	// 带重试的端口分配
	allocation3, err := pm.AllocatePortsWithRetry("session3", 3)
	if err != nil {
		log.Printf("重试端口分配失败: %v", err)
		return
	}
	fmt.Printf("重试分配端口: RTP=%d, RTCP=%d\n", allocation3.RTPPort, allocation3.RTCPPort)
	
	// 获取可用端口列表
	availablePorts, err := pm.GetAvailablePorts(5)
	if err != nil {
		log.Printf("获取可用端口失败: %v", err)
		return
	}
	fmt.Printf("可用端口: %v\n", availablePorts)
	
	pm.ReleasePorts(allocation3)
}

// 示例：动态传输头处理
func dynamicTransportExample() {
	fmt.Println("\n=== 动态传输头处理示例 ===")
	
	// 创建传输信息
	transport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportRTP,
		ClientPorts: "5004-5005",
		ServerPorts: "5006-5007",
		Mode:        "PLAY",
		Unicast:     true,
		SSRC:        "12345678",
	}
	
	fmt.Printf("原始传输头: %s\n", transport.String())
	
	// 解析传输头
	parsedTransport, err := rtspProtocon.ParseTransport(transport.String())
	if err != nil {
		log.Printf("解析传输头失败: %v", err)
		return
	}
	
	fmt.Printf("解析结果: Type=%v, ClientPorts=%s, ServerPorts=%s, Mode=%s, Unicast=%v\n",
		parsedTransport.Type,
		parsedTransport.ClientPorts,
		parsedTransport.ServerPorts,
		parsedTransport.Mode,
		parsedTransport.Unicast)
	
	// 动态修改端口
	parsedTransport.ServerPorts = "5010-5011"
	fmt.Printf("修改后传输头: %s\n", parsedTransport.String())
	
	// 多播传输示例
	multicastTransport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportRTP,
		Mode:        "PLAY",
		Multicast:   true,
		TTL:         16,
		Destination: "224.1.1.1",
	}
	fmt.Printf("多播传输头: %s\n", multicastTransport.String())
}

// 示例：会话管理
func sessionExample() {
	fmt.Println("\n=== 会话管理示例 ===")
	
	// 创建会话
	sessionID := rtspProtocon.GenerateSessionID()
	session := rtspProtocon.NewSession(sessionID)
	session.URL = "rtsp://example.com/stream"
	session.UpdateState(rtspProtocon.StateReady)
	
	fmt.Printf("会话ID: %s\n", session.ID)
	fmt.Printf("会话状态: %v\n", session.State)
	fmt.Printf("会话URL: %s\n", session.URL)
	fmt.Printf("创建时间: %s\n", session.Created.Format(time.RFC3339))
	fmt.Printf("最后访问: %s\n", session.LastSeen.Format(time.RFC3339))
	
	// 更新状态
	session.UpdateState(rtspProtocon.StatePlaying)
	fmt.Printf("更新后状态: %v\n", session.State)
}

// 示例：RTP统计
func rtpStatsExample() {
	fmt.Println("\n=== RTP统计示例 ===")
	
	// 创建RTP统计
	stats := rtspProtocon.NewRTPStats(rtspProtocon.GenerateSSRC())
	
	// 模拟发送数据
	for i := 0; i < 10; i++ {
		stats.UpdateSent(uint16(i+1), rtspProtocon.GenerateRTPTimestamp(), 1000)
	}
	
	// 模拟接收数据（丢失2个包）
	for i := 0; i < 8; i++ {
		stats.UpdateReceived(uint16(i+1), rtspProtocon.GenerateRTPTimestamp(), 1000)
	}
	
	fmt.Printf("发送包数: %d\n", stats.PacketsSent)
	fmt.Printf("接收包数: %d\n", stats.PacketsReceived)
	fmt.Printf("发送字节: %d\n", stats.BytesSent)
	fmt.Printf("接收字节: %d\n", stats.BytesReceived)
	fmt.Printf("丢包率: %.2f%%\n", stats.GetLossRate()*100)
}

func main() {
	fmt.Println("RTSP协议库示例程序")
	fmt.Println("==================")
	
	createCustomMessage()
	parseMessageExample()
	createSDPExample()
	parseSDPExample()
	createRTPExample()
	portManagementExample()
	dynamicTransportExample()
	sessionExample()
	rtpStatsExample()
	
	fmt.Println("\n所有示例执行完成！")
}
