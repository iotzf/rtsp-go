package rtspProtocon

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// 端口范围配置
type PortRange struct {
	Start int
	End   int
}

// 端口管理器
type PortManager struct {
	RTPRange    PortRange
	RTCPRange   PortRange
	UsedPorts   map[int]bool
	PortMutex   sync.RWMutex
	LastCleanup time.Time
}

// 端口分配信息
type PortAllocation struct {
	RTPPort    int
	RTCPPort   int
	AllocatedAt time.Time
	SessionID   string
}

// 创建新的端口管理器
func NewPortManager(rtpStart, rtpEnd, rtcpStart, rtcpEnd int) *PortManager {
	return &PortManager{
		RTPRange:    PortRange{Start: rtpStart, End: rtpEnd},
		RTCPRange:   PortRange{Start: rtcpStart, End: rtcpEnd},
		UsedPorts:   make(map[int]bool),
		LastCleanup: time.Now(),
	}
}

// 创建默认端口管理器
func NewDefaultPortManager() *PortManager {
	return NewPortManager(5000, 6000, 5000, 6000)
}

// 分配RTP端口对（带重试机制）
func (pm *PortManager) AllocatePortsWithRetry(sessionID string, maxRetries int) (*PortAllocation, error) {
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		allocation, err := pm.AllocatePorts(sessionID)
		if err == nil {
			return allocation, nil
		}
		
		lastErr = err
		
		// 如果不是第一次尝试，等待一段时间后重试
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			log.Printf("Port allocation attempt %d failed, retrying...", attempt+1)
		}
	}
	
	return nil, fmt.Errorf("failed to allocate ports after %d attempts: %v", maxRetries, lastErr)
}

// 检查端口冲突
func (pm *PortManager) CheckPortConflict(port int) bool {
	pm.PortMutex.RLock()
	defer pm.PortMutex.RUnlock()
	
	return pm.UsedPorts[port]
}

// 解决端口冲突
func (pm *PortManager) ResolvePortConflict(port int) error {
	pm.PortMutex.Lock()
	defer pm.PortMutex.Unlock()
	
	// 强制释放冲突的端口
	delete(pm.UsedPorts, port)
	log.Printf("Resolved port conflict for port %d", port)
	return nil
}

// 获取可用端口列表
func (pm *PortManager) GetAvailablePorts(count int) ([]int, error) {
	pm.PortMutex.RLock()
	defer pm.PortMutex.RUnlock()
	
	var availablePorts []int
	
	for port := pm.RTPRange.Start; port <= pm.RTPRange.End && len(availablePorts) < count; port += 2 {
		if pm.isPortAvailable(port) {
			availablePorts = append(availablePorts, port)
		}
	}
	
	if len(availablePorts) < count {
		return nil, fmt.Errorf("only %d ports available, requested %d", len(availablePorts), count)
	}
	
	return availablePorts, nil
}

// 预分配端口
func (pm *PortManager) PreAllocatePorts(count int) ([]*PortAllocation, error) {
	pm.PortMutex.Lock()
	defer pm.PortMutex.Unlock()
	
	var allocations []*PortAllocation
	
	for i := 0; i < count; i++ {
		allocation, err := pm.AllocatePorts(fmt.Sprintf("prealloc-%d", i))
		if err != nil {
			// 如果分配失败，释放已分配的端口
			for _, alloc := range allocations {
				pm.ReleasePorts(alloc)
			}
			return nil, fmt.Errorf("failed to pre-allocate port %d: %v", i, err)
		}
		allocations = append(allocations, allocation)
	}
	
	return allocations, nil
}

// 分配RTP端口对
func (pm *PortManager) AllocatePorts(sessionID string) (*PortAllocation, error) {
	pm.PortMutex.Lock()
	defer pm.PortMutex.Unlock()

	// 定期清理过期端口
	pm.cleanupExpiredPorts()

	// 尝试分配端口对
	for rtpPort := pm.RTPRange.Start; rtpPort <= pm.RTPRange.End; rtpPort += 2 {
		rtcpPort := rtpPort + 1
		
		// 检查端口是否可用
		if pm.isPortAvailable(rtpPort) && pm.isPortAvailable(rtcpPort) {
			// 标记端口为已使用
			pm.UsedPorts[rtpPort] = true
			pm.UsedPorts[rtcpPort] = true
			
			allocation := &PortAllocation{
				RTPPort:     rtpPort,
				RTCPPort:    rtcpPort,
				AllocatedAt: time.Now(),
				SessionID:   sessionID,
			}
			
			return allocation, nil
		}
	}
	
	return nil, fmt.Errorf("no available ports in range %d-%d", pm.RTPRange.Start, pm.RTPRange.End)
}

// 释放端口
func (pm *PortManager) ReleasePorts(allocation *PortAllocation) {
	pm.PortMutex.Lock()
	defer pm.PortMutex.Unlock()
	
	delete(pm.UsedPorts, allocation.RTPPort)
	delete(pm.UsedPorts, allocation.RTCPPort)
}

// 检查端口是否可用
func (pm *PortManager) isPortAvailable(port int) bool {
	// 检查是否已被分配
	if pm.UsedPorts[port] {
		return false
	}
	
	// 检查端口是否在范围内
	if port < pm.RTPRange.Start || port > pm.RTPRange.End {
		return false
	}
	
	// 尝试绑定端口检查是否被系统占用
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return false
	}
	
	conn.Close()
	return true
}

// 清理过期端口
func (pm *PortManager) cleanupExpiredPorts() {
	// 每5分钟清理一次
	if time.Since(pm.LastCleanup) < 5*time.Minute {
		return
	}
	
	pm.LastCleanup = time.Now()
	
	// 清理所有端口，让它们重新分配
	// 在实际应用中，这里应该基于会话超时来清理
	pm.UsedPorts = make(map[int]bool)
	log.Println("Cleaned up expired ports")
}

