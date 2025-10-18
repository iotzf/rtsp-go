package rtspProtocon

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SDP媒体类型
type MediaType string

const (
	MediaVideo MediaType = "video"
	MediaAudio MediaType = "audio"
)

// SDP传输协议
type TransportProtocol string

const (
	ProtocolRTPAVP TransportProtocol = "RTP/AVP"
	ProtocolRTPAVPF TransportProtocol = "RTP/AVPF"
)

// SDP会话描述
type SessionDescription struct {
	Version     int
	Origin      *Origin
	SessionName string
	SessionInfo string
	URI         string
	Email       string
	Phone       string
	Connection  *Connection
	Bandwidth   []*Bandwidth
	Timing      *Timing
	Attributes  map[string]string
	Media       []*MediaDescription
}

// SDP Origin
type Origin struct {
	Username       string
	SessID         int64
	SessVersion    int64
	NetType        string
	AddrType       string
	UnicastAddress string
}

// SDP Connection
type Connection struct {
	NetType        string
	AddrType       string
	ConnectionAddr string
}

// SDP Bandwidth
type Bandwidth struct {
	Type  string
	Value int
}

// SDP Timing
type Timing struct {
	StartTime int64
	StopTime  int64
}

// SDP Media Description
type MediaDescription struct {
	MediaType    MediaType
	Port         int
	Protocol     TransportProtocol
	Format       string
	Title        string
	Connection   *Connection
	Bandwidth    []*Bandwidth
	Attributes   map[string]string
	RTPMap       map[int]*RTPMap
}

// RTP Map
type RTPMap struct {
	PayloadType int
	Encoding    string
	ClockRate   int
	Parameters  string
}

// 创建新的SDP会话描述
func NewSessionDescription() *SessionDescription {
	return &SessionDescription{
		Version:    0,
		Attributes: make(map[string]string),
		Media:      make([]*MediaDescription, 0),
		Bandwidth: make([]*Bandwidth, 0),
	}
}

// 创建新的媒体描述
func NewMediaDescription(mediaType MediaType, port int, protocol TransportProtocol, format string) *MediaDescription {
	return &MediaDescription{
		MediaType:  mediaType,
		Port:       port,
		Protocol:   protocol,
		Format:     format,
		Attributes: make(map[string]string),
		RTPMap:     make(map[int]*RTPMap),
		Bandwidth:  make([]*Bandwidth, 0),
	}
}

// 序列化SDP为字符串
func (sdp *SessionDescription) String() string {
	var sb strings.Builder
	
	// v= (version)
	sb.WriteString(fmt.Sprintf("v=%d\r\n", sdp.Version))
	
	// o= (origin)
	if sdp.Origin != nil {
		sb.WriteString(fmt.Sprintf("o=%s %d %d %s %s %s\r\n",
			sdp.Origin.Username,
			sdp.Origin.SessID,
			sdp.Origin.SessVersion,
			sdp.Origin.NetType,
			sdp.Origin.AddrType,
			sdp.Origin.UnicastAddress))
	}
	
	// s= (session name)
	sb.WriteString(fmt.Sprintf("s=%s\r\n", sdp.SessionName))
	
	// i= (session information)
	if sdp.SessionInfo != "" {
		sb.WriteString(fmt.Sprintf("i=%s\r\n", sdp.SessionInfo))
	}
	
	// u= (URI)
	if sdp.URI != "" {
		sb.WriteString(fmt.Sprintf("u=%s\r\n", sdp.URI))
	}
	
	// e= (email)
	if sdp.Email != "" {
		sb.WriteString(fmt.Sprintf("e=%s\r\n", sdp.Email))
	}
	
	// p= (phone)
	if sdp.Phone != "" {
		sb.WriteString(fmt.Sprintf("p=%s\r\n", sdp.Phone))
	}
	
	// c= (connection)
	if sdp.Connection != nil {
		sb.WriteString(fmt.Sprintf("c=%s %s %s\r\n",
			sdp.Connection.NetType,
			sdp.Connection.AddrType,
			sdp.Connection.ConnectionAddr))
	}
	
	// b= (bandwidth)
	for _, bw := range sdp.Bandwidth {
		sb.WriteString(fmt.Sprintf("b=%s:%d\r\n", bw.Type, bw.Value))
	}
	
	// t= (timing)
	if sdp.Timing != nil {
		sb.WriteString(fmt.Sprintf("t=%d %d\r\n", sdp.Timing.StartTime, sdp.Timing.StopTime))
	}
	
	// a= (attributes)
	for key, value := range sdp.Attributes {
		if value == "" {
			sb.WriteString(fmt.Sprintf("a=%s\r\n", key))
		} else {
			sb.WriteString(fmt.Sprintf("a=%s:%s\r\n", key, value))
		}
	}
	
	// m= (media)
	for _, media := range sdp.Media {
		sb.WriteString(media.String())
	}
	
	return sb.String()
}

