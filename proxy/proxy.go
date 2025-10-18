package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"sync"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// RTSP代理配置
type ProxyConfig struct {
	ListenAddress string            // 代理监听地址
	ListenPort    int              // 代理监听端口
	UpstreamServers map[string]string // 上游服务器映射 (path -> server)
	DefaultServer string           // 默认上游服务器
	BufferSize    int              // 缓冲区大小
	Timeout       time.Duration    // 超时时间
	MaxConnections int             // 最大连接数
	EnableLogging bool             // 启用日志
}

// RTSP代理会话
type ProxySession struct {
	ID              string
	ClientConn      net.Conn
	UpstreamConn    net.Conn
	ClientSession   *rtspProtocon.Session
	UpstreamSession *rtspProtocon.Session
	StreamPath      string
	UpstreamURL     string
	Created         time.Time
	LastActivity    time.Time
	Stats           *ProxySessionStats
	MediaProxy      *MediaStreamProxy
	Mutex           sync.RWMutex
}

// 代理会话统计
type ProxySessionStats struct {
	MessagesForwarded uint64
	BytesForwarded    uint64
	MessagesReceived  uint64
	BytesReceived     uint64
	StartTime         time.Time
	LastActivity      time.Time
}

// RTSP代理服务器
type RTSPProxy struct {
	Config        *ProxyConfig
	Listener      net.Listener
	Sessions      map[string]*ProxySession
	SessionsMux   sync.RWMutex
	Running       bool
	Stats         *ProxyStats
	UpstreamPool  *UpstreamConnectionPool
	HealthChecker *HealthChecker
	AuthManager   *AuthManager
	Logger        *Logger
}

// 代理统计信息
type ProxyStats struct {
	TotalSessions     uint64
	ActiveSessions    uint64
	TotalConnections  uint64
	MessagesForwarded uint64
	BytesForwarded    uint64
	StartTime         time.Time
	Mutex             sync.RWMutex
}

// 上游连接池
type UpstreamConnectionPool struct {
	Connections map[string][]net.Conn
	Mutex       sync.RWMutex
	MaxPoolSize int
}

// 创建新的RTSP代理配置
func NewProxyConfig() *ProxyConfig {
	return &ProxyConfig{
		ListenAddress:   "0.0.0.0",
		ListenPort:      8555,
		UpstreamServers: make(map[string]string),
		BufferSize:      4096,
		Timeout:         30 * time.Second,
		MaxConnections:  1000,
		EnableLogging:   true,
	}
}

// 创建新的RTSP代理
func NewRTSPProxy(config *ProxyConfig) *RTSPProxy {
	proxy := &RTSPProxy{
		Config:       config,
		Sessions:     make(map[string]*ProxySession),
		Running:      false,
		Stats:        &ProxyStats{StartTime: time.Now()},
		UpstreamPool: NewUpstreamConnectionPool(),
	}
	
	// 初始化日志记录器
	logDir := "logs"
	if logger, err := NewLogger(logDir, LogLevelInfo, true); err == nil {
		proxy.Logger = logger
		proxy.Logger.Info("Logger initialized successfully")
	} else {
		log.Printf("Failed to initialize logger: %v", err)
		// 创建一个简单的默认日志记录器
		proxy.Logger = &Logger{Level: LogLevelInfo, EnableAudit: false}
	}
	
	// 初始化健康检查器
	proxy.HealthChecker = NewHealthChecker(proxy)
	
	// 初始化认证管理器
	proxy.AuthManager = NewAuthManager()
	proxy.AuthManager.CreateSampleUsers()
	proxy.AuthManager.CreateDefaultACLRules()
	
	proxy.Logger.Info("RTSP Proxy initialized successfully")
	return proxy
}

// 创建新的代理会话
func NewProxySession(id string, clientConn net.Conn) *ProxySession {
	session := &ProxySession{
		ID:           id,
		ClientConn:   clientConn,
		Created:      time.Now(),
		LastActivity: time.Now(),
		Stats: &ProxySessionStats{
			StartTime:    time.Now(),
			LastActivity: time.Now(),
		},
	}
	session.MediaProxy = NewMediaStreamProxy(session)
	return session
}

// 创建新的上游连接池
func NewUpstreamConnectionPool() *UpstreamConnectionPool {
	return &UpstreamConnectionPool{
		Connections: make(map[string][]net.Conn),
		MaxPoolSize: 10,
	}
}

// 启动RTSP代理
func (p *RTSPProxy) Start() error {
	addr := fmt.Sprintf("%s:%d", p.Config.ListenAddress, p.Config.ListenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start RTSP proxy: %v", err)
	}

	p.Listener = listener
	p.Running = true

	log.Printf("RTSP Proxy started on %s", addr)
	log.Printf("Upstream servers: %+v", p.Config.UpstreamServers)

	// 接受连接
	for p.Running {
		conn, err := listener.Accept()
		if err != nil {
			if p.Running {
				log.Printf("Error accepting connection: %v", err)
			}
			continue
		}

		// 处理客户端连接
		go p.handleClientConnection(conn)
	}

	return nil
}

