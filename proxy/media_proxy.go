package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// 媒体流代理
type MediaStreamProxy struct {
	Session         *ProxySession
	RTPProxy        *RTPStreamProxy
	RTCPProxy       *RTCPStreamProxy
	TCPProxy        *TCPStreamProxy
	UseTCPTransport bool
	Running         bool
	Mutex           sync.RWMutex
}

// RTP流代理
type RTPStreamProxy struct {
	ClientRTPConn   net.Conn
	UpstreamRTPConn net.Conn
	ClientRTCPConn  net.Conn
	UpstreamRTCPConn net.Conn
	Stats           *RTPProxyStats
	Running         bool
	Mutex           sync.RWMutex
}

// RTCP流代理
type RTCPStreamProxy struct {
	ClientConn    net.Conn
	UpstreamConn  net.Conn
	Stats         *RTCPProxyStats
	Running       bool
	Mutex         sync.RWMutex
}

// TCP流代理
type TCPStreamProxy struct {
	ClientConn    net.Conn
	UpstreamConn  net.Conn
	Stats         *TCPProxyStats
	Running       bool
	Mutex         sync.RWMutex
}

// RTP代理统计
type RTPProxyStats struct {
	RTPPacketsForwarded  uint64
	RTPBytesForwarded    uint64
	RTCPPacketsForwarded uint64
	RTCPBytesForwarded   uint64
	StartTime            time.Time
	LastActivity         time.Time
}

// RTCP代理统计
type RTCPProxyStats struct {
	PacketsForwarded uint64
	BytesForwarded  uint64
	StartTime        time.Time
	LastActivity     time.Time
}

// TCP代理统计
type TCPProxyStats struct {
	FramesForwarded uint64
	BytesForwarded  uint64
	StartTime       time.Time
	LastActivity    time.Time
}

// 创建新的媒体流代理
func NewMediaStreamProxy(session *ProxySession) *MediaStreamProxy {
	return &MediaStreamProxy{
		Session: session,
		RTPProxy: &RTPStreamProxy{
			Stats: &RTPProxyStats{
				StartTime:    time.Now(),
				LastActivity: time.Now(),
			},
		},
		RTCPProxy: &RTCPStreamProxy{
			Stats: &RTCPProxyStats{
				StartTime:    time.Now(),
				LastActivity: time.Now(),
			},
		},
		TCPProxy: &TCPStreamProxy{
			Stats: &TCPProxyStats{
				StartTime:    time.Now(),
				LastActivity: time.Now(),
			},
		},
		Running: false,
	}
}

// 启动媒体流代理
func (msp *MediaStreamProxy) Start() error {
	msp.Mutex.Lock()
	defer msp.Mutex.Unlock()

	if msp.Running {
		return fmt.Errorf("media stream proxy already running")
	}

	// 检查传输模式
	if msp.UseTCPTransport {
		// 启动TCP流代理
		go msp.startTCPStreamProxy()
	} else {
		// 启动UDP流代理
		go msp.startUDPStreamProxy()
	}

	msp.Running = true
	log.Printf("Media stream proxy started for session %s", msp.Session.ID)
	return nil
}

// 停止媒体流代理
func (msp *MediaStreamProxy) Stop() {
	msp.Mutex.Lock()
	defer msp.Mutex.Unlock()

	if !msp.Running {
		return
	}

	msp.Running = false

	// 停止RTP代理
	if msp.RTPProxy != nil {
		msp.RTPProxy.Stop()
	}

	// 停止RTCP代理
	if msp.RTCPProxy != nil {
		msp.RTCPProxy.Stop()
	}

	// 停止TCP代理
	if msp.TCPProxy != nil {
		msp.TCPProxy.Stop()
	}

	log.Printf("Media stream proxy stopped for session %s", msp.Session.ID)
}

// 启动UDP流代理
func (msp *MediaStreamProxy) startUDPStreamProxy() {
	log.Printf("Starting UDP stream proxy for session %s", msp.Session.ID)

	// 这里需要根据实际的RTP端口信息来建立UDP连接
	// 由于我们还没有完整的RTP端口信息，这里只是示例框架
	
	// 启动RTP流代理
	go msp.RTPProxy.startRTPProxy()
	
	// 启动RTCP流代理
	go msp.RTCPProxy.startRTCPProxy()
}

// 启动TCP流代理
func (msp *MediaStreamProxy) startTCPStreamProxy() {
	log.Printf("Starting TCP stream proxy for session %s", msp.Session.ID)
	
	// 启动TCP流代理
	go msp.TCPProxy.startTCPProxy()
}

