package rtspProtocon

import (
	"encoding/binary"
	"fmt"
	"time"
)

// RTP版本
const RTPVersion = 2

// RTP头部结构
type RTPHeader struct {
	Version     uint8  // 2 bits
	Padding     bool   // 1 bit
	Extension   bool   // 1 bit
	CC          uint8  // 4 bits (CSRC count)
	Marker      bool   // 1 bit
	PayloadType uint8  // 7 bits
	SequenceNum uint16 // 16 bits
	Timestamp   uint32 // 32 bits
	SSRC        uint32 // 32 bits
	CSRC        []uint32 // 0-15 items
}

// RTP包
type RTPPacket struct {
	Header  RTPHeader
	Payload []byte
}

// RTP统计信息
type RTPStats struct {
	PacketsSent     uint32
	PacketsReceived uint32
	BytesSent       uint32
	BytesReceived   uint32
	LastSequence    uint16
	LastTimestamp   uint32
	SSRC            uint32
}

// 创建新的RTP头部
func NewRTPHeader() *RTPHeader {
	return &RTPHeader{
		Version:     RTPVersion,
		Padding:     false,
		Extension:   false,
		CC:          0,
		Marker:      false,
		PayloadType: 96, // H.264
		SequenceNum: 0,
		Timestamp:   0,
		SSRC:        0,
		CSRC:        make([]uint32, 0),
	}
}

// 序列化RTP头部为字节数组
func (h *RTPHeader) Marshal() []byte {
	// RTP头部至少12字节
	data := make([]byte, 12+int(h.CC)*4)
	
	// 第一个字节：版本(2) + 填充(1) + 扩展(1) + CC(4)
	data[0] = (h.Version << 6) | (boolToUint8(h.Padding) << 5) | (boolToUint8(h.Extension) << 4) | h.CC
	
	// 第二个字节：标记(1) + 载荷类型(7)
	data[1] = (boolToUint8(h.Marker) << 7) | h.PayloadType
	
	// 序列号 (2字节)
	binary.BigEndian.PutUint16(data[2:4], h.SequenceNum)
	
	// 时间戳 (4字节)
	binary.BigEndian.PutUint32(data[4:8], h.Timestamp)
	
	// SSRC (4字节)
	binary.BigEndian.PutUint32(data[8:12], h.SSRC)
	
	// CSRC列表
	for i, csrc := range h.CSRC {
		if i < int(h.CC) {
			binary.BigEndian.PutUint32(data[12+i*4:16+i*4], csrc)
		}
	}
	
	return data
}

// 从字节数组解析RTP头部
func ParseRTPHeader(data []byte) (*RTPHeader, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("RTP header too short")
	}
	
	h := &RTPHeader{}
	
	// 第一个字节
	h.Version = (data[0] >> 6) & 0x03
	h.Padding = (data[0] >> 5) & 0x01 == 1
	h.Extension = (data[0] >> 4) & 0x01 == 1
	h.CC = data[0] & 0x0F
	
	// 第二个字节
	h.Marker = (data[1] >> 7) & 0x01 == 1
	h.PayloadType = data[1] & 0x7F
	
	// 序列号
	h.SequenceNum = binary.BigEndian.Uint16(data[2:4])
	
	// 时间戳
	h.Timestamp = binary.BigEndian.Uint32(data[4:8])
	
	// SSRC
	h.SSRC = binary.BigEndian.Uint32(data[8:12])
	
	// CSRC列表
	h.CSRC = make([]uint32, h.CC)
	for i := uint8(0); i < h.CC; i++ {
		if len(data) >= int(12+i*4+4) {
			h.CSRC[i] = binary.BigEndian.Uint32(data[12+i*4 : 16+i*4])
		}
	}
	
	return h, nil
}

// 创建新的RTP包
func NewRTPPacket(payloadType uint8, sequenceNum uint16, timestamp uint32, ssrc uint32, payload []byte) *RTPPacket {
	header := NewRTPHeader()
	header.PayloadType = payloadType
	header.SequenceNum = sequenceNum
	header.Timestamp = timestamp
	header.SSRC = ssrc
	
	return &RTPPacket{
		Header:  *header,
		Payload: payload,
	}
}

