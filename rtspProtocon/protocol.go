package rtspProtocon

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// RTSP方法定义
const (
	MethodOptions   = "OPTIONS"
	MethodDescribe  = "DESCRIBE"
	MethodSetup     = "SETUP"
	MethodPlay      = "PLAY"
	MethodPause     = "PAUSE"
	MethodTeardown  = "TEARDOWN"
	MethodGetParameter = "GET_PARAMETER"
	MethodSetParameter = "SET_PARAMETER"
)

// RTSP状态码
const (
	StatusOK                    = 200
	StatusBadRequest           = 400
	StatusUnauthorized         = 401
	StatusNotFound             = 404
	StatusMethodNotAllowed     = 405
	StatusNotAcceptable        = 406
	StatusUnsupportedMediaType = 415
	StatusInternalServerError  = 500
	StatusNotImplemented       = 501
	StatusBadGateway           = 502
	StatusServiceUnavailable   = 503
)

// RTSP版本
const RTSPVersion = "RTSP/1.0"

// RTSP消息类型
type MessageType int

const (
	Request MessageType = iota
	Response
)

// RTSP消息结构
type Message struct {
	Type       MessageType
	Method     string
	URL        string
	Version    string
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       string
}

// RTSP会话状态
type SessionState int

const (
	StateInit SessionState = iota
	StateReady
	StatePlaying
	StateRecording
)

// RTSP会话
type Session struct {
	ID       string
	State    SessionState
	URL      string
	Headers  map[string]string
	Created  time.Time
	LastSeen time.Time
}

// RTSP传输类型
type TransportType int

const (
	TransportRTP TransportType = iota
	TransportUDP
	TransportTCP
	TransportInterleaved // TCP interleaved模式
)

// RTSP传输信息
type Transport struct {
	Type        TransportType
	ClientPorts string
	ServerPorts string
	SSRC        string
	Mode        string
	Unicast     bool
	Multicast   bool
	TTL         int
	Destination string
	Interleaved string // TCP interleaved通道 (如: "0-1")
}

// 创建新的RTSP消息
func NewMessage(msgType MessageType) *Message {
	return &Message{
		Type:    msgType,
		Headers: make(map[string]string),
		Version: RTSPVersion,
	}
}

// 创建请求消息
func NewRequest(method, url string) *Message {
	msg := NewMessage(Request)
	msg.Method = method
	msg.URL = url
	return msg
}

// 创建响应消息
func NewResponse(statusCode int, statusText string) *Message {
	msg := NewMessage(Response)
	msg.StatusCode = statusCode
	msg.StatusText = statusText
	return msg
}

// 设置消息头
func (m *Message) SetHeader(key, value string) {
	m.Headers[key] = value
}

// 获取消息头
func (m *Message) GetHeader(key string) string {
	return m.Headers[key]
}

// 序列化消息为字符串
func (m *Message) String() string {
	var sb strings.Builder
	
	if m.Type == Request {
		sb.WriteString(fmt.Sprintf("%s %s %s\r\n", m.Method, m.URL, m.Version))
	} else {
		sb.WriteString(fmt.Sprintf("%s %d %s\r\n", m.Version, m.StatusCode, m.StatusText))
	}
	
	// 添加头部
	for key, value := range m.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	
	sb.WriteString("\r\n")
	
	// 添加消息体
	if m.Body != "" {
		sb.WriteString(m.Body)
	}
	
	return sb.String()
}

// 解析RTSP消息
func ParseMessage(data string) (*Message, error) {
	reader := strings.NewReader(data)
	tp := textproto.NewReader(bufio.NewReader(reader))
	
	// 读取第一行
	line, err := tp.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read first line: %v", err)
	}
	
	msg := &Message{
		Headers: make(map[string]string),
	}
	
	// 解析第一行
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid first line: %s", line)
	}
	
	// 判断是请求还是响应
	if strings.HasPrefix(parts[0], "RTSP/") {
		// 响应消息
		msg.Type = Response
		msg.Version = parts[0]
		msg.StatusCode, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid status code: %s", parts[1])
		}
		msg.StatusText = strings.Join(parts[2:], " ")
	} else {
		// 请求消息
		msg.Type = Request
		msg.Method = parts[0]
		msg.URL = parts[1]
		msg.Version = parts[2]
	}
	
	// 解析头部
	for {
		line, err := tp.ReadLine()
		if err != nil {
			break
		}
		if line == "" {
			break
		}
		
		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			continue
		}
		
		key := strings.TrimSpace(line[:colonIndex])
		value := strings.TrimSpace(line[colonIndex+1:])
		msg.Headers[key] = value
	}
	
	// 读取消息体（如果有）
	body, err := tp.ReadLine()
	if err == nil && body != "" {
		msg.Body = body
	}
	
	return msg, nil
}

