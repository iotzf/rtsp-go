package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	rtspProtocon "rtsp-go/rtspProtocon"
)

// RTSP客户端
type RTSPClient struct {
	ServerURL         string
	Conn              net.Conn
	Reader            *bufio.Reader
	Writer            *bufio.Writer
	Session           *rtspProtocon.Session
	SDP               *rtspProtocon.SessionDescription
	Transport         *rtspProtocon.Transport
	RTPConn           net.Conn
	RTCPConn          net.Conn
	Connected         bool
	SequenceNum       uint16
	RTPStats          *rtspProtocon.RTPStats
	PortAllocation    *rtspProtocon.PortAllocation
	DynamicPorts      bool
	UseTCPTransport   bool
	TCPChannel        *rtspProtocon.TCPChannel
	TCPStats          *rtspProtocon.TCPTransportStats
}

// 创建新的RTSP客户端
func NewRTSPClient(serverURL string) *RTSPClient {
	return &RTSPClient{
		ServerURL:       serverURL,
		Connected:       false,
		SequenceNum:     1,
		RTPStats:        rtspProtocon.NewRTPStats(rtspProtocon.GenerateSSRC()),
		DynamicPorts:    true, // 默认启用动态端口
		UseTCPTransport: false, // 默认使用UDP传输
		TCPStats:        rtspProtocon.NewTCPTransportStats(),
	}
}

// 连接到RTSP服务器
func (c *RTSPClient) Connect() error {
	// 解析服务器URL
	host, port, err := c.parseURL(c.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %v", err)
	}

	// 连接到服务器
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	c.Conn = conn
	c.Reader = bufio.NewReader(conn)
	c.Writer = bufio.NewWriter(conn)
	c.Connected = true

	log.Printf("Connected to RTSP server: %s", addr)
	return nil
}

// 断开连接
func (c *RTSPClient) Disconnect() error {
	if c.Connected {
		// 发送TEARDOWN请求
		c.Teardown()
		
		// 关闭RTP连接
		if c.RTPConn != nil {
			c.RTPConn.Close()
		}
		if c.RTCPConn != nil {
			c.RTCPConn.Close()
		}
		
		// 关闭RTSP连接
		c.Conn.Close()
		c.Connected = false
		log.Println("Disconnected from RTSP server")
	}
	return nil
}

// 发送OPTIONS请求
func (c *RTSPClient) Options() error {
	request := rtspProtocon.NewRequest(rtspProtocon.MethodOptions, c.ServerURL)
	request.SetHeader("User-Agent", "Go-RTSP-Client/1.0")
	request.SetHeader("CSeq", strconv.Itoa(int(c.SequenceNum)))

	response, err := c.sendRequest(request)
	if err != nil {
		return err
	}

	if response.StatusCode != rtspProtocon.StatusOK {
		return fmt.Errorf("OPTIONS request failed: %d %s", response.StatusCode, response.StatusText)
	}

	log.Println("OPTIONS request successful")
	c.SequenceNum++
	return nil
}

// 发送DESCRIBE请求
func (c *RTSPClient) Describe() error {
	request := rtspProtocon.NewRequest(rtspProtocon.MethodDescribe, c.ServerURL)
	request.SetHeader("User-Agent", "Go-RTSP-Client/1.0")
	request.SetHeader("CSeq", strconv.Itoa(int(c.SequenceNum)))
	request.SetHeader("Accept", "application/sdp")

	response, err := c.sendRequest(request)
	if err != nil {
		return err
	}

	if response.StatusCode != rtspProtocon.StatusOK {
		return fmt.Errorf("DESCRIBE request failed: %d %s", response.StatusCode, response.StatusText)
	}

	// 解析SDP
	sdp, err := rtspProtocon.ParseSDP(response.Body)
	if err != nil {
		return fmt.Errorf("failed to parse SDP: %v", err)
	}

	c.SDP = sdp
	log.Println("DESCRIBE request successful, SDP parsed")
	c.SequenceNum++
	return nil
}

