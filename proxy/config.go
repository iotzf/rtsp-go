package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

// 代理配置文件结构
type ProxyConfigFile struct {
	ListenAddress   string            `json:"listen_address"`
	ListenPort      int               `json:"listen_port"`
	UpstreamServers map[string]string `json:"upstream_servers"`
	DefaultServer   string            `json:"default_server"`
	BufferSize      int               `json:"buffer_size"`
	TimeoutSeconds  int               `json:"timeout_seconds"`
	MaxConnections  int               `json:"max_connections"`
	EnableLogging   bool              `json:"enable_logging"`
	LogLevel        string            `json:"log_level"`
	StatsInterval   int               `json:"stats_interval"`
}

// 从配置文件加载配置
func LoadConfigFromFile(filename string) (*ProxyConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var configFile ProxyConfigFile
	if err := json.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// 转换为ProxyConfig
	config := NewProxyConfig()
	config.ListenAddress = configFile.ListenAddress
	config.ListenPort = configFile.ListenPort
	config.UpstreamServers = configFile.UpstreamServers
	config.DefaultServer = configFile.DefaultServer
	config.BufferSize = configFile.BufferSize
	config.Timeout = time.Duration(configFile.TimeoutSeconds) * time.Second
	config.MaxConnections = configFile.MaxConnections
	config.EnableLogging = configFile.EnableLogging

	return config, nil
}

// 保存配置到文件
func SaveConfigToFile(config *ProxyConfig, filename string) error {
	configFile := ProxyConfigFile{
		ListenAddress:   config.ListenAddress,
		ListenPort:      config.ListenPort,
		UpstreamServers: config.UpstreamServers,
		DefaultServer:   config.DefaultServer,
		BufferSize:      config.BufferSize,
		TimeoutSeconds:  int(config.Timeout.Seconds()),
		MaxConnections:  config.MaxConnections,
		EnableLogging:   config.EnableLogging,
		LogLevel:        "info",
		StatsInterval:   30,
	}

	data, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// 创建默认配置文件
func CreateDefaultConfigFile(filename string) error {
	config := NewProxyConfig()
	config.ListenAddress = "0.0.0.0"
	config.ListenPort = 8555
	config.UpstreamServers = map[string]string{
		"/test":    "rtsp://localhost:8554/test",
		"/camera1": "rtsp://localhost:8554/camera1",
		"/camera2": "rtsp://localhost:8554/camera2",
	}
	config.DefaultServer = "rtsp://localhost:8554/default"
	config.BufferSize = 4096
	config.Timeout = 30 * time.Second
	config.MaxConnections = 1000
	config.EnableLogging = true

	return SaveConfigToFile(config, filename)
}

// 验证配置
func ValidateConfig(config *ProxyConfig) error {
	if config.ListenAddress == "" {
		return fmt.Errorf("listen address cannot be empty")
	}

	if config.ListenPort <= 0 || config.ListenPort > 65535 {
		return fmt.Errorf("invalid listen port: %d", config.ListenPort)
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if config.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be positive")
	}

	if config.BufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive")
	}

	return nil
}

// 打印配置信息
func PrintConfig(config *ProxyConfig) {
	fmt.Println("RTSP Proxy Configuration")
	fmt.Println("========================")
	fmt.Printf("Listen Address: %s:%d\n", config.ListenAddress, config.ListenPort)
	fmt.Printf("Default Server: %s\n", config.DefaultServer)
	fmt.Printf("Timeout: %v\n", config.Timeout)
	fmt.Printf("Max Connections: %d\n", config.MaxConnections)
	fmt.Printf("Buffer Size: %d\n", config.BufferSize)
	fmt.Printf("Enable Logging: %v\n", config.EnableLogging)
	fmt.Println("\nUpstream Servers:")
	for path, server := range config.UpstreamServers {
		fmt.Printf("  %s -> %s\n", path, server)
	}
	fmt.Println()
}

// 示例配置文件内容
const ExampleConfigJSON = `{
  "listen_address": "0.0.0.0",
  "listen_port": 8555,
  "upstream_servers": {
    "/test": "rtsp://localhost:8554/test",
    "/camera1": "rtsp://localhost:8554/camera1",
    "/camera2": "rtsp://localhost:8554/camera2"
  },
  "default_server": "rtsp://localhost:8554/default",
  "buffer_size": 4096,
  "timeout_seconds": 30,
  "max_connections": 1000,
  "enable_logging": true,
  "log_level": "info",
  "stats_interval": 30
}`

// 创建示例配置文件
func CreateExampleConfigFile(filename string) error {
	return ioutil.WriteFile(filename, []byte(ExampleConfigJSON), 0644)
}
