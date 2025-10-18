package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// RTSP服务器
type RTSPServer struct {
	Address           string
	Port              int
	Listener          net.Listener
	Sessions          map[string]*rtspProtocon.Session
	SessionsMux       sync.RWMutex
	Streams           map[string]*Stream
	StreamsMux        sync.RWMutex
	PortManager       *rtspProtocon.PortManager
	TCPTransportMgr   *rtspProtocon.TCPTransportManager
	Running           bool
}

// 流信息
type Stream struct {
	ID          string
	Name        string
	SDP         *rtspProtocon.SessionDescription
	RTPPort     int
	RTCPPort    int
	Active      bool
	Created     time.Time
	LastAccess  time.Time
}

// 客户端连接
type ClientConnection struct {
	Conn             net.Conn
	Session          *rtspProtocon.Session
	Server           *RTSPServer
	Reader           *bufio.Reader
	Writer           *bufio.Writer
	RemoteAddr       string
	PortAllocation   *rtspProtocon.PortAllocation
	TCPChannel       *rtspProtocon.TCPChannel
	TCPStats         *rtspProtocon.TCPTransportStats
	UseTCPTransport  bool
}

// 创建新的RTSP服务器
func NewRTSPServer(address string, port int) *RTSPServer {
	return &RTSPServer{
		Address:         address,
		Port:            port,
		Sessions:        make(map[string]*rtspProtocon.Session),
		Streams:         make(map[string]*Stream),
		PortManager:     rtspProtocon.NewDefaultPortManager(),
		TCPTransportMgr: rtspProtocon.NewTCPTransportManager(),
		Running:         false,
	}
}

// 启动服务器
func (s *RTSPServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.Address, s.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start RTSP server: %v", err)
	}

	s.Listener = listener
	s.Running = true

	log.Printf("RTSP Server started on %s", addr)

	// 初始化默认流
	s.initializeDefaultStreams()

	// 接受连接
	for s.Running {
		conn, err := listener.Accept()
		if err != nil {
			if s.Running {
				log.Printf("Error accepting connection: %v", err)
			}
			continue
		}

		// 处理客户端连接
		go s.handleClient(conn)
	}

	return nil
}

// 停止服务器
func (s *RTSPServer) Stop() {
	s.Running = false
	if s.Listener != nil {
		s.Listener.Close()
	}
	log.Println("RTSP Server stopped")
}

// 处理客户端连接
func (s *RTSPServer) handleClient(conn net.Conn) {
	defer conn.Close()

	client := &ClientConnection{
		Conn:            conn,
		Server:          s,
		Reader:          bufio.NewReader(conn),
		Writer:          bufio.NewWriter(conn),
		RemoteAddr:      conn.RemoteAddr().String(),
		TCPStats:        rtspProtocon.NewTCPTransportStats(),
		UseTCPTransport: false,
	}

	log.Printf("New client connected: %s", client.RemoteAddr)

	for {
		// 读取RTSP消息
		message, err := client.readMessage()
		if err != nil {
			log.Printf("Error reading message from %s: %v", client.RemoteAddr, err)
			break
		}

		// 处理RTSP消息
		response, err := s.handleMessage(client, message)
		if err != nil {
			log.Printf("Error handling message: %v", err)
			response = rtspProtocon.NewResponse(rtspProtocon.StatusInternalServerError, rtspProtocon.GetStatusText(rtspProtocon.StatusInternalServerError))
		}

		// 发送响应
		if err := client.writeMessage(response); err != nil {
			log.Printf("Error writing response to %s: %v", client.RemoteAddr, err)
			break
		}
	}

	log.Printf("Client disconnected: %s", client.RemoteAddr)
}