// 序列化媒体描述为字符串
func (media *MediaDescription) String() string {
	var sb strings.Builder
	
	// m= (media)
	sb.WriteString(fmt.Sprintf("m=%s %d %s %s\r\n",
		string(media.MediaType),
		media.Port,
		string(media.Protocol),
		media.Format))
	
	// i= (media title)
	if media.Title != "" {
		sb.WriteString(fmt.Sprintf("i=%s\r\n", media.Title))
	}
	
	// c= (connection)
	if media.Connection != nil {
		sb.WriteString(fmt.Sprintf("c=%s %s %s\r\n",
			media.Connection.NetType,
			media.Connection.AddrType,
			media.Connection.ConnectionAddr))
	}
	
	// b= (bandwidth)
	for _, bw := range media.Bandwidth {
		sb.WriteString(fmt.Sprintf("b=%s:%d\r\n", bw.Type, bw.Value))
	}
	
	// a= (attributes)
	for key, value := range media.Attributes {
		if value == "" {
			sb.WriteString(fmt.Sprintf("a=%s\r\n", key))
		} else {
			sb.WriteString(fmt.Sprintf("a=%s:%s\r\n", key, value))
		}
	}
	
	// a=rtpmap (RTP map)
	for _, rtpMap := range media.RTPMap {
		sb.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d%s\r\n",
			rtpMap.PayloadType,
			rtpMap.Encoding,
			rtpMap.ClockRate,
			rtpMap.Parameters))
	}
	
	return sb.String()
}