// 停止RTSP代理
func (p *RTSPProxy) Stop() {
	p.Running = false
	if p.Listener != nil {
		p.Listener.Close()
	}

	// 关闭所有会话
	p.SessionsMux.Lock()
	for _, session := range p.Sessions {
		session.Close()
	}
	p.SessionsMux.Unlock()

	log.Println("RTSP Proxy stopped")
}

// 处理客户端连接
func (p *RTSPProxy) handleClientConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// 创建代理会话
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	session := NewProxySession(sessionID, clientConn)

	// 添加到会话列表
	p.SessionsMux.Lock()
	p.Sessions[sessionID] = session
	p.SessionsMux.Unlock()

	// 更新统计
	p.Stats.Mutex.Lock()
	p.Stats.TotalConnections++
	p.Stats.ActiveSessions++
	p.Stats.Mutex.Unlock()

	log.Printf("New client connected: %s (session: %s)", clientConn.RemoteAddr().String(), sessionID)

	// 处理RTSP会话
	err := p.handleRTSPSession(session)
	if err != nil {
		log.Printf("Error handling RTSP session %s: %v", sessionID, err)
	}

	// 清理会话
	p.SessionsMux.Lock()
	delete(p.Sessions, sessionID)
	p.SessionsMux.Unlock()

	// 更新统计
	p.Stats.Mutex.Lock()
	p.Stats.ActiveSessions--
	p.Stats.Mutex.Unlock()

	log.Printf("Client disconnected: %s (session: %s)", clientConn.RemoteAddr().String(), sessionID)
}

// 处理RTSP会话
func (p *RTSPProxy) handleRTSPSession(session *ProxySession) error {
	clientReader := NewRTSPReader(session.ClientConn)
	clientWriter := NewRTSPWriter(session.ClientConn)

	for {
		// 读取客户端请求
		request, err := clientReader.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read client request: %v", err)
		}

		// 更新会话活动时间
		session.UpdateActivity()

		// 处理请求
		response, err := p.processRequest(session, request)
		if err != nil {
			log.Printf("Error processing request: %v", err)
			response = rtspProtocon.NewResponse(rtspProtocon.StatusInternalServerError, rtspProtocon.GetStatusText(rtspProtocon.StatusInternalServerError))
		}

		// 发送响应给客户端
		if err := clientWriter.WriteMessage(response); err != nil {
			return fmt.Errorf("failed to write response to client: %v", err)
		}

		// 更新统计
		session.UpdateStats(response)
		p.updateProxyStats(response)
	}

	return nil
}

// 处理RTSP请求
func (p *RTSPProxy) processRequest(session *ProxySession, request *rtspProtocon.Message) (*rtspProtocon.Message, error) {
	// 确定上游服务器
	upstreamURL, err := p.resolveUpstreamServer(request.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve upstream server: %v", err)
	}

	// 建立上游连接（如果还没有）
	if session.UpstreamConn == nil {
		if err := p.connectToUpstream(session, upstreamURL); err != nil {
			return nil, fmt.Errorf("failed to connect to upstream: %v", err)
		}
	}

	// 转发请求到上游
	upstreamWriter := NewRTSPWriter(session.UpstreamConn)
	if err := upstreamWriter.WriteMessage(request); err != nil {
		return nil, fmt.Errorf("failed to forward request to upstream: %v", err)
	}

	// 读取上游响应
	upstreamReader := NewRTSPReader(session.UpstreamConn)
	response, err := upstreamReader.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read upstream response: %v", err)
	}

	// 处理响应（可能需要修改URL等）
	modifiedResponse := p.modifyResponse(session, response)

	// 如果是PLAY请求，启动媒体流代理
	if request.Method == rtspProtocon.MethodPlay {
		if err := p.startMediaProxy(session, request, response); err != nil {
			log.Printf("Failed to start media proxy: %v", err)
		}
	}

	return modifiedResponse, nil
}

// 解析上游服务器
func (p *RTSPProxy) resolveUpstreamServer(requestURL string) (string, error) {
	// 解析请求URL
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("invalid request URL: %v", err)
	}

	// 查找匹配的上游服务器
	path := parsedURL.Path
	if upstreamServer, exists := p.Config.UpstreamServers[path]; exists {
		return upstreamServer, nil
	}

	// 使用默认服务器
	if p.Config.DefaultServer != "" {
		return p.Config.DefaultServer, nil
	}

	return "", fmt.Errorf("no upstream server configured for path: %s", path)
}

// 连接到上游服务器
func (p *RTSPProxy) connectToUpstream(session *ProxySession, upstreamURL string) error {
	// 解析上游URL
	parsedURL, err := url.Parse(upstreamURL)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %v", err)
	}

	// 连接到上游服务器
	upstreamConn, err := net.DialTimeout("tcp", parsedURL.Host, p.Config.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to upstream server: %v", err)
	}

	session.UpstreamConn = upstreamConn
	session.UpstreamURL = upstreamURL

	log.Printf("Connected to upstream server: %s", upstreamURL)
	return nil
}

