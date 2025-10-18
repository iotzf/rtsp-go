package rtspProtocon

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"
)

// TCP Interleaved帧头
type TCPInterleavedHeader struct {
	Magic     byte   // '$' (0x24)
	Channel   byte   // 通道号 (0=RTP, 1=RTCP)
	Length    uint16 // 数据长度
}

// TCP Interleaved帧
type TCPInterleavedFrame struct {
	Header TCPInterleavedHeader
	Data   []byte
}

// TCP传输通道
type TCPChannel struct {
	RTPChannel  byte
	RTCPChannel byte
	SessionID   string
	Created     time.Time
}

// TCP传输管理器
type TCPTransportManager struct {
	Channels map[string]*TCPChannel
	NextChannel byte
}

// 创建新的TCP传输管理器
func NewTCPTransportManager() *TCPTransportManager {
	return &TCPTransportManager{
		Channels: make(map[string]*TCPChannel),
		NextChannel: 0,
	}
}

// 分配TCP通道
func (tm *TCPTransportManager) AllocateChannel(sessionID string) *TCPChannel {
	channel := &TCPChannel{
		RTPChannel:  tm.NextChannel,
		RTCPChannel: tm.NextChannel + 1,
		SessionID:   sessionID,
		Created:     time.Now(),
	}
	
	tm.Channels[sessionID] = channel
	tm.NextChannel += 2
	
	return channel
}

// 释放TCP通道
func (tm *TCPTransportManager) ReleaseChannel(sessionID string) {
	delete(tm.Channels, sessionID)
}

// 获取通道信息
func (tm *TCPTransportManager) GetChannel(sessionID string) *TCPChannel {
	return tm.Channels[sessionID]
}

// 创建TCP Interleaved帧
func NewTCPInterleavedFrame(channel byte, data []byte) *TCPInterleavedFrame {
	return &TCPInterleavedFrame{
		Header: TCPInterleavedHeader{
			Magic:   0x24, // '$'
			Channel: channel,
			Length:  uint16(len(data)),
		},
		Data: data,
	}
}

// 序列化TCP Interleaved帧
func (frame *TCPInterleavedFrame) Marshal() []byte {
	data := make([]byte, 4+len(frame.Data))
	
	// 写入帧头
	data[0] = frame.Header.Magic
	data[1] = frame.Header.Channel
	binary.BigEndian.PutUint16(data[2:4], frame.Header.Length)
	
	// 写入数据
	copy(data[4:], frame.Data)
	
	return data
}

// 从字节数组解析TCP Interleaved帧
func ParseTCPInterleavedFrame(data []byte) (*TCPInterleavedFrame, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("TCP interleaved frame too short")
	}
	
	if data[0] != 0x24 {
		return nil, fmt.Errorf("invalid magic byte: expected 0x24, got 0x%02x", data[0])
	}
	
	length := binary.BigEndian.Uint16(data[2:4])
	if len(data) < int(4+length) {
		return nil, fmt.Errorf("incomplete frame: expected %d bytes, got %d", 4+length, len(data))
	}
	
	frame := &TCPInterleavedFrame{
		Header: TCPInterleavedHeader{
			Magic:   data[0],
			Channel: data[1],
			Length:  length,
		},
		Data: make([]byte, length),
	}
	
	copy(frame.Data, data[4:4+length])
	
	return frame, nil
}

// 从连接读取TCP Interleaved帧
func ReadTCPInterleavedFrame(reader io.Reader) (*TCPInterleavedFrame, error) {
	// 读取帧头 (4字节)
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, fmt.Errorf("failed to read frame header: %v", err)
	}
	
	if header[0] != 0x24 {
		return nil, fmt.Errorf("invalid magic byte: expected 0x24, got 0x%02x", header[0])
	}
	
	length := binary.BigEndian.Uint16(header[2:4])
	
	// 读取数据
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, fmt.Errorf("failed to read frame data: %v", err)
	}
	
	return &TCPInterleavedFrame{
		Header: TCPInterleavedHeader{
			Magic:   header[0],
			Channel: header[1],
			Length:  length,
		},
		Data: data,
	}, nil
}

// 向连接写入TCP Interleaved帧
func WriteTCPInterleavedFrame(writer io.Writer, frame *TCPInterleavedFrame) error {
	data := frame.Marshal()
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write TCP interleaved frame: %v", err)
	}
	return nil
}

