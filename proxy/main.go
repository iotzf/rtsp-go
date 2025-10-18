package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 命令行参数
var (
	listenAddr     = flag.String("listen", "0.0.0.0:8555", "代理监听地址")
	upstreamServer = flag.String("upstream", "rtsp://localhost:8554/test", "默认上游服务器")
	timeout        = flag.Duration("timeout", 30*time.Second, "连接超时时间")
	maxConnections = flag.Int("max-conn", 1000, "最大连接数")
	enableLogging  = flag.Bool("log", true, "启用日志")
	configFile     = flag.String("config", "", "配置文件路径")
)

func main() {
	flag.Parse()

	// 创建代理配置
	config := NewProxyConfig()
	config.ListenAddress = "0.0.0.0"
	config.ListenPort = 8555
	config.Timeout = *timeout
	config.MaxConnections = *maxConnections
	config.EnableLogging = *enableLogging

	// 设置默认上游服务器
	if *upstreamServer != "" {
		config.DefaultServer = *upstreamServer
	}

	// 添加一些示例上游服务器
	config.UpstreamServers["/test"] = "rtsp://localhost:8554/test"
	config.UpstreamServers["/camera1"] = "rtsp://localhost:8554/camera1"
	config.UpstreamServers["/camera2"] = "rtsp://localhost:8554/camera2"

	// 创建RTSP代理
	proxy := NewRTSPProxy(config)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动代理
	go func() {
		log.Printf("Starting RTSP Proxy on %s", *listenAddr)
		log.Printf("Default upstream server: %s", config.DefaultServer)
		log.Printf("Upstream servers: %+v", config.UpstreamServers)

		if err := proxy.Start(); err != nil {
			log.Fatalf("Failed to start RTSP proxy: %v", err)
		}
	}()

	// 启动统计信息打印
	go printStats(proxy)

	// 等待信号
	<-sigChan
	log.Println("Received shutdown signal")

	// 停止代理
	proxy.Stop()
	log.Println("RTSP Proxy stopped")
}

// 打印统计信息
func printStats(proxy *RTSPProxy) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := proxy.GetStats()
			log.Printf("Proxy Stats: Active Sessions: %v, Total Connections: %v, Messages Forwarded: %v, Bytes Forwarded: %v",
				stats["active_sessions"],
				stats["total_connections"],
				stats["messages_forwarded"],
				stats["bytes_forwarded"])
		}
	}
}

// 初始化函数
func init() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[RTSP-Proxy] ")

	// 打印版本信息
	fmt.Println("RTSP Proxy Server")
	fmt.Println("=================")
	fmt.Println("Version: 1.0.0")
	fmt.Println("Author: RTSP-Go Team")
	fmt.Println()
}