// 创建新的会话
func NewSession(id string) *Session {
	return &Session{
		ID:       id,
		State:    StateInit,
		Headers:  make(map[string]string),
		Created:  time.Now(),
		LastSeen: time.Now(),
	}
}

// 更新会话状态
func (s *Session) UpdateState(state SessionState) {
	s.State = state
	s.LastSeen = time.Now()
}

// 解析传输头
func ParseTransport(transportHeader string) (*Transport, error) {
	transport := &Transport{
		Unicast: true, // 默认为单播
	}
	
	parts := strings.Split(transportHeader, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "RTP/AVP") {
			transport.Type = TransportRTP
		} else if strings.Contains(part, "RTP/AVPF") {
			transport.Type = TransportRTP
		} else if strings.Contains(part, "RTP/AVP/TCP") {
			transport.Type = TransportInterleaved
		} else if strings.Contains(part, "client_port=") {
			transport.ClientPorts = strings.Split(part, "=")[1]
		} else if strings.Contains(part, "server_port=") {
			transport.ServerPorts = strings.Split(part, "=")[1]
		} else if strings.Contains(part, "ssrc=") {
			transport.SSRC = strings.Split(part, "=")[1]
		} else if strings.Contains(part, "mode=") {
			transport.Mode = strings.Split(part, "=")[1]
		} else if strings.Contains(part, "unicast") {
			transport.Unicast = true
			transport.Multicast = false
		} else if strings.Contains(part, "multicast") {
			transport.Multicast = true
			transport.Unicast = false
		} else if strings.Contains(part, "ttl=") {
			if ttl, err := strconv.Atoi(strings.Split(part, "=")[1]); err == nil {
				transport.TTL = ttl
			}
		} else if strings.Contains(part, "destination=") {
			transport.Destination = strings.Split(part, "=")[1]
		} else if strings.Contains(part, "interleaved=") {
			transport.Interleaved = strings.Split(part, "=")[1]
		}
	}
	
	return transport, nil
}

// 生成传输头字符串
func (t *Transport) String() string {
	var sb strings.Builder
	
	switch t.Type {
	case TransportRTP:
		sb.WriteString("RTP/AVP")
	case TransportUDP:
		sb.WriteString("UDP")
	case TransportTCP:
		sb.WriteString("TCP")
	case TransportInterleaved:
		sb.WriteString("RTP/AVP/TCP")
	}
	
	if t.Unicast {
		sb.WriteString(";unicast")
	} else if t.Multicast {
		sb.WriteString(";multicast")
	}
	
	if t.ClientPorts != "" {
		sb.WriteString(fmt.Sprintf(";client_port=%s", t.ClientPorts))
	}
	
	if t.ServerPorts != "" {
		sb.WriteString(fmt.Sprintf(";server_port=%s", t.ServerPorts))
	}
	
	if t.SSRC != "" {
		sb.WriteString(fmt.Sprintf(";ssrc=%s", t.SSRC))
	}
	
	if t.Mode != "" {
		sb.WriteString(fmt.Sprintf(";mode=%s", t.Mode))
	}
	
	if t.TTL > 0 {
		sb.WriteString(fmt.Sprintf(";ttl=%d", t.TTL))
	}
	
	if t.Destination != "" {
		sb.WriteString(fmt.Sprintf(";destination=%s", t.Destination))
	}
	
	if t.Interleaved != "" {
		sb.WriteString(fmt.Sprintf(";interleaved=%s", t.Interleaved))
	}
	
	return sb.String()
}

// 生成会话ID
func GenerateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// 验证RTSP URL
func ValidateRTSPURL(url string) bool {
	return strings.HasPrefix(url, "rtsp://")
}

// 获取状态码文本
func GetStatusText(code int) string {
	statusTexts := map[int]string{
		StatusOK:                    "OK",
		StatusBadRequest:           "Bad Request",
		StatusUnauthorized:         "Unauthorized",
		StatusNotFound:             "Not Found",
		StatusMethodNotAllowed:     "Method Not Allowed",
		StatusNotAcceptable:        "Not Acceptable",
		StatusUnsupportedMediaType: "Unsupported Media Type",
		StatusInternalServerError:  "Internal Server Error",
		StatusNotImplemented:       "Not Implemented",
		StatusBadGateway:           "Bad Gateway",
		StatusServiceUnavailable:   "Service Unavailable",
	}
	
	if text, exists := statusTexts[code]; exists {
		return text
	}
	return "Unknown"
}
