package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// 健康检查器
type HealthChecker struct {
	Proxy        *RTSPProxy
	CheckInterval time.Duration
	Timeout      time.Duration
	Running      bool
	Mutex        sync.RWMutex
	Stats        *HealthStats
}

// 健康检查统计
type HealthStats struct {
	TotalChecks       uint64
	SuccessfulChecks  uint64
	FailedChecks      uint64
	LastCheckTime     time.Time
	LastSuccessTime   time.Time
	LastFailureTime   time.Time
	ConsecutiveFailures uint64
}

// 上游服务器健康状态
type UpstreamHealth struct {
	URL             string
	LastCheck       time.Time
	IsHealthy       bool
	ResponseTime    time.Duration
	ConsecutiveFailures uint64
	LastError       error
}

// 创建新的健康检查器
func NewHealthChecker(proxy *RTSPProxy) *HealthChecker {
	return &HealthChecker{
		Proxy:         proxy,
		CheckInterval: 30 * time.Second,
		Timeout:       10 * time.Second,
		Running:       false,
		Stats: &HealthStats{
			LastCheckTime: time.Now(),
		},
	}
}

// 启动健康检查
func (hc *HealthChecker) Start() error {
	hc.Mutex.Lock()
	defer hc.Mutex.Unlock()

	if hc.Running {
		return fmt.Errorf("health checker already running")
	}

	hc.Running = true
	go hc.runHealthChecks()

	log.Println("Health checker started")
	return nil
}

// 停止健康检查
func (hc *HealthChecker) Stop() {
	hc.Mutex.Lock()
	defer hc.Mutex.Unlock()

	hc.Running = false
	log.Println("Health checker stopped")
}

// 运行健康检查
func (hc *HealthChecker) runHealthChecks() {
	ticker := time.NewTicker(hc.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.Mutex.RLock()
			running := hc.Running
			hc.Mutex.RUnlock()

			if !running {
				return
			}

			hc.performHealthCheck()
		}
	}
}

// 执行健康检查
func (hc *HealthChecker) performHealthCheck() {
	hc.Mutex.Lock()
	hc.Stats.TotalChecks++
	hc.Stats.LastCheckTime = time.Now()
	hc.Mutex.Unlock()

	// 检查代理服务器状态
	if !hc.checkProxyHealth() {
		hc.handleProxyUnhealthy()
		return
	}

	// 检查上游服务器健康状态
	hc.checkUpstreamHealth()

	// 检查会话状态
	hc.checkSessionHealth()

	hc.Mutex.Lock()
	hc.Stats.SuccessfulChecks++
	hc.Stats.LastSuccessTime = time.Now()
	hc.Stats.ConsecutiveFailures = 0
	hc.Mutex.Unlock()

	log.Println("Health check passed")
}

// 检查代理服务器健康状态
func (hc *HealthChecker) checkProxyHealth() bool {
	// 检查代理是否运行
	if !hc.Proxy.Running {
		log.Println("Health check failed: proxy not running")
		return false
	}

	// 检查监听器
	if hc.Proxy.Listener == nil {
		log.Println("Health check failed: listener is nil")
		return false
	}

	// 尝试连接到代理端口进行健康检查
	addr := hc.Proxy.Listener.Addr().String()
	conn, err := net.DialTimeout("tcp", addr, hc.Timeout)
	if err != nil {
		log.Printf("Health check failed: cannot connect to %s: %v", addr, err)
		return false
	}
	defer conn.Close()

	return true
}

// 检查上游服务器健康状态
func (hc *HealthChecker) checkUpstreamHealth() {
	for path, upstreamURL := range hc.Proxy.Config.UpstreamServers {
		health := hc.checkUpstreamServer(upstreamURL)
		if !health.IsHealthy {
			log.Printf("Upstream server %s (%s) is unhealthy: %v", path, upstreamURL, health.LastError)
		}
	}

	// 检查默认上游服务器
	if hc.Proxy.Config.DefaultServer != "" {
		health := hc.checkUpstreamServer(hc.Proxy.Config.DefaultServer)
		if !health.IsHealthy {
			log.Printf("Default upstream server %s is unhealthy: %v", hc.Proxy.Config.DefaultServer, health.LastError)
		}
	}
}

// 检查单个上游服务器
func (hc *HealthChecker) checkUpstreamServer(upstreamURL string) *UpstreamHealth {
	start := time.Now()
	health := &UpstreamHealth{
		URL:       upstreamURL,
		LastCheck: time.Now(),
	}

	// 解析URL并尝试连接
	conn, err := net.DialTimeout("tcp", extractHostPort(upstreamURL), hc.Timeout)
	if err != nil {
		health.IsHealthy = false
		health.LastError = err
		health.ConsecutiveFailures++
	} else {
		conn.Close()
		health.IsHealthy = true
		health.ResponseTime = time.Since(start)
		health.ConsecutiveFailures = 0
	}

	return health
}