// 读取RTSP消息
func (c *ClientConnection) readMessage() (*rtspProtocon.Message, error) {
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
	if contentLength := c.getHeaderValue(lines, "Content-Length"); contentLength != "" {
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

// 写入RTSP消息
func (c *ClientConnection) writeMessage(message *rtspProtocon.Message) error {
	messageStr := message.String()
	_, err := c.Writer.WriteString(messageStr)
	if err != nil {
		return err
	}
	return c.Writer.Flush()
}

// 从头部行中获取指定头的值
func (c *ClientConnection) getHeaderValue(lines []string, headerName string) string {
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), strings.ToLower(headerName)+":") {
			return strings.TrimSpace(line[len(headerName)+1:])
		}
	}
	return ""
}

// 处理RTSP消息
func (s *RTSPServer) handleMessage(client *ClientConnection, message *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	switch message.Method {
	case rtspProtocon.MethodOptions:
		return s.handleOptions(client, message)
	case rtspProtocon.MethodDescribe:
		return s.handleDescribe(client, message)
	case rtspProtocon.MethodSetup:
		return s.handleSetup(client, message)
	case rtspProtocon.MethodPlay:
		return s.handlePlay(client, message)
	case rtspProtocon.MethodPause:
		return s.handlePause(client, message)
	case rtspProtocon.MethodTeardown:
		return s.handleTeardown(client, message)
	default:
		return rtspProtocon.NewResponse(rtspProtocon.StatusMethodNotAllowed, rtspProtocon.GetStatusText(rtspProtocon.StatusMethodNotAllowed)), nil
	}
}

// 处理OPTIONS请求
func (s *RTSPServer) handleOptions(client *ClientConnection, message *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	response := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
	response.SetHeader("Public", "OPTIONS, DESCRIBE, SETUP, PLAY, PAUSE, TEARDOWN")
	response.SetHeader("Server", "Go-RTSP-Server/1.0")
	return response, nil
}

// 处理DESCRIBE请求
func (s *RTSPServer) handleDescribe(client *ClientConnection, message *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	// 从URL中提取流名称
	streamName := s.extractStreamName(message.URL)
	
	s.StreamsMux.RLock()
	stream, exists := s.Streams[streamName]
	s.StreamsMux.RUnlock()

	if !exists {
		return rtspProtocon.NewResponse(rtspProtocon.StatusNotFound, rtspProtocon.GetStatusText(rtspProtocon.StatusNotFound)), nil
	}

	response := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
	response.SetHeader("Content-Type", "application/sdp")
	response.SetHeader("Content-Length", strconv.Itoa(len(stream.SDP.String())))
	response.Body = stream.SDP.String()
	
	return response, nil
}

// 处理SETUP请求
func (s *RTSPServer) handleSetup(client *ClientConnection, message *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	// 解析传输头
	transportHeader := message.GetHeader("Transport")
	if transportHeader == "" {
		return rtspProtocon.NewResponse(rtspProtocon.StatusBadRequest, rtspProtocon.GetStatusText(rtspProtocon.StatusBadRequest)), nil
	}

	transport, err := rtspProtocon.ParseTransport(transportHeader)
	if err != nil {
		return rtspProtocon.NewResponse(rtspProtocon.StatusBadRequest, rtspProtocon.GetStatusText(rtspProtocon.StatusBadRequest)), nil
	}

	// 创建会话
	sessionID := rtspProtocon.GenerateSessionID()
	session := rtspProtocon.NewSession(sessionID)
	session.URL = message.URL
	session.UpdateState(rtspProtocon.StateReady)

	s.SessionsMux.Lock()
	s.Sessions[sessionID] = session
	s.SessionsMux.Unlock()

	client.Session = session

	// 检查是否使用TCP传输
	if rtspProtocon.IsTCPTransport(transport) {
		// 使用TCP interleaved传输
		client.UseTCPTransport = true
		tcpChannel := s.TCPTransportMgr.AllocateChannel(sessionID)
		client.TCPChannel = tcpChannel
		
		transport.Type = rtspProtocon.TransportInterleaved
		transport.Interleaved = fmt.Sprintf("%d-%d", tcpChannel.RTPChannel, tcpChannel.RTCPChannel)
		
		log.Printf("Allocated TCP channels for session %s: RTP=%d, RTCP=%d", sessionID, tcpChannel.RTPChannel, tcpChannel.RTCPChannel)
	} else {
		// 使用UDP传输
		portAllocation, err := s.PortManager.AllocatePortsWithRetry(sessionID, 3)
		if err != nil {
			log.Printf("Failed to allocate ports for session %s after retries: %v", sessionID, err)
			return rtspProtocon.NewResponse(rtspProtocon.StatusServiceUnavailable, rtspProtocon.GetStatusText(rtspProtocon.StatusServiceUnavailable)), nil
		}

		client.PortAllocation = portAllocation
		transport.ServerPorts = fmt.Sprintf("%d-%d", portAllocation.RTPPort, portAllocation.RTCPPort)
		
		log.Printf("Allocated UDP ports for session %s: RTP=%d, RTCP=%d", sessionID, portAllocation.RTPPort, portAllocation.RTCPPort)
	}

	response := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
	response.SetHeader("Transport", transport.String())
	response.SetHeader("Session", sessionID)
	response.SetHeader("Server", "Go-RTSP-Server/1.0")

	return response, nil
}