// 序列化RTP包为字节数组
func (p *RTPPacket) Marshal() []byte {
	headerData := p.Header.Marshal()
	result := make([]byte, len(headerData)+len(p.Payload))
	copy(result, headerData)
	copy(result[len(headerData):], p.Payload)
	return result
}

// 从字节数组解析RTP包
func ParseRTPPacket(data []byte) (*RTPPacket, error) {
	header, err := ParseRTPHeader(data)
	if err != nil {
		return nil, err
	}
	
	headerSize := 12 + int(header.CC)*4
	if len(data) < headerSize {
		return nil, fmt.Errorf("RTP packet too short")
	}
	
	payload := make([]byte, len(data)-headerSize)
	copy(payload, data[headerSize:])
	
	return &RTPPacket{
		Header:  *header,
		Payload: payload,
	}, nil
}

// 创建新的RTP统计信息
func NewRTPStats(ssrc uint32) *RTPStats {
	return &RTPStats{
		SSRC: ssrc,
	}
}

// 更新发送统计
func (s *RTPStats) UpdateSent(sequenceNum uint16, timestamp uint32, bytes int) {
	s.PacketsSent++
	s.BytesSent += uint32(bytes)
	s.LastSequence = sequenceNum
	s.LastTimestamp = timestamp
}

// 更新接收统计
func (s *RTPStats) UpdateReceived(sequenceNum uint16, timestamp uint32, bytes int) {
	s.PacketsReceived++
	s.BytesReceived += uint32(bytes)
	s.LastSequence = sequenceNum
	s.LastTimestamp = timestamp
}

// 计算丢包率
func (s *RTPStats) GetLossRate() float64 {
	if s.PacketsReceived == 0 {
		return 0.0
	}
	
	expected := s.PacketsReceived + s.PacketsSent
	if expected == 0 {
		return 0.0
	}
	
	return float64(s.PacketsSent) / float64(expected)
}

// 生成SSRC
func GenerateSSRC() uint32 {
	return uint32(time.Now().UnixNano())
}

// 生成RTP时间戳（90kHz时钟）
func GenerateRTPTimestamp() uint32 {
	return uint32(time.Now().UnixNano() / 1000000 * 90) // 90kHz
}

// 检查RTP包是否有效
func ValidateRTPPacket(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	
	// 检查版本
	version := (data[0] >> 6) & 0x03
	if version != RTPVersion {
		return false
	}
	
	// 检查载荷类型
	payloadType := data[1] & 0x7F
	if payloadType > 127 {
		return false
	}
	
	return true
}

// 获取RTP包大小
func GetRTPPacketSize(header *RTPHeader, payloadSize int) int {
	return 12 + int(header.CC)*4 + payloadSize
}

// 辅助函数：bool转uint8
func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// RTP载荷类型常量
const (
	PayloadTypePCMU    = 0
	PayloadTypePCMA    = 8
	PayloadTypeG722    = 9
	PayloadTypeG729    = 18
	PayloadTypeH261    = 31
	PayloadTypeH263    = 34
	PayloadTypeH264    = 96
	PayloadTypeH265    = 97
	PayloadTypeVP8     = 98
	PayloadTypeVP9     = 99
	PayloadTypeAV1     = 100
)

// 获取载荷类型名称
func GetPayloadTypeName(payloadType uint8) string {
	names := map[uint8]string{
		PayloadTypePCMU: "PCMU",
		PayloadTypePCMA: "PCMA",
		PayloadTypeG722: "G722",
		PayloadTypeG729: "G729",
		PayloadTypeH261: "H261",
		PayloadTypeH263: "H263",
		PayloadTypeH264: "H264",
		PayloadTypeH265: "H265",
		PayloadTypeVP8:  "VP8",
		PayloadTypeVP9:  "VP9",
		PayloadTypeAV1:  "AV1",
	}
	
	if name, exists := names[payloadType]; exists {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", payloadType)
}