// 发送SETUP请求
func (c *RTSPClient) Setup() error {
	if c.SDP == nil {
		return fmt.Errorf("SDP not available, call Describe() first")
	}

	// 创建传输信息
	transport := &rtspProtocon.Transport{
		Type:        rtspProtocon.TransportRTP,
		ClientPorts: "5004-5005", // 默认客户端端口
		Mode:        "PLAY",
		Unicast:     true,
	}

	// 如果启用TCP传输
	if c.UseTCPTransport {
		transport.Type = rtspProtocon.TransportInterleaved
		transport.Interleaved = "0-1" // 请求TCP interleaved通道
		transport.ClientPorts = ""    // TCP传输不需要端口
	} else if c.DynamicPorts {
		// 如果启用动态端口，让服务器分配端口
		transport.ClientPorts = "" // 让服务器分配端口
	}

	request := rtspProtocon.NewRequest(rtspProtocon.MethodSetup, c.ServerURL)
	request.SetHeader("User-Agent", "Go-RTSP-Client/1.0")
	request.SetHeader("CSeq", strconv.Itoa(int(c.SequenceNum)))
	request.SetHeader("Transport", transport.String())

	response, err := c.sendRequest(request)
	if err != nil {
		return err
	}

	if response.StatusCode != rtspProtocon.StatusOK {
		return fmt.Errorf("SETUP request failed: %d %s", response.StatusCode, response.StatusText)
	}

	// 解析传输响应
	transportHeader := response.GetHeader("Transport")
	if transportHeader == "" {
		return fmt.Errorf("no Transport header in response")
	}

	serverTransport, err := rtspProtocon.ParseTransport(transportHeader)
	if err != nil {
		return fmt.Errorf("failed to parse transport header: %v", err)
	}

	c.Transport = serverTransport

	// 检查是否使用TCP传输
	if rtspProtocon.IsTCPTransport(serverTransport) {
		c.UseTCPTransport = true
		
		// 解析TCP通道
		if serverTransport.Interleaved != "" {
			ports := strings.Split(serverTransport.Interleaved, "-")
			if len(ports) >= 2 {
				rtpChannel, _ := strconv.Atoi(ports[0])
				rtcpChannel, _ := strconv.Atoi(ports[1])
				
				c.TCPChannel = &rtspProtocon.TCPChannel{
					RTPChannel:  byte(rtpChannel),
					RTCPChannel: byte(rtcpChannel),
					SessionID:   "",
					Created:     time.Now(),
				}
				
				log.Printf("Server allocated TCP channels: RTP=%d, RTCP=%d", rtpChannel, rtcpChannel)
			}
		}
	} else {
		// 解析UDP端口
		if serverTransport.ServerPorts != "" {
			ports := strings.Split(serverTransport.ServerPorts, "-")
			if len(ports) >= 2 {
				rtpPort, _ := strconv.Atoi(ports[0])
				rtcpPort, _ := strconv.Atoi(ports[1])
				
				c.PortAllocation = &rtspProtocon.PortAllocation{
					RTPPort:     rtpPort,
					RTCPPort:    rtcpPort,
					AllocatedAt: time.Now(),
					SessionID:   "",
				}
				
				log.Printf("Server allocated UDP ports: RTP=%d, RTCP=%d", rtpPort, rtcpPort)
			}
		}
	}

	// 获取会话ID
	sessionID := response.GetHeader("Session")
	if sessionID == "" {
		return fmt.Errorf("no Session header in response")
	}

	c.Session = rtspProtocon.NewSession(sessionID)
	c.Session.UpdateState(rtspProtocon.StateReady)
	
	if c.PortAllocation != nil {
		c.PortAllocation.SessionID = sessionID
	}
	
	if c.TCPChannel != nil {
		c.TCPChannel.SessionID = sessionID
	}

	log.Printf("SETUP request successful, Session: %s", sessionID)
	c.SequenceNum++
	return nil
}
 