// 处理PLAY请求
func (s *RTSPServer) handlePlay(client *ClientConnection, message *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	sessionID := message.GetHeader("Session")
	if sessionID == "" {
		return rtspProtocon.NewResponse(rtspProtocon.StatusBadRequest, rtspProtocon.GetStatusText(rtspProtocon.StatusBadRequest)), nil
	}

	s.SessionsMux.RLock()
	session, exists := s.Sessions[sessionID]
	s.SessionsMux.RUnlock()

	if !exists {
		return rtspProtocon.NewResponse(rtspProtocon.StatusNotFound, rtspProtocon.GetStatusText(rtspProtocon.StatusNotFound)), nil
	}

	session.UpdateState(rtspProtocon.StatePlaying)

	// 如果使用TCP传输，启动媒体流
	if client.UseTCPTransport {
		streamName := s.extractStreamName(session.URL)
		if err := s.SendTCPMediaStream(client, streamName); err != nil {
			log.Printf("Failed to start TCP media stream: %v", err)
		}
		
		// 启动TCP数据处理器
		go s.handleTCPInterleavedData(client)
	}

	response := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
	response.SetHeader("Session", sessionID)
	response.SetHeader("RTP-Info", "url="+session.URL+";seq=0;rtptime=0")

	return response, nil
}

// 处理PAUSE请求
func (s *RTSPServer) handlePause(client *ClientConnection, message *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	sessionID := message.GetHeader("Session")
	if sessionID == "" {
		return rtspProtocon.NewResponse(rtspProtocon.StatusBadRequest, rtspProtocon.GetStatusText(rtspProtocon.StatusBadRequest)), nil
	}

	s.SessionsMux.RLock()
	session, exists := s.Sessions[sessionID]
	s.SessionsMux.RUnlock()

	if !exists {
		return rtspProtocon.NewResponse(rtspProtocon.StatusNotFound, rtspProtocon.GetStatusText(rtspProtocon.StatusNotFound)), nil
	}

	session.UpdateState(rtspProtocon.StateReady)

	response := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
	response.SetHeader("Session", sessionID)

	return response, nil
}

