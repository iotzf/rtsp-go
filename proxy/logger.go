package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// 日志记录器
type Logger struct {
	Level        LogLevel
	File         *os.File
	AuditFile    *os.File
	Mutex        sync.RWMutex
	Stats        *LogStats
	EnableAudit  bool
	LogDir       string
}

// 日志统计
type LogStats struct {
	TotalLogs     uint64
	DebugLogs     uint64
	InfoLogs      uint64
	WarnLogs      uint64
	ErrorLogs     uint64
	FatalLogs     uint64
	AuditLogs     uint64
	StartTime     time.Time
	LastLogTime   time.Time
}

// 审计事件
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	SessionID   string                 `json:"session_id,omitempty"`
	ClientIP    string                 `json:"client_ip,omitempty"`
	Username    string                 `json:"username,omitempty"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource,omitempty"`
	Result      string                 `json:"result"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
}

// 创建新的日志记录器
func NewLogger(logDir string, level LogLevel, enableAudit bool) (*Logger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	logger := &Logger{
		Level:       level,
		EnableAudit: enableAudit,
		LogDir:      logDir,
		Stats: &LogStats{
			StartTime: time.Now(),
		},
	}

	// 打开主日志文件
	logFile := filepath.Join(logDir, "proxy.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	logger.File = file

	// 打开审计日志文件
	if enableAudit {
		auditFile := filepath.Join(logDir, "audit.log")
		audit, err := os.OpenFile(auditFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit file: %v", err)
		}
		logger.AuditFile = audit
	}

	return logger, nil
}

// 关闭日志记录器
func (l *Logger) Close() error {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	var err error
	if l.File != nil {
		err = l.File.Close()
	}
	if l.AuditFile != nil {
		if auditErr := l.AuditFile.Close(); auditErr != nil {
			if err == nil {
				err = auditErr
			}
		}
	}

	return err
}

// 记录日志
func (l *Logger) Log(level LogLevel, format string, args ...interface{}) {
	if level < l.Level {
		return
	}

	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	// 更新统计
	l.Stats.TotalLogs++
	l.Stats.LastLogTime = time.Now()

	switch level {
	case LogLevelDebug:
		l.Stats.DebugLogs++
	case LogLevelInfo:
		l.Stats.InfoLogs++
	case LogLevelWarn:
		l.Stats.WarnLogs++
	case LogLevelError:
		l.Stats.ErrorLogs++
	case LogLevelFatal:
		l.Stats.FatalLogs++
	}

	// 格式化日志消息
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := l.getLevelString(level)
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, levelStr, message)

	// 写入文件
	if l.File != nil {
		l.File.WriteString(logEntry)
		l.File.Sync()
	}

	// 同时输出到标准日志
	switch level {
	case LogLevelDebug:
		log.Printf("[DEBUG] %s", message)
	case LogLevelInfo:
		log.Printf("[INFO] %s", message)
	case LogLevelWarn:
		log.Printf("[WARN] %s", message)
	case LogLevelError:
		log.Printf("[ERROR] %s", message)
	case LogLevelFatal:
		log.Printf("[FATAL] %s", message)
	}
}

// 记录调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Log(LogLevelDebug, format, args...)
}

// 记录信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(LogLevelInfo, format, args...)
}

// 记录警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.Log(LogLevelWarn, format, args...)
}

// 记录错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(LogLevelError, format, args...)
}

// 记录致命错误日志
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.Log(LogLevelFatal, format, args...)
}

// 记录审计事件
func (l *Logger) AuditEvent(event *AuditEvent) {
	if !l.EnableAudit || l.AuditFile == nil {
		return
	}

	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	l.Stats.AuditLogs++

	// 序列化为JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		l.Error("Failed to marshal audit event: %v", err)
		return
	}

	// 写入审计日志
	auditEntry := fmt.Sprintf("%s\n", string(jsonData))
	l.AuditFile.WriteString(auditEntry)
	l.AuditFile.Sync()
}

// 记录连接事件
func (l *Logger) LogConnection(sessionID, clientIP, action string, duration time.Duration) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "connection",
		SessionID: sessionID,
		ClientIP:  clientIP,
		Action:    action,
		Result:    "success",
		Duration:  duration,
	}

	l.AuditEvent(event)
	l.Info("Connection %s: session=%s, client=%s, duration=%v", action, sessionID, clientIP, duration)
}

// 记录认证事件
func (l *Logger) LogAuthentication(sessionID, clientIP, username, action, result string) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "authentication",
		SessionID: sessionID,
		ClientIP:  clientIP,
		Username:  username,
		Action:    action,
		Result:    result,
	}

	l.AuditEvent(event)
	l.Info("Authentication %s: user=%s, client=%s, result=%s", action, username, clientIP, result)
}

// 记录RTSP请求事件
func (l *Logger) LogRTSPRequest(sessionID, clientIP, method, url, result string, duration time.Duration) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "rtsp_request",
		SessionID: sessionID,
		ClientIP:  clientIP,
		Action:    method,
		Resource:  url,
		Result:    result,
		Duration:  duration,
	}

	l.AuditEvent(event)
	l.Info("RTSP %s: session=%s, client=%s, url=%s, result=%s, duration=%v", 
		method, sessionID, clientIP, url, result, duration)
}

// 记录媒体流事件
func (l *Logger) LogMediaStream(sessionID, clientIP, streamType, action string, bytes uint64) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "media_stream",
		SessionID: sessionID,
		ClientIP:  clientIP,
		Action:    action,
		Resource:  streamType,
		Result:    "success",
		Details: map[string]interface{}{
			"bytes": bytes,
		},
	}

	l.AuditEvent(event)
	l.Info("Media stream %s: session=%s, client=%s, type=%s, bytes=%d", 
		action, sessionID, clientIP, streamType, bytes)
}

// 记录错误事件
func (l *Logger) LogError(sessionID, clientIP, errorType, message string) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "error",
		SessionID: sessionID,
		ClientIP:  clientIP,
		Action:    errorType,
		Result:    "error",
		Details: map[string]interface{}{
			"message": message,
		},
	}

	l.AuditEvent(event)
	l.Error("Error: session=%s, client=%s, type=%s, message=%s", sessionID, clientIP, errorType, message)
}

// 记录配置变更事件
func (l *Logger) LogConfigChange(action, resource string, details map[string]interface{}) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "config_change",
		Action:    action,
		Resource:  resource,
		Result:    "success",
		Details:   details,
	}

	l.AuditEvent(event)
	l.Info("Config change: action=%s, resource=%s", action, resource)
}

// 记录健康检查事件
func (l *Logger) LogHealthCheck(result string, details map[string]interface{}) {
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "health_check",
		Action:    "health_check",
		Result:    result,
		Details:   details,
	}

	l.AuditEvent(event)
	l.Info("Health check: result=%s", result)
}

// 获取日志级别字符串
func (l *Logger) getLevelString(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// 获取日志统计
func (l *Logger) GetStats() map[string]interface{} {
	l.Mutex.RLock()
	defer l.Mutex.RUnlock()

	return map[string]interface{}{
		"level":           l.getLevelString(l.Level),
		"enable_audit":    l.EnableAudit,
		"log_dir":         l.LogDir,
		"total_logs":      l.Stats.TotalLogs,
		"debug_logs":      l.Stats.DebugLogs,
		"info_logs":       l.Stats.InfoLogs,
		"warn_logs":       l.Stats.WarnLogs,
		"error_logs":      l.Stats.ErrorLogs,
		"fatal_logs":      l.Stats.FatalLogs,
		"audit_logs":      l.Stats.AuditLogs,
		"start_time":      l.Stats.StartTime.Format(time.RFC3339),
		"last_log_time":   l.Stats.LastLogTime.Format(time.RFC3339),
	}
}

// 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	l.Level = level
	l.Info("Log level changed to %s", l.getLevelString(level))
}

// 轮转日志文件
func (l *Logger) RotateLogs() error {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()

	// 关闭当前日志文件
	if l.File != nil {
		l.File.Close()
	}
	if l.AuditFile != nil {
		l.AuditFile.Close()
	}

	// 重命名现有日志文件
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	
	if l.File != nil {
		oldLogFile := filepath.Join(l.LogDir, "proxy.log")
		newLogFile := filepath.Join(l.LogDir, fmt.Sprintf("proxy-%s.log", timestamp))
		os.Rename(oldLogFile, newLogFile)
	}

	if l.AuditFile != nil {
		oldAuditFile := filepath.Join(l.LogDir, "audit.log")
		newAuditFile := filepath.Join(l.LogDir, fmt.Sprintf("audit-%s.log", timestamp))
		os.Rename(oldAuditFile, newAuditFile)
	}

	// 重新打开日志文件
	logFile := filepath.Join(l.LogDir, "proxy.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen log file: %v", err)
	}
	l.File = file

	if l.EnableAudit {
		auditFile := filepath.Join(l.LogDir, "audit.log")
		audit, err := os.OpenFile(auditFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to reopen audit file: %v", err)
		}
		l.AuditFile = audit
	}

	l.Info("Log files rotated successfully")
	return nil
}

// 清理旧日志文件
func (l *Logger) CleanupOldLogs(maxAge time.Duration) error {
	files, err := filepath.Glob(filepath.Join(l.LogDir, "*.log"))
	if err != nil {
		return fmt.Errorf("failed to list log files: %v", err)
	}

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				l.Error("Failed to remove old log file %s: %v", file, err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		l.Info("Cleaned up %d old log files", cleaned)
	}

	return nil
}