// 检查会话健康状态
func (hc *HealthChecker) checkSessionHealth() {
	hc.Proxy.SessionsMux.RLock()
	sessionCount := len(hc.Proxy.Sessions)
	hc.Proxy.SessionsMux.RUnlock()

	log.Printf("Active sessions: %d", sessionCount)

	// 检查是否有长时间无活动的会话
	now := time.Now()
	hc.Proxy.SessionsMux.RLock()
	for _, session := range hc.Proxy.Sessions {
		if now.Sub(session.LastActivity) > 5*time.Minute {
			log.Printf("Session %s has been inactive for %v", session.ID, now.Sub(session.LastActivity))
		}
	}
	hc.Proxy.SessionsMux.RUnlock()
}

// 处理代理不健康的情况
func (hc *HealthChecker) handleProxyUnhealthy() {
	hc.Mutex.Lock()
	hc.Stats.FailedChecks++
	hc.Stats.LastFailureTime = time.Now()
	hc.Stats.ConsecutiveFailures++
	hc.Mutex.Unlock()

	log.Printf("Health check failed (consecutive failures: %d)", hc.Stats.ConsecutiveFailures)

	// 如果连续失败次数过多，尝试重启代理
	if hc.Stats.ConsecutiveFailures >= 3 {
		log.Println("Too many consecutive failures, attempting proxy restart...")
		hc.restartProxy()
	}
}

// 重启代理
func (hc *HealthChecker) restartProxy() {
	log.Println("Restarting proxy...")

	// 停止当前代理
	hc.Proxy.Stop()

	// 等待一段时间
	time.Sleep(5 * time.Second)

	// 重新启动代理
	if err := hc.Proxy.Start(); err != nil {
		log.Printf("Failed to restart proxy: %v", err)
	} else {
		log.Println("Proxy restarted successfully")
		
		// 重置失败计数
		hc.Mutex.Lock()
		hc.Stats.ConsecutiveFailures = 0
		hc.Mutex.Unlock()
	}
}

// 获取健康检查统计
func (hc *HealthChecker) GetStats() map[string]interface{} {
	hc.Mutex.RLock()
	defer hc.Mutex.RUnlock()

	successRate := float64(0)
	if hc.Stats.TotalChecks > 0 {
		successRate = float64(hc.Stats.SuccessfulChecks) / float64(hc.Stats.TotalChecks) * 100
	}

	return map[string]interface{}{
		"running":                hc.Running,
		"total_checks":          hc.Stats.TotalChecks,
		"successful_checks":     hc.Stats.SuccessfulChecks,
		"failed_checks":         hc.Stats.FailedChecks,
		"success_rate":          fmt.Sprintf("%.2f%%", successRate),
		"consecutive_failures":  hc.Stats.ConsecutiveFailures,
		"last_check_time":       hc.Stats.LastCheckTime.Format(time.RFC3339),
		"last_success_time":     hc.Stats.LastSuccessTime.Format(time.RFC3339),
		"last_failure_time":     hc.Stats.LastFailureTime.Format(time.RFC3339),
		"check_interval":        hc.CheckInterval.String(),
		"timeout":               hc.Timeout.String(),
	}
}

// 设置健康检查间隔
func (hc *HealthChecker) SetCheckInterval(interval time.Duration) {
	hc.Mutex.Lock()
	defer hc.Mutex.Unlock()

	hc.CheckInterval = interval
	log.Printf("Health check interval set to %v", interval)
}

// 设置健康检查超时
func (hc *HealthChecker) SetTimeout(timeout time.Duration) {
	hc.Mutex.Lock()
	defer hc.Mutex.Unlock()

	hc.Timeout = timeout
	log.Printf("Health check timeout set to %v", timeout)
}

// 从URL中提取主机和端口
func extractHostPort(url string) string {
	// 简单的URL解析，实际应用中应该使用url.Parse
	// 这里假设URL格式为 rtsp://host:port/path
	if len(url) > 7 && url[:7] == "rtsp://" {
		url = url[7:] // 移除 rtsp://
	}
	
	// 查找第一个 '/' 来分离主机端口和路径
	for i, char := range url {
		if char == '/' {
			return url[:i]
		}
	}
	
	return url
}

// HTTP健康检查端点
func (hc *HealthChecker) StartHTTPHealthEndpoint(port int) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		stats := hc.GetStats()
		
		// 如果代理不健康，返回503状态码
		if hc.Stats.ConsecutiveFailures > 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		
		// 返回JSON格式的健康状态
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"%s","stats":%+v}`, 
			map[bool]string{true: "healthy", false: "unhealthy"}[hc.Stats.ConsecutiveFailures == 0],
			stats)
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if hc.Proxy.Running {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "ready")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, "not ready")
		}
	})

	go func() {
		log.Printf("HTTP health endpoint started on port %d", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Printf("Failed to start HTTP health endpoint: %v", err)
		}
	}()
}