// 处理TEARDOWN请求
func (s *RTSPServer) handleTeardown(client *ClientConnection, message *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	sessionID := message.GetHeader("Session")
	if sessionID != "" {
		s.SessionsMux.Lock()
		delete(s.Sessions, sessionID)
		s.SessionsMux.Unlock()
		
		// 释放端口或TCP通道
		if client.UseTCPTransport && client.TCPChannel != nil {
			s.TCPTransportMgr.ReleaseChannel(sessionID)
			log.Printf("Released TCP channels for session %s: RTP=%d, RTCP=%d", 
				sessionID, client.TCPChannel.RTPChannel, client.TCPChannel.RTCPChannel)
		} else if client.PortAllocation != nil {
			s.PortManager.ReleasePorts(client.PortAllocation)
			log.Printf("Released UDP ports for session %s: RTP=%d, RTCP=%d", 
				sessionID, client.PortAllocation.RTPPort, client.PortAllocation.RTCPPort)
		}
	}

	response := rtspProtocon.NewResponse(rtspProtocon.StatusOK, rtspProtocon.GetStatusText(rtspProtocon.StatusOK))
	response.SetHeader("Session", sessionID)

	return response, nil
}

// 从URL中提取流名称
func (s *RTSPServer) extractStreamName(url string) string {
	// 简单实现：从URL路径中提取流名称
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "default"
}

// 初始化默认流
func (s *RTSPServer) initializeDefaultStreams() {
	// 创建测试视频流
	videoStream := &Stream{
		ID:       "test",
		Name:     "Test Video Stream",
		RTPPort:  5004,
		RTCPPort: 5005,
		Active:   true,
		Created:  time.Now(),
	}

	// 创建SDP
	videoStream.SDP = rtspProtocon.CreateVideoSDP(s.Address, videoStream.RTPPort)

	s.StreamsMux.Lock()
	s.Streams["test"] = videoStream
	s.StreamsMux.Unlock()

	log.Printf("Initialized stream: %s", videoStream.Name)
}

// 添加新流
func (s *RTSPServer) AddStream(id, name string, rtpPort int) error {
	stream := &Stream{
		ID:       id,
		Name:     name,
		RTPPort:  rtpPort,
		RTCPPort: rtpPort + 1,
		Active:   true,
		Created:  time.Now(),
	}

	stream.SDP = rtspProtocon.CreateVideoSDP(s.Address, rtpPort)

	s.StreamsMux.Lock()
	s.Streams[id] = stream
	s.StreamsMux.Unlock()

	log.Printf("Added stream: %s (%s)", name, id)
	return nil
}

// 删除流
func (s *RTSPServer) RemoveStream(id string) error {
	s.StreamsMux.Lock()
	delete(s.Streams, id)
	s.StreamsMux.Unlock()

	log.Printf("Removed stream: %s", id)
	return nil
}

// 获取服务器统计信息
func (s *RTSPServer) GetStats() map[string]interface{} {
	s.SessionsMux.RLock()
	sessionCount := len(s.Sessions)
	s.SessionsMux.RUnlock()

	s.StreamsMux.RLock()
	streamCount := len(s.Streams)
	s.StreamsMux.RUnlock()

	portStats := s.PortManager.GetStats()

	return map[string]interface{}{
		"address":       fmt.Sprintf("%s:%d", s.Address, s.Port),
		"running":       s.Running,
		"session_count": sessionCount,
		"stream_count":  streamCount,
		"port_stats":    portStats,
	}
}

// 设置端口范围
func (s *RTSPServer) SetPortRange(rtpStart, rtpEnd, rtcpStart, rtcpEnd int) error {
	return s.PortManager.SetPortRange(rtpStart, rtpEnd, rtcpStart, rtcpEnd)
}

// 获取端口使用情况
func (s *RTSPServer) GetPortUsage() map[int]bool {
	return s.PortManager.GetPortUsage()
}

// 强制释放端口
func (s *RTSPServer) ForceReleasePort(port int) {
	s.PortManager.ForceReleasePort(port)
}