// 获取端口使用统计
func (pm *PortManager) GetStats() map[string]interface{} {
	pm.PortMux.RLock()
	defer pm.PortMux.RUnlock()
	
	usedCount := len(pm.UsedPorts)
	totalRTP := pm.RTPRange.End - pm.RTPRange.Start + 1
	
	return map[string]interface{}{
		"rtp_range_start":    pm.RTPRange.Start,
		"rtp_range_end":      pm.RTPRange.End,
		"rtcp_range_start":   pm.RTCPRange.Start,
		"rtcp_range_end":     pm.RTCPRange.End,
		"used_ports":         usedCount,
		"total_rtp_ports":    totalRTP,
		"available_ports":    totalRTP - usedCount,
		"last_cleanup":       pm.LastCleanup.Format(time.RFC3339),
	}
}

// 验证端口范围
func (pm *PortManager) ValidatePortRange() error {
	if pm.RTPRange.Start < 1024 || pm.RTPRange.Start > 65535 {
		return fmt.Errorf("RTP start port must be between 1024 and 65535")
	}
	
	if pm.RTPRange.End < pm.RTPRange.Start || pm.RTPRange.End > 65535 {
		return fmt.Errorf("RTP end port must be >= start port and <= 65535")
	}
	
	if pm.RTCPRange.Start < 1024 || pm.RTCPRange.Start > 65535 {
		return fmt.Errorf("RTCP start port must be between 1024 and 65535")
	}
	
	if pm.RTCPRange.End < pm.RTCPRange.Start || pm.RTCPRange.End > 65535 {
		return fmt.Errorf("RTCP end port must be >= start port and <= 65535")
	}
	
	return nil
}

// 设置端口范围
func (pm *PortManager) SetPortRange(rtpStart, rtpEnd, rtcpStart, rtcpEnd int) error {
	newRTPRange := PortRange{Start: rtpStart, End: rtpEnd}
	newRTCPRange := PortRange{Start: rtcpStart, End: rtcpEnd}
	
	// 临时设置范围进行验证
	tempPM := &PortManager{
		RTPRange:  newRTPRange,
		RTCPRange: newRTCPRange,
	}
	
	if err := tempPM.ValidatePortRange(); err != nil {
		return err
	}
	
	pm.PortMutex.Lock()
	defer pm.PortMutex.Unlock()
	
	pm.RTPRange = newRTPRange
	pm.RTCPRange = newRTCPRange
	
	return nil
}

// 检查端口是否在范围内
func (pm *PortManager) IsPortInRange(port int) bool {
	pm.PortMux.RLock()
	defer pm.PortMux.RUnlock()
	
	return (port >= pm.RTPRange.Start && port <= pm.RTPRange.End) ||
		   (port >= pm.RTCPRange.Start && port <= pm.RTCPRange.End)
}

// 获取下一个可用端口
func (pm *PortManager) GetNextAvailablePort(startPort int) (int, error) {
	pm.PortMutex.Lock()
	defer pm.PortMutex.Unlock()
	
	for port := startPort; port <= pm.RTPRange.End; port++ {
		if pm.isPortAvailable(port) {
			return port, nil
		}
	}
	
	return 0, fmt.Errorf("no available ports starting from %d", startPort)
}

// 批量检查端口可用性
func (pm *PortManager) CheckPortsAvailability(ports []int) map[int]bool {
	pm.PortMutex.RLock()
	defer pm.PortMutex.RUnlock()
	
	result := make(map[int]bool)
	for _, port := range ports {
		result[port] = pm.isPortAvailable(port)
	}
	
	return result
}

// 强制释放端口（用于清理）
func (pm *PortManager) ForceReleasePort(port int) {
	pm.PortMutex.Lock()
	defer pm.PortMutex.Unlock()
	
	delete(pm.UsedPorts, port)
}

// 获取端口使用情况
func (pm *PortManager) GetPortUsage() map[int]bool {
	pm.PortMutex.RLock()
	defer pm.PortMutex.RUnlock()
	
	// 返回副本
	result := make(map[int]bool)
	for port, used := range pm.UsedPorts {
		result[port] = used
	}
	
	return result
}

// 端口分配历史记录
type PortAllocationHistory struct {
	Allocations []*PortAllocation
	Mutex       sync.RWMutex
}

// 创建端口分配历史
func NewPortAllocationHistory() *PortAllocationHistory {
	return &PortAllocationHistory{
		Allocations: make([]*PortAllocation, 0),
	}
}

// 添加分配记录
func (pah *PortAllocationHistory) AddAllocation(allocation *PortAllocation) {
	pah.Mutex.Lock()
	defer pah.Mutex.Unlock()
	
	pah.Allocations = append(pah.Allocations, allocation)
}

// 获取分配历史
func (pah *PortAllocationHistory) GetHistory() []*PortAllocation {
	pah.Mutex.RLock()
	defer pah.Mutex.RUnlock()
	
	result := make([]*PortAllocation, len(pah.Allocations))
	copy(result, pah.Allocations)
	
	return result
}

// 清理历史记录
func (pah *PortAllocationHistory) CleanupHistory(maxAge time.Duration) {
	pah.Mutex.Lock()
	defer pah.Mutex.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	var validAllocations []*PortAllocation
	
	for _, allocation := range pah.Allocations {
		if allocation.AllocatedAt.After(cutoff) {
			validAllocations = append(validAllocations, allocation)
		}
	}
	
	pah.Allocations = validAllocations
}
