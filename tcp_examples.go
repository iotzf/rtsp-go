package main

import (
	"fmt"
	"log"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// 示例：TCP Interleaved帧处理
func tcpInterleavedFrameExample() {
	fmt.Println("\n=== TCP Interleaved帧处理示例 ===")
	
	// 创建RTP包
	payload := []byte("test video frame data")
	rtpPacket := rtspProtocon.NewRTPPacket(
		rtspProtocon.PayloadTypeH264,
		1,
		rtspProtocon.GenerateRTPTimestamp(),
		rtspProtocon.GenerateSSRC(),
		payload,
	)
	
	// 创建TCP Interleaved帧
	frame := rtspProtocon.CreateRTPInterleavedFrame(0, rtpPacket)
	
	fmt.Printf("TCP Interleaved帧大小: %d 字节\n", len(frame.Marshal()))
	fmt.Printf("通道号: %d\n", frame.Header.Channel)
	fmt.Printf("数据长度: %d 字节\n", frame.Header.Length)
	
	// 序列化帧
	frameData := frame.Marshal()
	fmt.Printf("序列化后大小: %d 字节\n", len(frameData))
	
	// 解析帧
	parsedFrame, err := rtspProtocon.ParseTCPInterleavedFrame(frameData)
	if err != nil {
		log.Printf("解析TCP Interleaved帧失败: %v", err)
		return
	}
	
	fmt.Printf("解析成功: 通道=%d, 长度=%d\n", parsedFrame.Header.Channel, parsedFrame.Header.Length)
	
	// 从帧中解析RTP包
	parsedRTPPacket, err := rtspProtocon.ParseRTPFromInterleavedFrame(parsedFrame)
	if err != nil {
		log.Printf("解析RTP包失败: %v", err)
		return
	}
	
	fmt.Printf("RTP包解析成功: 序列号=%d, 时间戳=%d, 载荷大小=%d\n",
		parsedRTPPacket.Header.SequenceNum,
		parsedRTPPacket.Header.Timestamp,
		len(parsedRTPPacket.Payload))
}

// 示例：TCP传输管理器
func tcpTransportManagerExample() {
	fmt.Println("\n=== TCP传输管理器示例 ===")
	
	// 创建TCP传输管理器
	tm := rtspProtocon.NewTCPTransportManager()
	
	// 分配TCP通道
	channel1 := tm.AllocateChannel("session1")
	fmt.Printf("分配通道1: RTP=%d, RTCP=%d\n", channel1.RTPChannel, channel1.RTCPChannel)
	
	channel2 := tm.AllocateChannel("session2")
	fmt.Printf("分配通道2: RTP=%d, RTCP=%d\n", channel2.RTPChannel, channel2.RTCPChannel)
	
	// 获取通道信息
	retrievedChannel := tm.GetChannel("session1")
	if retrievedChannel != nil {
		fmt.Printf("获取通道1: RTP=%d, RTCP=%d\n", retrievedChannel.RTPChannel, retrievedChannel.RTCPChannel)
	}
	
	// 释放通道
	tm.ReleaseChannel("session1")
	fmt.Println("释放通道1")
	
	// 再次分配通道（应该重用通道1的号码）
	channel3 := tm.AllocateChannel("session3")
	fmt.Printf("分配通道3: RTP=%d, RTCP=%d\n", channel3.RTPChannel, channel3.RTCPChannel)
}

// 示例：TCP传输统计
func tcpTransportStatsExample() {
	fmt.Println("\n=== TCP传输统计示例 ===")
	
	// 创建TCP传输统计
	stats := rtspProtocon.NewTCPTransportStats()
	
	// 模拟发送RTP帧
	for i := 0; i < 10; i++ {
		payload := []byte(fmt.Sprintf("frame %d", i))
		rtpPacket := rtspProtocon.NewRTPPacket(
			rtspProtocon.PayloadTypeH264,
			uint16(i+1),
			rtspProtocon.GenerateRTPTimestamp(),
			rtspProtocon.GenerateSSRC(),
			payload,
		)
		
		frame := rtspProtocon.CreateRTPInterleavedFrame(0, rtpPacket)
		stats.UpdateSent(frame)
	}
	
	// 模拟发送RTCP帧
	for i := 0; i < 3; i++ {
		rtcpData := []byte(fmt.Sprintf("rtcp packet %d", i))
		frame := rtspProtocon.NewTCPInterleavedFrame(1, rtcpData)
		stats.UpdateSent(frame)
	}
	
	// 模拟接收帧
	for i := 0; i < 8; i++ {
		payload := []byte(fmt.Sprintf("received frame %d", i))
		rtpPacket := rtspProtocon.NewRTPPacket(
			rtspProtocon.PayloadTypeH264,
			uint16(i+1),
			rtspProtocon.GenerateRTPTimestamp(),
			rtspProtocon.GenerateSSRC(),
			payload,
		)
		
		frame := rtspProtocon.CreateRTPInterleavedFrame(0, rtpPacket)
		stats.UpdateReceived(frame)
	}
	
	// 获取统计信息
	statsData := stats.GetStats()
	fmt.Printf("TCP传输统计: %+v\n", statsData)
}

// 示例：TCP传输模式检测
func tcpTransportDetectionExample() {
	fmt.Println("\n=== TCP传输模式检测示例 ===")
	
	// UDP传输
	udpTransport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportRTP,
		ClientPorts: "5004-5005",
		ServerPorts: "5006-5007",
		Mode:        "PLAY",
		Unicast:     true,
	}
	
	fmt.Printf("UDP传输检测: %v\n", rtspProtocon.IsTCPTransport(udpTransport))
	
	// TCP Interleaved传输
	tcpTransport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportInterleaved,
		Interleaved: "0-1",
		Mode:        "PLAY",
		Unicast:     true,
	}
	
	fmt.Printf("TCP传输检测: %v\n", rtspProtocon.IsTCPTransport(tcpTransport))
	
	// 获取TCP通道号
	rtpChannel := rtspProtocon.GetTCPChannel(tcpTransport, true)
	rtcpChannel := rtspProtocon.GetTCPChannel(tcpTransport, false)
	fmt.Printf("TCP通道号: RTP=%d, RTCP=%d\n", rtpChannel, rtcpChannel)
}