// 发送TCP媒体流
func (s *RTSPServer) SendTCPMediaStream(client *ClientConnection, streamName string) error {
	if !client.UseTCPTransport || client.TCPChannel == nil {
		return fmt.Errorf("client not using TCP transport")
	}

	s.StreamsMux.RLock()
	stream, exists := s.Streams[streamName]
	s.StreamsMux.RUnlock()

	if !exists {
		return fmt.Errorf("stream %s not found", streamName)
	}

	log.Printf("Starting TCP media stream for session %s", client.Session.ID)

	// 模拟媒体数据发送
	go func() {
		sequenceNum := uint16(1)
		timestamp := rtspProtocon.GenerateRTPTimestamp()
		
		for client.Session.State == rtspProtocon.StatePlaying {
			// 创建模拟的RTP包
			payload := []byte(fmt.Sprintf("Frame %d", sequenceNum))
			rtpPacket := rtspProtocon.NewRTPPacket(
				rtspProtocon.PayloadTypeH264,
				sequenceNum,
				timestamp,
				rtspProtocon.GenerateSSRC(),
				payload,
			)

			// 创建TCP Interleaved帧
			frame := rtspProtocon.CreateRTPInterleavedFrame(client.TCPChannel.RTPChannel, rtpPacket)

			// 发送帧
			if err := rtspProtocon.WriteTCPInterleavedFrame(client.Writer, frame); err != nil {
				log.Printf("Failed to send TCP frame: %v", err)
				break
			}

			// 更新统计
			client.TCPStats.UpdateSent(frame)

			// 模拟帧率 (25fps)
			time.Sleep(40 * time.Millisecond)
			sequenceNum++
			timestamp += 3600 // 90kHz时钟
		}

		log.Printf("TCP media stream ended for session %s", client.Session.ID)
	}()

	return nil
}

// 处理TCP Interleaved数据
func (s *RTSPServer) handleTCPInterleavedData(client *ClientConnection) error {
	if !client.UseTCPTransport {
		return fmt.Errorf("client not using TCP transport")
	}

	log.Printf("Starting TCP interleaved data handler for session %s", client.Session.ID)

	for {
		// 读取TCP Interleaved帧
		frame, err := rtspProtocon.ReadTCPInterleavedFrame(client.Reader)
		if err != nil {
			if err == io.EOF {
				log.Printf("Client disconnected: %s", client.RemoteAddr)
			} else {
				log.Printf("Error reading TCP interleaved frame: %v", err)
			}
			break
		}

		// 更新统计
		client.TCPStats.UpdateReceived(frame)

		// 处理RTP包
		if frame.Header.Channel == client.TCPChannel.RTPChannel {
			rtpPacket, err := rtspProtocon.ParseRTPFromInterleavedFrame(frame)
			if err != nil {
				log.Printf("Failed to parse RTP packet: %v", err)
				continue
			}
			
			log.Printf("Received RTP packet: seq=%d, timestamp=%d", 
				rtpPacket.Header.SequenceNum, rtpPacket.Header.Timestamp)
		} else if frame.Header.Channel == client.TCPChannel.RTCPChannel {
			// 处理RTCP包
			rtcpData, err := rtspProtocon.ParseRTCPFromInterleavedFrame(frame)
			if err != nil {
				log.Printf("Failed to parse RTCP packet: %v", err)
				continue
			}
			
			log.Printf("Received RTCP packet: %d bytes", len(rtcpData))
		}
	}

	return nil
}

func main() {
	// 创建RTSP服务器
	server := NewRTSPServer("0.0.0.0", 8554)

	// 设置自定义端口范围
	err := server.SetPortRange(5000, 5100, 5000, 5100)
	if err != nil {
		log.Printf("Failed to set port range: %v", err)
	} else {
		log.Println("Port range set to 5000-5100")
	}

	// 添加一些测试流
	server.AddStream("camera1", "Camera 1 Stream", 5006)
	server.AddStream("camera2", "Camera 2 Stream", 5008)

	// 打印端口统计信息
	stats := server.GetStats()
	log.Printf("Server stats: %+v", stats)

	// 启动服务器
	log.Println("Starting RTSP Server with dynamic port allocation...")
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