// 修改响应
func (p *RTSPProxy) modifyResponse(session *ProxySession, response *rtspProtocon.Message) *rtspProtocon.Message {
	// 修改响应中的URL，使其指向代理服务器
	if response.GetHeader("Location") != "" {
		// 修改Location头
		originalLocation := response.GetHeader("Location")
		modifiedLocation := p.modifyURL(originalLocation)
		response.SetHeader("Location", modifiedLocation)
	}

	// 修改RTP-Info头中的URL
	if response.GetHeader("RTP-Info") != "" {
		rtpInfo := response.GetHeader("RTP-Info")
		modifiedRTPInfo := p.modifyRTPInfo(rtpInfo)
		response.SetHeader("RTP-Info", modifiedRTPInfo)
	}

	return response
}

// 修改URL
func (p *RTSPProxy) modifyURL(originalURL string) string {
	// 将上游服务器URL替换为代理服务器URL
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	// 替换主机和端口
	parsedURL.Host = fmt.Sprintf("%s:%d", p.Config.ListenAddress, p.Config.ListenPort)
	return parsedURL.String()
}

// 修改RTP-Info
func (p *RTSPProxy) modifyRTPInfo(rtpInfo string) string {
	// 简单的RTP-Info修改，实际应用中可能需要更复杂的解析
	// 这里只是示例，实际实现需要解析RTP-Info格式
	return rtpInfo
}

// 更新会话活动时间
func (s *ProxySession) UpdateActivity() {
	s.Mutex.Lock()
	s.LastActivity = time.Now()
	s.Stats.LastActivity = time.Now()
	s.Mutex.Unlock()
}

// 更新会话统计
func (s *ProxySession) UpdateStats(response *rtspProtocon.Message) {
	s.Mutex.Lock()
	s.Stats.MessagesReceived++
	s.Stats.BytesReceived += uint64(len(response.String()))
	s.Mutex.Unlock()
}

// 更新代理统计
func (p *RTSPProxy) updateProxyStats(response *rtspProtocon.Message) {
	p.Stats.Mutex.Lock()
	p.Stats.MessagesForwarded++
	p.Stats.BytesForwarded += uint64(len(response.String()))
	p.Stats.Mutex.Unlock()
}

// 关闭会话
func (s *ProxySession) Close() {
	if s.ClientConn != nil {
		s.ClientConn.Close()
	}
	if s.UpstreamConn != nil {
		s.UpstreamConn.Close()
	}
	if s.MediaProxy != nil {
		s.MediaProxy.Stop()
	}
}

// 获取代理统计信息
func (p *RTSPProxy) GetStats() map[string]interface{} {
	p.Stats.Mutex.RLock()
	defer p.Stats.Mutex.RUnlock()

	p.SessionsMux.RLock()
	activeSessions := len(p.Sessions)
	p.SessionsMux.RUnlock()

	return map[string]interface{}{
		"listen_address":      fmt.Sprintf("%s:%d", p.Config.ListenAddress, p.Config.ListenPort),
		"running":            p.Running,
		"total_sessions":     p.Stats.TotalSessions,
		"active_sessions":    activeSessions,
		"total_connections":  p.Stats.TotalConnections,
		"messages_forwarded": p.Stats.MessagesForwarded,
		"bytes_forwarded":    p.Stats.BytesForwarded,
		"uptime":            time.Since(p.Stats.StartTime).String(),
	}
}

// 添加上游服务器
func (p *RTSPProxy) AddUpstreamServer(path, serverURL string) {
	p.Config.UpstreamServers[path] = serverURL
	log.Printf("Added upstream server: %s -> %s", path, serverURL)
}

// 移除上游服务器
func (p *RTSPProxy) RemoveUpstreamServer(path string) {
	delete(p.Config.UpstreamServers, path)
	log.Printf("Removed upstream server: %s", path)
}

// 设置默认上游服务器
func (p *RTSPProxy) SetDefaultUpstreamServer(serverURL string) {
	p.Config.DefaultServer = serverURL
	log.Printf("Set default upstream server: %s", serverURL)
}

// 启动媒体流代理
func (p *RTSPProxy) startMediaProxy(session *ProxySession, request, response *rtspProtocon.Message) error {
	if session.MediaProxy == nil {
		return fmt.Errorf("media proxy not initialized")
	}

	// 检查传输模式
	transportHeader := response.GetHeader("Transport")
	if transportHeader != "" {
		transport, err := rtspProtocon.ParseTransport(transportHeader)
		if err != nil {
			return fmt.Errorf("failed to parse transport header: %v", err)
		}

		// 设置传输模式
		session.MediaProxy.SetTransportMode(rtspProtocon.IsTCPTransport(transport))
	}

	// 启动媒体流代理
	if err := session.MediaProxy.Start(); err != nil {
		return fmt.Errorf("failed to start media proxy: %v", err)
	}

	log.Printf("Media proxy started for session %s", session.ID)
	return nil
}

// 停止媒体流代理
func (p *RTSPProxy) stopMediaProxy(session *ProxySession) {
	if session.MediaProxy != nil {
		session.MediaProxy.Stop()
		log.Printf("Media proxy stopped for session %s", session.ID)
	}
}
