package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// RTSP消息读取器
type RTSPReader struct {
	conn   net.Conn
	reader *bufio.Reader
}

// RTSP消息写入器
type RTSPWriter struct {
	conn   net.Conn
	writer *bufio.Writer
}

// 创建新的RTSP读取器
func NewRTSPReader(conn net.Conn) *RTSPReader {
	return &RTSPReader{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

// 创建新的RTSP写入器
func NewRTSPWriter(conn net.Conn) *RTSPWriter {
	return &RTSPWriter{
		conn:   conn,
		writer: bufio.NewWriter(conn),
	}
}

// 读取RTSP消息
func (r *RTSPReader) ReadMessage() (*rtspProtocon.Message, error) {
	var lines []string
	var body string

	// 读取头部
	for {
		line, err := r.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read line: %v", err)
		}
		
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lines = append(lines, line)
	}

	// 检查是否有消息体
	contentLength := r.getHeaderValue(lines, "Content-Length")
	if contentLength != "" {
		if length, err := strconv.Atoi(contentLength); err == nil && length > 0 {
			bodyBytes := make([]byte, length)
			if _, err := io.ReadFull(r.reader, bodyBytes); err != nil {
				return nil, fmt.Errorf("failed to read message body: %v", err)
			}
			body = string(bodyBytes)
		}
	}

	// 构建消息字符串
	messageStr := strings.Join(lines, "\r\n") + "\r\n\r\n" + body

	// 解析消息
	message, err := rtspProtocon.ParseMessage(messageStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RTSP message: %v", err)
	}

	return message, nil
}

// 写入RTSP消息
func (w *RTSPWriter) WriteMessage(message *rtspProtocon.Message) error {
	messageStr := message.String()
	if _, err := w.writer.WriteString(messageStr); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}
	
	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %v", err)
	}
	
	return nil
}

// 从头部行中获取指定头的值
func (r *RTSPReader) getHeaderValue(lines []string, headerName string) string {
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), strings.ToLower(headerName)+":") {
			return strings.TrimSpace(line[len(headerName)+1:])
		}
	}
	return ""
}

// 设置连接超时
func (r *RTSPReader) SetTimeout(timeout time.Duration) {
	if tcpConn, ok := r.conn.(*net.TCPConn); ok {
		tcpConn.SetReadDeadline(time.Now().Add(timeout))
	}
}

// 设置连接超时
func (w *RTSPWriter) SetTimeout(timeout time.Duration) {
	if tcpConn, ok := w.conn.(*net.TCPConn); ok {
		tcpConn.SetWriteDeadline(time.Now().Add(timeout))
	}
}

// 关闭连接
func (r *RTSPReader) Close() error {
	return r.conn.Close()
}

// 关闭连接
func (w *RTSPWriter) Close() error {
	return w.conn.Close()
}

// 获取远程地址
func (r *RTSPReader) RemoteAddr() net.Addr {
	return r.conn.RemoteAddr()
}

// 获取远程地址
func (w *RTSPWriter) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}