// 发送PLAY请求
func (c *RTSPClient) Play() error {
	if c.Session == nil {
		return fmt.Errorf("session not available, call Setup() first")
	}

	request := rtspProtocon.NewRequest(rtspProtocon.MethodPlay, c.ServerURL)
	request.SetHeader("User-Agent", "Go-RTSP-Client/1.0")
	request.SetHeader("CSeq", strconv.Itoa(int(c.SequenceNum)))
	request.SetHeader("Session", c.Session.ID)

	response, err := c.sendRequest(request)
	if err != nil {
		return err
	}

	if response.StatusCode != rtspProtocon.StatusOK {
		return fmt.Errorf("PLAY request failed: %d %s", response.StatusCode, response.StatusText)
	}

	c.Session.UpdateState(rtspProtocon.StatePlaying)
	
	// 如果使用TCP传输，启动媒体流接收
	if c.UseTCPTransport {
		if err := c.ReceiveTCPMediaStream(); err != nil {
			log.Printf("Failed to start TCP media stream receiver: %v", err)
		}
	}
	
	log.Println("PLAY request successful, streaming started")
	c.SequenceNum++
	return nil
}

// 发送PAUSE请求
func (c *RTSPClient) Pause() error {
	if c.Session == nil {
		return fmt.Errorf("session not available")
	}

	request := rtspProtocon.NewRequest(rtspProtocon.MethodPause, c.ServerURL)
	request.SetHeader("User-Agent", "Go-RTSP-Client/1.0")
	request.SetHeader("CSeq", strconv.Itoa(int(c.SequenceNum)))
	request.SetHeader("Session", c.Session.ID)

	response, err := c.sendRequest(request)
	if err != nil {
		return err
	}

	if response.StatusCode != rtspProtocon.StatusOK {
		return fmt.Errorf("PAUSE request failed: %d %s", response.StatusCode, response.StatusText)
	}

	c.Session.UpdateState(rtspProtocon.StateReady)
	log.Println("PAUSE request successful")
	c.SequenceNum++
	return nil
}

// 发送TEARDOWN请求
func (c *RTSPClient) Teardown() error {
	if c.Session == nil {
		return nil // 没有会话，无需teardown
	}

	request := rtspProtocon.NewRequest(rtspProtocon.MethodTeardown, c.ServerURL)
	request.SetHeader("User-Agent", "Go-RTSP-Client/1.0")
	request.SetHeader("CSeq", strconv.Itoa(int(c.SequenceNum)))
	request.SetHeader("Session", c.Session.ID)

	response, err := c.sendRequest(request)
	if err != nil {
		return err
	}

	if response.StatusCode != rtspProtocon.StatusOK {
		return fmt.Errorf("TEARDOWN request failed: %d %s", response.StatusCode, response.StatusText)
	}

	c.Session = nil
	log.Println("TEARDOWN request successful")
	c.SequenceNum++
	return nil
}

// 发送请求并接收响应
func (c *RTSPClient) sendRequest(request *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	if !c.Connected {
		return nil, fmt.Errorf("not connected to server")
	}

	// 发送请求
	requestStr := request.String()
	if _, err := c.Writer.WriteString(requestStr); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	if err := c.Writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush request: %v", err)
	}

	// 读取响应
	response, err := c.readResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return response, nil
}

// 读取响应
func (c *RTSPClient) readResponse() (*rtspProtocon.Message, error) {
	var lines []string
	var body string

	// 读取头部
	for {
		line, err := c.Reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		
		if line == "" {
			break
		}
		lines = append(lines, line)
	}

	// 检查是否有消息体
	contentLength := c.getHeaderValue(lines, "Content-Length")
	if contentLength != "" {
		if length, err := strconv.Atoi(contentLength); err == nil && length > 0 {
			bodyBytes := make([]byte, length)
			if _, err := c.Reader.Read(bodyBytes); err != nil {
				return nil, err
			}
			body = string(bodyBytes)
		}
	}

	// 构建消息字符串
	messageStr := strings.Join(lines, "\r\n") + "\r\n\r\n" + body

	// 解析消息
	return rtspProtocon.ParseMessage(messageStr)
}