// 创建RTP包的TCP Interleaved帧
func CreateRTPInterleavedFrame(channel byte, rtpPacket *RTPPacket) *TCPInterleavedFrame {
	rtpData := rtpPacket.Marshal()
	return NewTCPInterleavedFrame(channel, rtpData)
}

// 从TCP Interleaved帧解析RTP包
func ParseRTPFromInterleavedFrame(frame *TCPInterleavedFrame) (*RTPPacket, error) {
	if frame.Header.Channel%2 != 0 {
		return nil, fmt.Errorf("invalid RTP channel: %d (should be even)", frame.Header.Channel)
	}
	
	return ParseRTPPacket(frame.Data)
}

// 从TCP Interleaved帧解析RTCP包
func ParseRTCPFromInterleavedFrame(frame *TCPInterleavedFrame) ([]byte, error) {
	if frame.Header.Channel%2 != 1 {
		return nil, fmt.Errorf("invalid RTCP channel: %d (should be odd)", frame.Header.Channel)
	}
	
	return frame.Data, nil
}

// TCP传输统计
type TCPTransportStats struct {
	FramesSent     uint64
	FramesReceived uint64
	BytesSent      uint64
	BytesReceived  uint64
	RTPFrames      uint64
	RTCPFrames     uint64
	LastActivity   time.Time
}

// 创建TCP传输统计
func NewTCPTransportStats() *TCPTransportStats {
	return &TCPTransportStats{
		LastActivity: time.Now(),
	}
}

// 更新发送统计
func (stats *TCPTransportStats) UpdateSent(frame *TCPInterleavedFrame) {
	stats.FramesSent++
	stats.BytesSent += uint64(len(frame.Data) + 4)
	stats.LastActivity = time.Now()
	
	if frame.Header.Channel%2 == 0 {
		stats.RTPFrames++
	} else {
		stats.RTCPFrames++
	}
}

// 更新接收统计
func (stats *TCPTransportStats) UpdateReceived(frame *TCPInterleavedFrame) {
	stats.FramesReceived++
	stats.BytesReceived += uint64(len(frame.Data) + 4)
	stats.LastActivity = time.Now()
	
	if frame.Header.Channel%2 == 0 {
		stats.RTPFrames++
	} else {
		stats.RTCPFrames++
	}
}

// 获取统计信息
func (stats *TCPTransportStats) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"frames_sent":     stats.FramesSent,
		"frames_received": stats.FramesReceived,
		"bytes_sent":      stats.BytesSent,
		"bytes_received":  stats.BytesReceived,
		"rtp_frames":      stats.RTPFrames,
		"rtcp_frames":     stats.RTCPFrames,
		"last_activity":   stats.LastActivity.Format(time.RFC3339),
	}
}

// TCP传输模式检测
func IsTCPTransport(transport *Transport) bool {
	return transport.Type == TransportInterleaved || transport.Type == TransportTCP
}

// 获取TCP通道号
func GetTCPChannel(transport *Transport, isRTP bool) byte {
	if transport.Interleaved != "" {
		// 解析interleaved字段 (如: "0-1")
		parts := strings.Split(transport.Interleaved, "-")
		if len(parts) >= 2 {
			if isRTP {
				if channel, err := strconv.Atoi(parts[0]); err == nil {
					return byte(channel)
				}
			} else {
				if channel, err := strconv.Atoi(parts[1]); err == nil {
					return byte(channel)
				}
			}
		}
	}
	
	// 默认通道号
	if isRTP {
		return 0
	}
	return 1
}

// 验证TCP通道号
func ValidateTCPChannel(channel byte) bool {
	return channel >= 0 && channel <= 255
}

// TCP传输错误类型
type TCPTransportError struct {
	Message string
	Channel byte
	Err     error
}

func (e *TCPTransportError) Error() string {
	return fmt.Sprintf("TCP transport error on channel %d: %s: %v", e.Channel, e.Message, e.Err)
}

// 创建TCP传输错误
func NewTCPTransportError(channel byte, message string, err error) *TCPTransportError {
	return &TCPTransportError{
		Message: message,
		Channel: channel,
		Err:     err,
	}
}