// 解析SDP字符串
func ParseSDP(sdpStr string) (*SessionDescription, error) {
	sdp := NewSessionDescription()
	lines := strings.Split(sdpStr, "\r\n")
	
	var currentMedia *MediaDescription
	
	for _, line := range lines {
		if len(line) < 2 || line[1] != '=' {
			continue
		}
		
		key := line[0]
		value := line[2:]
		
		switch key {
		case 'v':
			// version
			if v, err := strconv.Atoi(value); err == nil {
				sdp.Version = v
			}
			
		case 'o':
			// origin
			parts := strings.Split(value, " ")
			if len(parts) >= 6 {
				sessID, _ := strconv.ParseInt(parts[1], 10, 64)
				sessVersion, _ := strconv.ParseInt(parts[2], 10, 64)
				sdp.Origin = &Origin{
					Username:       parts[0],
					SessID:         sessID,
					SessVersion:    sessVersion,
					NetType:        parts[3],
					AddrType:       parts[4],
					UnicastAddress: parts[5],
				}
			}
			
		case 's':
			// session name
			sdp.SessionName = value
			
		case 'i':
			// session information
			if currentMedia == nil {
				sdp.SessionInfo = value
			} else {
				currentMedia.Title = value
			}
			
		case 'u':
			// URI
			sdp.URI = value
			
		case 'e':
			// email
			sdp.Email = value
			
		case 'p':
			// phone
			sdp.Phone = value
			
		case 'c':
			// connection
			parts := strings.Split(value, " ")
			if len(parts) >= 3 {
				conn := &Connection{
					NetType:        parts[0],
					AddrType:       parts[1],
					ConnectionAddr: parts[2],
				}
				if currentMedia == nil {
					sdp.Connection = conn
				} else {
					currentMedia.Connection = conn
				}
			}
			
		case 'b':
			// bandwidth
			parts := strings.Split(value, ":")
			if len(parts) == 2 {
				if bw, err := strconv.Atoi(parts[1]); err == nil {
					bwObj := &Bandwidth{
						Type:  parts[0],
						Value: bw,
					}
					if currentMedia == nil {
						sdp.Bandwidth = append(sdp.Bandwidth, bwObj)
					} else {
						currentMedia.Bandwidth = append(currentMedia.Bandwidth, bwObj)
					}
				}
			}
			
		case 't':
			// timing
			parts := strings.Split(value, " ")
			if len(parts) >= 2 {
				startTime, _ := strconv.ParseInt(parts[0], 10, 64)
				stopTime, _ := strconv.ParseInt(parts[1], 10, 64)
				sdp.Timing = &Timing{
					StartTime: startTime,
					StopTime:  stopTime,
				}
			}
			
		case 'a':
			// attributes
			parts := strings.Split(value, ":")
			key := parts[0]
			var val string
			if len(parts) > 1 {
				val = parts[1]
			}
			
			if strings.HasPrefix(key, "rtpmap") {
				// RTP map
				if currentMedia != nil && len(parts) >= 2 {
					rtpParts := strings.Split(parts[1], " ")
					if len(rtpParts) >= 2 {
						payloadType, _ := strconv.Atoi(rtpParts[0])
						encodingParts := strings.Split(rtpParts[1], "/")
						clockRate, _ := strconv.Atoi(encodingParts[1])
						
						var parameters string
						if len(encodingParts) > 2 {
							parameters = "/" + strings.Join(encodingParts[2:], "/")
						}
						
						currentMedia.RTPMap[payloadType] = &RTPMap{
							PayloadType: payloadType,
							Encoding:    encodingParts[0],
							ClockRate:   clockRate,
							Parameters:  parameters,
						}
					}
				}
			} else {
				// 普通属性
				if currentMedia == nil {
					sdp.Attributes[key] = val
				} else {
					currentMedia.Attributes[key] = val
				}
			}
			
		case 'm':
			// media
			parts := strings.Split(value, " ")
			if len(parts) >= 4 {
				port, _ := strconv.Atoi(parts[1])
				media := NewMediaDescription(
					MediaType(parts[0]),
					port,
					TransportProtocol(parts[2]),
					parts[3],
				)
				sdp.Media = append(sdp.Media, media)
				currentMedia = media
			}
		}
	}
	
	return sdp, nil
}

// 创建简单的视频SDP
func CreateVideoSDP(serverIP string, port int) *SessionDescription {
	sdp := NewSessionDescription()
	sdp.Version = 0
	sdp.Origin = &Origin{
		Username:       "-",
		SessID:         time.Now().Unix(),
		SessVersion:    time.Now().Unix(),
		NetType:        "IN",
		AddrType:       "IP4",
		UnicastAddress: serverIP,
	}
	sdp.SessionName = "RTSP Stream"
	sdp.Connection = &Connection{
		NetType:        "IN",
		AddrType:       "IP4",
		ConnectionAddr: serverIP,
	}
	sdp.Timing = &Timing{
		StartTime: 0,
		StopTime:  0,
	}
	
	// 添加视频媒体
	videoMedia := NewMediaDescription(MediaVideo, port, ProtocolRTPAVP, "96")
	videoMedia.Connection = &Connection{
		NetType:        "IN",
		AddrType:       "IP4",
		ConnectionAddr: serverIP,
	}
	videoMedia.RTPMap[96] = &RTPMap{
		PayloadType: 96,
		Encoding:    "H264",
		ClockRate:   90000,
		Parameters:  "",
	}
	videoMedia.Attributes["control"] = "track1"
	
	sdp.Media = append(sdp.Media, videoMedia)
	
	return sdp
}