// 示例：TCP传输头解析
func tcpTransportHeaderExample() {
	fmt.Println("\n=== TCP传输头解析示例 ===")
	
	// TCP Interleaved传输头
	transportHeader := "RTP/AVP/TCP;interleaved=0-1;mode=PLAY"
	transport, err := rtspProtocon.ParseTransport(transportHeader)
	if err != nil {
		log.Printf("解析传输头失败: %v", err)
		return
	}
	
	fmt.Printf("传输类型: %v\n", transport.Type)
	fmt.Printf("Interleaved: %s\n", transport.Interleaved)
	fmt.Printf("模式: %s\n", transport.Mode)
	fmt.Printf("单播: %v\n", transport.Unicast)
	
	// 生成传输头字符串
	transportStr := transport.String()
	fmt.Printf("生成的传输头: %s\n", transportStr)
	
	// 多播TCP传输示例
	multicastTransport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportInterleaved,
		Interleaved: "2-3",
		Mode:        "PLAY",
		Multicast:   true,
		TTL:         16,
		Destination: "224.1.1.1",
	}
	
	fmt.Printf("多播TCP传输头: %s\n", multicastTransport.String())
}

// 示例：TCP传输错误处理
func tcpTransportErrorExample() {
	fmt.Println("\n=== TCP传输错误处理示例 ===")
	
	// 创建TCP传输错误
	err := rtspProtocon.NewTCPTransportError(0, "connection lost", fmt.Errorf("network timeout"))
	fmt.Printf("TCP传输错误: %v\n", err)
	
	// 验证TCP通道号
	validChannels := []byte{0, 1, 2, 3, 255}
	invalidChannels := []byte{256, 300}
	
	for _, channel := range validChannels {
		valid := rtspProtocon.ValidateTCPChannel(channel)
		fmt.Printf("通道 %d 验证: %v\n", channel, valid)
	}
	
	for _, channel := range invalidChannels {
		valid := rtspProtocon.ValidateTCPChannel(channel)
		fmt.Printf("通道 %d 验证: %v\n", channel, valid)
	}
}

// 示例：完整的TCP传输流程
func completeTCPTransportExample() {
	fmt.Println("\n=== 完整TCP传输流程示例 ===")
	
	// 1. 创建TCP传输管理器
	tm := rtspProtocon.NewTCPTransportManager()
	
	// 2. 分配TCP通道
	channel := tm.AllocateChannel("demo-session")
	fmt.Printf("分配的TCP通道: RTP=%d, RTCP=%d\n", channel.RTPChannel, channel.RTCPChannel)
	
	// 3. 创建传输信息
	transport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportInterleaved,
		Interleaved: fmt.Sprintf("%d-%d", channel.RTPChannel, channel.RTCPChannel),
		Mode:        "PLAY",
		Unicast:     true,
	}
	
	fmt.Printf("传输头: %s\n", transport.String())
	
	// 4. 创建TCP传输统计
	stats := rtspProtocon.NewTCPTransportStats()
	
	// 5. 模拟媒体流传输
	fmt.Println("模拟媒体流传输...")
	for i := 0; i < 5; i++ {
		// 创建RTP包
		payload := []byte(fmt.Sprintf("Video frame %d", i))
		rtpPacket := rtspProtocon.NewRTPPacket(
			rtspProtocon.PayloadTypeH264,
			uint16(i+1),
			rtspProtocon.GenerateRTPTimestamp(),
			rtspProtocon.GenerateSSRC(),
			payload,
		)
		
		// 创建TCP Interleaved帧
		frame := rtspProtocon.CreateRTPInterleavedFrame(channel.RTPChannel, rtpPacket)
		
		// 更新统计
		stats.UpdateSent(frame)
		
		fmt.Printf("发送帧 %d: 通道=%d, 大小=%d字节\n", 
			i+1, frame.Header.Channel, frame.Header.Length)
		
		time.Sleep(100 * time.Millisecond)
	}
	
	// 6. 获取最终统计
	finalStats := stats.GetStats()
	fmt.Printf("最终统计: %+v\n", finalStats)
	
	// 7. 释放通道
	tm.ReleaseChannel("demo-session")
	fmt.Println("释放TCP通道")
}

func main() {
	fmt.Println("RTSP TCP传输示例程序")
	fmt.Println("====================")
	
	tcpInterleavedFrameExample()
	tcpTransportManagerExample()
	tcpTransportStatsExample()
	tcpTransportDetectionExample()
	tcpTransportHeaderExample()
	tcpTransportErrorExample()
	completeTCPTransportExample()
	
	fmt.Println("\n所有TCP传输示例执行完成！")
}
