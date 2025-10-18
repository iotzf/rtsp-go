package main

import (
	"fmt"
	"log"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// 示例：创建RTSP代理
func createRTSPProxyExample() {
	fmt.Println("=== 创建RTSP代理示例 ===")
	
	// 创建代理配置
	config := NewProxyConfig()
	config.ListenAddress = "0.0.0.0"
	config.ListenPort = 8555
	config.Timeout = 30 * time.Second
	config.MaxConnections = 100
	config.EnableLogging = true
	
	// 添加上游服务器
	config.UpstreamServers["/test"] = "rtsp://localhost:8554/test"
	config.UpstreamServers["/camera1"] = "rtsp://localhost:8554/camera1"
	config.UpstreamServers["/camera2"] = "rtsp://localhost:8554/camera2"
	
	// 设置默认上游服务器
	config.DefaultServer = "rtsp://localhost:8554/default"
	
	fmt.Printf("代理配置: %+v\n", config)
	
	// 创建RTSP代理
	proxy := NewRTSPProxy(config)
	
	fmt.Printf("RTSP代理创建成功: %s\n", proxy.Config.ListenAddress)
}

// 示例：代理配置管理
func proxyConfigManagementExample() {
	fmt.Println("\n=== 代理配置管理示例 ===")
	
	// 创建代理
	config := NewProxyConfig()
	proxy := NewRTSPProxy(config)
	
	// 添加上游服务器
	proxy.AddUpstreamServer("/stream1", "rtsp://server1:8554/stream1")
	proxy.AddUpstreamServer("/stream2", "rtsp://server2:8554/stream2")
	proxy.AddUpstreamServer("/stream3", "rtsp://server3:8554/stream3")
	
	// 设置默认上游服务器
	proxy.SetDefaultUpstreamServer("rtsp://default:8554/default")
	
	// 移除上游服务器
	proxy.RemoveUpstreamServer("/stream2")
	
	fmt.Printf("上游服务器配置: %+v\n", proxy.Config.UpstreamServers)
	fmt.Printf("默认上游服务器: %s\n", proxy.Config.DefaultServer)
}

// 示例：代理会话管理
func proxySessionManagementExample() {
	fmt.Println("\n=== 代理会话管理示例 ===")
	
	// 创建代理会话
	sessionID := "session-12345"
	// 注意：这里只是示例，实际使用中需要真实的连接
	// clientConn := &net.TCPConn{}
	// session := NewProxySession(sessionID, clientConn)
	
	fmt.Printf("会话ID: %s\n", sessionID)
	fmt.Println("代理会话创建成功")
	
	// 模拟会话统计
	sessionStats := &ProxySessionStats{
		MessagesForwarded: 100,
		BytesForwarded:    1024000,
		StartTime:         time.Now(),
		LastActivity:      time.Now(),
	}
	
	fmt.Printf("会话统计: %+v\n", sessionStats)
}

// 示例：媒体流代理
func mediaStreamProxyExample() {
	fmt.Println("\n=== 媒体流代理示例 ===")
	
	// 创建媒体流代理
	// 注意：这里只是示例，实际使用中需要真实的会话
	// session := &ProxySession{ID: "test-session"}
	// mediaProxy := NewMediaStreamProxy(session)
	
	fmt.Println("媒体流代理创建成功")
	
	// 模拟RTP代理统计
	rtpStats := &RTPProxyStats{
		RTPPacketsForwarded:  1000,
		RTPBytesForwarded:    10240000,
		RTCPPacketsForwarded: 50,
		RTCPBytesForwarded:   51200,
		StartTime:            time.Now(),
		LastActivity:         time.Now(),
	}
	
	fmt.Printf("RTP代理统计: %+v\n", rtpStats)
	
	// 模拟RTCP代理统计
	rtcpStats := &RTCPProxyStats{
		PacketsForwarded: 50,
		BytesForwarded:   51200,
		StartTime:        time.Now(),
		LastActivity:     time.Now(),
	}
	
	fmt.Printf("RTCP代理统计: %+v\n", rtcpStats)
	
	// 模拟TCP代理统计
	tcpStats := &TCPProxyStats{
		FramesForwarded: 1000,
		BytesForwarded:  10240000,
		StartTime:       time.Now(),
		LastActivity:    time.Now(),
	}
	
	fmt.Printf("TCP代理统计: %+v\n", tcpStats)
}

// 示例：代理统计信息
func proxyStatsExample() {
	fmt.Println("\n=== 代理统计信息示例 ===")
	
	// 创建代理统计
	stats := &ProxyStats{
		TotalSessions:     100,
		ActiveSessions:    5,
		TotalConnections:  1000,
		MessagesForwarded: 50000,
		BytesForwarded:    1024000000,
		StartTime:         time.Now(),
	}
	
	fmt.Printf("代理统计: %+v\n", stats)
	
	// 计算运行时间
	uptime := time.Since(stats.StartTime)
	fmt.Printf("运行时间: %s\n", uptime.String())
	
	// 计算平均消息转发率
	if uptime.Seconds() > 0 {
		avgMessagesPerSecond := float64(stats.MessagesForwarded) / uptime.Seconds()
		fmt.Printf("平均消息转发率: %.2f 消息/秒\n", avgMessagesPerSecond)
	}
	
	// 计算平均字节转发率
	if uptime.Seconds() > 0 {
		avgBytesPerSecond := float64(stats.BytesForwarded) / uptime.Seconds()
		fmt.Printf("平均字节转发率: %.2f 字节/秒\n", avgBytesPerSecond)
	}
}

// 示例：RTSP消息转发
func rtspMessageForwardingExample() {
	fmt.Println("\n=== RTSP消息转发示例 ===")
	
	// 创建RTSP请求
	request := rtspProtocon.NewRequest(rtspProtocon.MethodDescribe, "rtsp://proxy:8555/test")
	request.SetHeader("CSeq", "1")
	request.SetHeader("User-Agent", "RTSP-Proxy-Client/1.0")
	request.SetHeader("Accept", "application/sdp")
	
	fmt.Printf("原始请求: %s\n", request.String())
	
	// 模拟代理修改请求
	modifiedRequest := request
	modifiedRequest.URL = "rtsp://upstream:8554/test"
	
	fmt.Printf("修改后请求: %s\n", modifiedRequest.String())
	
	// 创建RTSP响应
	response := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
	response.SetHeader("CSeq", "1")
	response.SetHeader("Content-Type", "application/sdp")
	response.SetHeader("Content-Length", "500")
	
	fmt.Printf("上游响应: %s\n", response.String())
	
	// 模拟代理修改响应
	modifiedResponse := response
	modifiedResponse.SetHeader("Location", "rtsp://proxy:8555/test")
	
	fmt.Printf("修改后响应: %s\n", modifiedResponse.String())
}

// 示例：传输模式检测
func transportModeDetectionExample() {
	fmt.Println("\n=== 传输模式检测示例 ===")
	
	// UDP传输
	udpTransport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportRTP,
		ClientPorts: "5004-5005",
		ServerPorts: "5006-5007",
		Mode:        "PLAY",
		Unicast:     true,
	}
	
	fmt.Printf("UDP传输检测: %v\n", rtspProtocon.IsTCPTransport(udpTransport))
	fmt.Printf("UDP传输头: %s\n", udpTransport.String())
	
	// TCP传输
	tcpTransport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportInterleaved,
		Interleaved: "0-1",
		Mode:        "PLAY",
		Unicast:     true,
	}
	
	fmt.Printf("TCP传输检测: %v\n", rtspProtocon.IsTCPTransport(tcpTransport))
	fmt.Printf("TCP传输头: %s\n", tcpTransport.String())
}