// 启动RTP代理
func (rtp *RTPStreamProxy) startRTPProxy() {
	rtp.Mutex.Lock()
	rtp.Running = true
	rtp.Mutex.Unlock()

	log.Println("RTP stream proxy started")

	// 这里需要实现实际的RTP包转发逻辑
	// 由于我们还没有RTP连接信息，这里只是示例框架
	
	for rtp.Running {
		// 模拟RTP包转发
		time.Sleep(100 * time.Millisecond)
		
		rtp.Mutex.Lock()
		rtp.Stats.LastActivity = time.Now()
		rtp.Mutex.Unlock()
	}

	log.Println("RTP stream proxy stopped")
}

// 启动RTCP代理
func (rtcp *RTCPStreamProxy) startRTCPProxy() {
	rtcp.Mutex.Lock()
	rtcp.Running = true
	rtcp.Mutex.Unlock()

	log.Println("RTCP stream proxy started")

	// 这里需要实现实际的RTCP包转发逻辑
	
	for rtcp.Running {
		// 模拟RTCP包转发
		time.Sleep(1000 * time.Millisecond)
		
		rtcp.Mutex.Lock()
		rtcp.Stats.LastActivity = time.Now()
		rtcp.Mutex.Unlock()
	}

	log.Println("RTCP stream proxy stopped")
}

// 启动TCP代理
func (tcp *TCPStreamProxy) startTCPProxy() {
	tcp.Mutex.Lock()
	tcp.Running = true
	tcp.Mutex.Unlock()

	log.Println("TCP stream proxy started")

	// 这里需要实现实际的TCP Interleaved帧转发逻辑
	
	for tcp.Running {
		// 模拟TCP帧转发
		time.Sleep(100 * time.Millisecond)
		
		tcp.Mutex.Lock()
		tcp.Stats.LastActivity = time.Now()
		tcp.Mutex.Unlock()
	}

	log.Println("TCP stream proxy stopped")
}

// 停止RTP代理
func (rtp *RTPStreamProxy) Stop() {
	rtp.Mutex.Lock()
	defer rtp.Mutex.Unlock()

	rtp.Running = false

	if rtp.ClientRTPConn != nil {
		rtp.ClientRTPConn.Close()
	}
	if rtp.UpstreamRTPConn != nil {
		rtp.UpstreamRTPConn.Close()
	}
	if rtp.ClientRTCPConn != nil {
		rtp.ClientRTCPConn.Close()
	}
	if rtp.UpstreamRTCPConn != nil {
		rtp.UpstreamRTCPConn.Close()
	}
}

// 停止RTCP代理
func (rtcp *RTCPStreamProxy) Stop() {
	rtcp.Mutex.Lock()
	defer rtcp.Mutex.Unlock()

	rtcp.Running = false

	if rtcp.ClientConn != nil {
		rtcp.ClientConn.Close()
	}
	if rtcp.UpstreamConn != nil {
		rtcp.UpstreamConn.Close()
	}
}

// 停止TCP代理
func (tcp *TCPStreamProxy) Stop() {
	tcp.Mutex.Lock()
	defer tcp.Mutex.Unlock()

	tcp.Running = false

	if tcp.ClientConn != nil {
		tcp.ClientConn.Close()
	}
	if tcp.UpstreamConn != nil {
		tcp.UpstreamConn.Close()
	}
}

// 转发RTP包
func (rtp *RTPStreamProxy) ForwardRTPPacket(packet *rtspProtocon.RTPPacket, direction string) error {
	rtp.Mutex.Lock()
	defer rtp.Mutex.Unlock()

	if !rtp.Running {
		return fmt.Errorf("RTP proxy not running")
	}

	// 这里需要实现实际的RTP包转发逻辑
	// 根据direction参数决定转发方向
	
	rtp.Stats.RTPPacketsForwarded++
	rtp.Stats.RTPBytesForwarded += uint64(len(packet.Payload))
	rtp.Stats.LastActivity = time.Now()

	return nil
}

// 转发RTCP包
func (rtcp *RTCPStreamProxy) ForwardRTCPPacket(data []byte, direction string) error {
	rtcp.Mutex.Lock()
	defer rtcp.Mutex.Unlock()

	if !rtcp.Running {
		return fmt.Errorf("RTCP proxy not running")
	}

	// 这里需要实现实际的RTCP包转发逻辑
	
	rtcp.Stats.PacketsForwarded++
	rtcp.Stats.BytesForwarded += uint64(len(data))
	rtcp.Stats.LastActivity = time.Now()

	return nil
}

// 转发TCP Interleaved帧
func (tcp *TCPStreamProxy) ForwardTCPFrame(frame *rtspProtocon.TCPInterleavedFrame, direction string) error {
	tcp.Mutex.Lock()
	defer tcp.Mutex.Unlock()

	if !tcp.Running {
		return fmt.Errorf("TCP proxy not running")
	}

	// 这里需要实现实际的TCP帧转发逻辑
	
	tcp.Stats.FramesForwarded++
	tcp.Stats.BytesForwarded += uint64(len(frame.Data))
	tcp.Stats.LastActivity = time.Now()

	return nil
}