// 从头部行中获取指定头的值
func (c *RTSPClient) getHeaderValue(lines []string, headerName string) string {
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), strings.ToLower(headerName)+":") {
			return strings.TrimSpace(line[len(headerName)+1:])
		}
	}
	return ""
}

// 解析URL
func (c *RTSPClient) parseURL(url string) (string, int, error) {
	if !rtspProtocon.ValidateRTSPURL(url) {
		return "", 0, fmt.Errorf("invalid RTSP URL format")
	}

	// 移除rtsp://前缀
	url = strings.TrimPrefix(url, "rtsp://")
	
	// 分割主机和端口
	parts := strings.Split(url, ":")
	if len(parts) == 1 {
		return parts[0], 554, nil // 默认RTSP端口
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port number")
	}

	return parts[0], port, nil
}

// 获取客户端统计信息
func (c *RTSPClient) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"connected":        c.Connected,
		"server_url":       c.ServerURL,
		"sequence_num":     c.SequenceNum,
		"dynamic_ports":    c.DynamicPorts,
		"tcp_transport":    c.UseTCPTransport,
	}

	if c.Session != nil {
		stats["session_id"] = c.Session.ID
		stats["session_state"] = c.Session.State
	}

	if c.PortAllocation != nil {
		stats["port_allocation"] = map[string]interface{}{
			"rtp_port":      c.PortAllocation.RTPPort,
			"rtcp_port":     c.PortAllocation.RTCPPort,
			"allocated_at":  c.PortAllocation.AllocatedAt.Format(time.RFC3339),
			"session_id":    c.PortAllocation.SessionID,
		}
	}

	if c.TCPChannel != nil {
		stats["tcp_channel"] = map[string]interface{}{
			"rtp_channel":   c.TCPChannel.RTPChannel,
			"rtcp_channel":  c.TCPChannel.RTCPChannel,
			"created":       c.TCPChannel.Created.Format(time.RFC3339),
			"session_id":    c.TCPChannel.SessionID,
		}
	}

	if c.TCPStats != nil {
		stats["tcp_stats"] = c.TCPStats.GetStats()
	}

	if c.RTPStats != nil {
		stats["rtp_stats"] = map[string]interface{}{
			"packets_sent":     c.RTPStats.PacketsSent,
			"packets_received": c.RTPStats.PacketsReceived,
			"bytes_sent":       c.RTPStats.BytesSent,
			"bytes_received":   c.RTPStats.BytesReceived,
			"loss_rate":        c.RTPStats.GetLossRate(),
		}
	}

	return stats
}

// 设置动态端口模式
func (c *RTSPClient) SetDynamicPorts(enabled bool) {
	c.DynamicPorts = enabled
}

// 设置TCP传输模式
func (c *RTSPClient) SetTCPTransport(enabled bool) {
	c.UseTCPTransport = enabled
}