// 示例：代理性能测试
func proxyPerformanceTestExample() {
	fmt.Println("\n=== 代理性能测试示例 ===")
	
	// 模拟性能测试数据
	testData := map[string]interface{}{
		"concurrent_sessions": 100,
		"messages_per_second": 1000,
		"bytes_per_second":    1024000,
		"average_latency_ms":  5.5,
		"cpu_usage_percent":   25.0,
		"memory_usage_mb":     128.0,
		"network_throughput": "10 Mbps",
	}
	
	fmt.Printf("性能测试结果: %+v\n", testData)
	
	// 计算性能指标
	concurrentSessions := testData["concurrent_sessions"].(int)
	messagesPerSecond := testData["messages_per_second"].(int)
	
	throughputPerSession := float64(messagesPerSecond) / float64(concurrentSessions)
	fmt.Printf("每会话吞吐量: %.2f 消息/秒\n", throughputPerSession)
	
	// 计算资源利用率
	cpuUsage := testData["cpu_usage_percent"].(float64)
	memoryUsage := testData["memory_usage_mb"].(float64)
	
	fmt.Printf("CPU利用率: %.1f%%\n", cpuUsage)
	fmt.Printf("内存使用: %.1f MB\n", memoryUsage)
}

// 示例：代理监控和告警
func proxyMonitoringExample() {
	fmt.Println("\n=== 代理监控和告警示例 ===")
	
	// 模拟监控数据
	monitoringData := map[string]interface{}{
		"active_sessions":     50,
		"total_connections":   1000,
		"error_rate":          0.01,
		"average_response_time": 10.5,
		"upstream_servers":    []string{"server1", "server2", "server3"},
		"health_status":       "healthy",
		"last_restart":        time.Now().Add(-24 * time.Hour),
	}
	
	fmt.Printf("监控数据: %+v\n", monitoringData)
	
	// 检查告警条件
	activeSessions := monitoringData["active_sessions"].(int)
	errorRate := monitoringData["error_rate"].(float64)
	
	if activeSessions > 80 {
		fmt.Println("⚠️  告警: 活跃会话数过高")
	}
	
	if errorRate > 0.05 {
		fmt.Println("⚠️  告警: 错误率过高")
	}
	
	// 健康检查
	healthStatus := monitoringData["health_status"].(string)
	if healthStatus == "healthy" {
		fmt.Println("✅ 代理状态健康")
	} else {
		fmt.Println("❌ 代理状态异常")
	}
}

func main() {
	fmt.Println("RTSP代理示例程序")
	fmt.Println("================")
	
	createRTSPProxyExample()
	proxyConfigManagementExample()
	proxySessionManagementExample()
	mediaStreamProxyExample()
	proxyStatsExample()
	rtspMessageForwardingExample()
	transportModeDetectionExample()
	proxyPerformanceTestExample()
	proxyMonitoringExample()
	
	fmt.Println("\n所有RTSP代理示例执行完成！")
}