// 获取RTP代理统计
func (rtp *RTPStreamProxy) GetStats() map[string]interface{} {
	rtp.Mutex.RLock()
	defer rtp.Mutex.RUnlock()

	return map[string]interface{}{
		"rtp_packets_forwarded":  rtp.Stats.RTPPacketsForwarded,
		"rtp_bytes_forwarded":    rtp.Stats.RTPBytesForwarded,
		"rtcp_packets_forwarded": rtp.Stats.RTCPPacketsForwarded,
		"rtcp_bytes_forwarded":   rtp.Stats.RTCPBytesForwarded,
		"start_time":             rtp.Stats.StartTime.Format(time.RFC3339),
		"last_activity":          rtp.Stats.LastActivity.Format(time.RFC3339),
		"running":                rtp.Running,
	}
}

// 获取RTCP代理统计
func (rtcp *RTCPStreamProxy) GetStats() map[string]interface{} {
	rtcp.Mutex.RLock()
	defer rtcp.Mutex.RUnlock()

	return map[string]interface{}{
		"packets_forwarded": rtcp.Stats.PacketsForwarded,
		"bytes_forwarded":   rtcp.Stats.BytesForwarded,
		"start_time":        rtcp.Stats.StartTime.Format(time.RFC3339),
		"last_activity":     rtcp.Stats.LastActivity.Format(time.RFC3339),
		"running":           rtcp.Running,
	}
}

// 获取TCP代理统计
func (tcp *TCPStreamProxy) GetStats() map[string]interface{} {
	tcp.Mutex.RLock()
	defer tcp.Mutex.RUnlock()

	return map[string]interface{}{
		"frames_forwarded": tcp.Stats.FramesForwarded,
		"bytes_forwarded":  tcp.Stats.BytesForwarded,
		"start_time":        tcp.Stats.StartTime.Format(time.RFC3339),
		"last_activity":     tcp.Stats.LastActivity.Format(time.RFC3339),
		"running":           tcp.Running,
	}
}

// 获取媒体流代理统计
func (msp *MediaStreamProxy) GetStats() map[string]interface{} {
	msp.Mutex.RLock()
	defer msp.Mutex.RUnlock()

	stats := map[string]interface{}{
		"session_id":        msp.Session.ID,
		"use_tcp_transport": msp.UseTCPTransport,
		"running":           msp.Running,
	}

	if msp.RTPProxy != nil {
		stats["rtp_proxy"] = msp.RTPProxy.GetStats()
	}

	if msp.RTCPProxy != nil {
		stats["rtcp_proxy"] = msp.RTCPProxy.GetStats()
	}

	if msp.TCPProxy != nil {
		stats["tcp_proxy"] = msp.TCPProxy.GetStats()
	}

	return stats
}

// 设置传输模式
func (msp *MediaStreamProxy) SetTransportMode(useTCP bool) {
	msp.Mutex.Lock()
	defer msp.Mutex.Unlock()

	msp.UseTCPTransport = useTCP
}

// 双向数据转发
func (msp *MediaStreamProxy) ForwardData(src, dst net.Conn, proxyType string) {
	buffer := make([]byte, 4096)
	
	for {
		n, err := src.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from %s: %v", proxyType, err)
			}
			break
		}

		_, err = dst.Write(buffer[:n])
		if err != nil {
			log.Printf("Error writing to %s: %v", proxyType, err)
			break
		}

		// 更新统计
		msp.updateProxyStats(proxyType, n)
	}
}

// 更新代理统计
func (msp *MediaStreamProxy) updateProxyStats(proxyType string, bytes int) {
	switch proxyType {
	case "rtp":
		if msp.RTPProxy != nil {
			msp.RTPProxy.Mutex.Lock()
			msp.RTPProxy.Stats.RTPBytesForwarded += uint64(bytes)
			msp.RTPProxy.Stats.LastActivity = time.Now()
			msp.RTPProxy.Mutex.Unlock()
		}
	case "rtcp":
		if msp.RTCPProxy != nil {
			msp.RTCPProxy.Mutex.Lock()
			msp.RTCPProxy.Stats.BytesForwarded += uint64(bytes)
			msp.RTCPProxy.Stats.LastActivity = time.Now()
			msp.RTCPProxy.Mutex.Unlock()
		}
	case "tcp":
		if msp.TCPProxy != nil {
			msp.TCPProxy.Mutex.Lock()
			msp.TCPProxy.Stats.BytesForwarded += uint64(bytes)
			msp.TCPProxy.Stats.LastActivity = time.Now()
			msp.TCPProxy.Mutex.Unlock()
		}
	}
}