// 接收TCP媒体流
func (c *RTSPClient) ReceiveTCPMediaStream() error {
	if !c.UseTCPTransport || c.TCPChannel == nil {
		return fmt.Errorf("not using TCP transport")
	}

	log.Println("Starting TCP media stream receiver...")

	go func() {
		for c.Connected && c.Session != nil && c.Session.State == rtspProtocon.StatePlaying {
			// 读取TCP Interleaved帧
			frame, err := rtspProtocon.ReadTCPInterleavedFrame(c.Reader)
			if err != nil {
				if err == io.EOF {
					log.Println("Server disconnected")
				} else {
					log.Printf("Error reading TCP interleaved frame: %v", err)
				}
				break
			}

			// 更新统计
			c.TCPStats.UpdateReceived(frame)

			// 处理RTP包
			if frame.Header.Channel == c.TCPChannel.RTPChannel {
				rtpPacket, err := rtspProtocon.ParseRTPFromInterleavedFrame(frame)
				if err != nil {
					log.Printf("Failed to parse RTP packet: %v", err)
					continue
				}

				// 更新RTP统计
				c.RTPStats.UpdateReceived(
					rtpPacket.Header.SequenceNum,
					rtpPacket.Header.Timestamp,
					len(rtpPacket.Payload),
				)

				log.Printf("Received RTP packet: seq=%d, timestamp=%d, payload=%d bytes",
					rtpPacket.Header.SequenceNum,
					rtpPacket.Header.Timestamp,
					len(rtpPacket.Payload))

				// 这里可以添加媒体数据处理逻辑
				// 例如：解码、显示、保存等

			} else if frame.Header.Channel == c.TCPChannel.RTCPChannel {
				// 处理RTCP包
				rtcpData, err := rtspProtocon.ParseRTCPFromInterleavedFrame(frame)
				if err != nil {
					log.Printf("Failed to parse RTCP packet: %v", err)
					continue
				}

				log.Printf("Received RTCP packet: %d bytes", len(rtcpData))
			}
		}

		log.Println("TCP media stream receiver ended")
	}()

	return nil
}

// 发送TCP RTCP包
func (c *RTSPClient) SendTCPRTCPPacket(data []byte) error {
	if !c.UseTCPTransport || c.TCPChannel == nil {
		return fmt.Errorf("not using TCP transport")
	}

	frame := rtspProtocon.NewTCPInterleavedFrame(c.TCPChannel.RTCPChannel, data)
	
	if err := rtspProtocon.WriteTCPInterleavedFrame(c.Writer, frame); err != nil {
		return fmt.Errorf("failed to send RTCP packet: %v", err)
	}

	// 更新统计
	c.TCPStats.UpdateSent(frame)

	return nil
}

// 完整的RTSP会话流程
func (c *RTSPClient) StartSession() error {
	log.Println("Starting RTSP session...")

	// 1. 连接
	if err := c.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	// 2. OPTIONS
	if err := c.Options(); err != nil {
		return fmt.Errorf("OPTIONS failed: %v", err)
	}

	// 3. DESCRIBE
	if err := c.Describe(); err != nil {
		return fmt.Errorf("DESCRIBE failed: %v", err)
	}

	// 4. SETUP
	if err := c.Setup(); err != nil {
		return fmt.Errorf("SETUP failed: %v", err)
	}

	// 5. PLAY
	if err := c.Play(); err != nil {
		return fmt.Errorf("PLAY failed: %v", err)
	}

	log.Println("RTSP session started successfully")
	return nil
}

func main() {
	// 创建RTSP客户端
	client := NewRTSPClient("rtsp://localhost:8554/test")

	// 启用TCP传输
	client.SetTCPTransport(true)
	log.Println("TCP transport enabled")

	// 启动会话
	if err := client.StartSession(); err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}

	// 打印传输信息
	if client.TCPChannel != nil {
		log.Printf("TCP channels: RTP=%d, RTCP=%d", client.TCPChannel.RTPChannel, client.TCPChannel.RTCPChannel)
	} else if client.PortAllocation != nil {
		log.Printf("UDP ports: RTP=%d, RTCP=%d", client.PortAllocation.RTPPort, client.PortAllocation.RTCPPort)
	}

	// 保持连接一段时间
	log.Println("Streaming for 30 seconds...")
	time.Sleep(30 * time.Second)

	// 暂停流
	log.Println("Pausing stream...")
	if err := client.Pause(); err != nil {
		log.Printf("Failed to pause: %v", err)
	}

	time.Sleep(5 * time.Second)

	// 恢复播放
	log.Println("Resuming stream...")
	if err := client.Play(); err != nil {
		log.Printf("Failed to resume: %v", err)
	}

	time.Sleep(10 * time.Second)

	// 断开连接
	log.Println("Ending session...")
	if err := client.Disconnect(); err != nil {
		log.Printf("Failed to disconnect: %v", err)
	}

	// 打印统计信息
	stats := client.GetStats()
	log.Printf("Client stats: %+v", stats)
}
