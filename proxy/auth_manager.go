package main

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	rtspProtocon "github.com/iotzf/rtsp-go/rtspProtocon"
)

// 认证管理器
type AuthManager struct {
	Users         map[string]*User
	ACLRules      []*ACLRule
	AuthRequired  bool
	DefaultAllow  bool
	Mutex         sync.RWMutex
	Stats         *AuthStats
}

// 用户信息
type User struct {
	Username    string
	Password    string
	Permissions []string
	Created     time.Time
	LastLogin   time.Time
	Active      bool
}

// ACL规则
type ACLRule struct {
	ID          string
	Name        string
	Path        string
	Method      string
	SourceIP    string
	UserGroup   string
	Action      string // allow, deny
	Priority    int
	Enabled     bool
	Created     time.Time
	LastModified time.Time
}

// 认证统计
type AuthStats struct {
	TotalRequests    uint64
	AllowedRequests  uint64
	DeniedRequests   uint64
	AuthFailures     uint64
	LastAuthTime     time.Time
	ActiveUsers      uint64
}

// 创建新的认证管理器
func NewAuthManager() *AuthManager {
	return &AuthManager{
		Users:        make(map[string]*User),
		ACLRules:     make([]*ACLRule, 0),
		AuthRequired: false,
		DefaultAllow: true,
		Stats: &AuthStats{
			LastAuthTime: time.Now(),
		},
	}
}

// 添加用户
func (am *AuthManager) AddUser(username, password string, permissions []string) error {
	am.Mutex.Lock()
	defer am.Mutex.Unlock()

	if _, exists := am.Users[username]; exists {
		return fmt.Errorf("user %s already exists", username)
	}

	user := &User{
		Username:    username,
		Password:    password,
		Permissions: permissions,
		Created:     time.Now(),
		Active:      true,
	}

	am.Users[username] = user
	am.Stats.ActiveUsers++

	log.Printf("User %s added successfully", username)
	return nil
}

// 删除用户
func (am *AuthManager) RemoveUser(username string) error {
	am.Mutex.Lock()
	defer am.Mutex.Unlock()

	if _, exists := am.Users[username]; !exists {
		return fmt.Errorf("user %s not found", username)
	}

	delete(am.Users, username)
	am.Stats.ActiveUsers--

	log.Printf("User %s removed successfully", username)
	return nil
}

// 验证用户凭据
func (am *AuthManager) AuthenticateUser(username, password string) (*User, error) {
	am.Mutex.RLock()
	user, exists := am.Users[username]
	am.Mutex.RUnlock()

	if !exists {
		am.Mutex.Lock()
		am.Stats.AuthFailures++
		am.Mutex.Unlock()
		return nil, fmt.Errorf("user %s not found", username)
	}

	if !user.Active {
		am.Mutex.Lock()
		am.Stats.AuthFailures++
		am.Mutex.Unlock()
		return nil, fmt.Errorf("user %s is inactive", username)
	}

	if user.Password != password {
		am.Mutex.Lock()
		am.Stats.AuthFailures++
		am.Mutex.Unlock()
		return nil, fmt.Errorf("invalid password for user %s", username)
	}

	// 更新最后登录时间
	am.Mutex.Lock()
	user.LastLogin = time.Now()
	am.Mutex.Unlock()

	am.Mutex.Lock()
	am.Stats.LastAuthTime = time.Now()
	am.Mutex.Unlock()

	log.Printf("User %s authenticated successfully", username)
	return user, nil
}

// 添加ACL规则
func (am *AuthManager) AddACLRule(rule *ACLRule) error {
	am.Mutex.Lock()
	defer am.Mutex.Unlock()

	rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	rule.Created = time.Now()
	rule.LastModified = time.Now()

	am.ACLRules = append(am.ACLRules, rule)

	log.Printf("ACL rule %s added successfully", rule.ID)
	return nil
}

// 删除ACL规则
func (am *AuthManager) RemoveACLRule(ruleID string) error {
	am.Mutex.Lock()
	defer am.Mutex.Unlock()

	for i, rule := range am.ACLRules {
		if rule.ID == ruleID {
			am.ACLRules = append(am.ACLRules[:i], am.ACLRules[i+1:]...)
			log.Printf("ACL rule %s removed successfully", ruleID)
			return nil
		}
	}

	return fmt.Errorf("ACL rule %s not found", ruleID)
}

// 检查访问权限
func (am *AuthManager) CheckAccess(clientIP, username, path, method string) bool {
	am.Mutex.Lock()
	am.Stats.TotalRequests++
	am.Mutex.Unlock()

	// 如果不需要认证且默认允许，直接通过
	if !am.AuthRequired && am.DefaultAllow {
		am.Mutex.Lock()
		am.Stats.AllowedRequests++
		am.Mutex.Unlock()
		return true
	}

	// 如果需要认证但没有用户名，拒绝
	if am.AuthRequired && username == "" {
		am.Mutex.Lock()
		am.Stats.DeniedRequests++
		am.Mutex.Unlock()
		return false
	}

	// 检查ACL规则
	am.Mutex.RLock()
	defer am.Mutex.RUnlock()

	// 按优先级排序规则
	rules := make([]*ACLRule, len(am.ACLRules))
	copy(rules, am.ACLRules)

	// 简单的排序（实际应用中应该使用更高效的排序算法）
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	// 检查每个规则
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if am.matchACLRule(rule, clientIP, username, path, method) {
			allowed := rule.Action == "allow"
			
			am.Mutex.Lock()
			if allowed {
				am.Stats.AllowedRequests++
			} else {
				am.Stats.DeniedRequests++
			}
			am.Mutex.Unlock()

			return allowed
		}
	}

	// 如果没有匹配的规则，使用默认策略
	allowed := am.DefaultAllow
	
	am.Mutex.Lock()
	if allowed {
		am.Stats.AllowedRequests++
	} else {
		am.Stats.DeniedRequests++
	}
	am.Mutex.Unlock()

	return allowed
}

// 匹配ACL规则
func (am *AuthManager) matchACLRule(rule *ACLRule, clientIP, username, path, method string) bool {
	// 检查路径匹配
	if rule.Path != "" && !strings.HasPrefix(path, rule.Path) {
		return false
	}

	// 检查方法匹配
	if rule.Method != "" && rule.Method != method {
		return false
	}

	// 检查IP匹配
	if rule.SourceIP != "" && !am.matchIP(rule.SourceIP, clientIP) {
		return false
	}

	// 检查用户组匹配
	if rule.UserGroup != "" {
		user, exists := am.Users[username]
		if !exists {
			return false
		}

		if !am.userInGroup(user, rule.UserGroup) {
			return false
		}
	}

	return true
}

// 匹配IP地址
func (am *AuthManager) matchIP(ruleIP, clientIP string) bool {
	// 简单的IP匹配，实际应用中应该支持CIDR等格式
	if ruleIP == "*" || ruleIP == clientIP {
		return true
	}

	// 支持通配符匹配
	if strings.Contains(ruleIP, "*") {
		ruleParts := strings.Split(ruleIP, ".")
		clientParts := strings.Split(clientIP, ".")

		if len(ruleParts) != len(clientParts) {
			return false
		}

		for i, rulePart := range ruleParts {
			if rulePart != "*" && rulePart != clientParts[i] {
				return false
			}
		}

		return true
	}

	return false
}

// 检查用户是否在组中
func (am *AuthManager) userInGroup(user *User, group string) bool {
	for _, permission := range user.Permissions {
		if permission == group {
			return true
		}
	}
	return false
}

// 解析RTSP认证头
func (am *AuthManager) ParseAuthHeader(authHeader string) (username, password string, err error) {
	if authHeader == "" {
		return "", "", fmt.Errorf("no authorization header")
	}

	// 支持Basic认证
	if strings.HasPrefix(authHeader, "Basic ") {
		encoded := authHeader[6:] // 移除 "Basic " 前缀
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", "", fmt.Errorf("failed to decode basic auth: %v", err)
		}

		credentials := string(decoded)
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid basic auth format")
		}

		return parts[0], parts[1], nil
	}

	// 支持Digest认证（简化版本）
	if strings.HasPrefix(authHeader, "Digest ") {
		// 这里应该实现完整的Digest认证
		// 为了简化，这里只是示例
		return "", "", fmt.Errorf("digest authentication not implemented")
	}

	return "", "", fmt.Errorf("unsupported authentication method")
}

// 生成认证挑战
func (am *AuthManager) GenerateAuthChallenge(realm string) string {
	// Basic认证挑战
	return fmt.Sprintf(`Basic realm="%s"`, realm)
}

// 获取认证统计
func (am *AuthManager) GetStats() map[string]interface{} {
	am.Mutex.RLock()
	defer am.Mutex.RUnlock()

	denyRate := float64(0)
	if am.Stats.TotalRequests > 0 {
		denyRate = float64(am.Stats.DeniedRequests) / float64(am.Stats.TotalRequests) * 100
	}

	return map[string]interface{}{
		"auth_required":        am.AuthRequired,
		"default_allow":        am.DefaultAllow,
		"total_requests":       am.Stats.TotalRequests,
		"allowed_requests":     am.Stats.AllowedRequests,
		"denied_requests":      am.Stats.DeniedRequests,
		"auth_failures":        am.Stats.AuthFailures,
		"deny_rate":           fmt.Sprintf("%.2f%%", denyRate),
		"active_users":         am.Stats.ActiveUsers,
		"acl_rules_count":      len(am.ACLRules),
		"last_auth_time":       am.Stats.LastAuthTime.Format(time.RFC3339),
	}
}

// 设置认证要求
func (am *AuthManager) SetAuthRequired(required bool) {
	am.Mutex.Lock()
	defer am.Mutex.Unlock()

	am.AuthRequired = required
	log.Printf("Authentication required set to %v", required)
}

// 设置默认允许策略
func (am *AuthManager) SetDefaultAllow(allow bool) {
	am.Mutex.Lock()
	defer am.Mutex.Unlock()

	am.DefaultAllow = allow
	log.Printf("Default allow policy set to %v", allow)
}

// 创建默认ACL规则
func (am *AuthManager) CreateDefaultACLRules() {
	// 允许本地访问
	localRule := &ACLRule{
		Name:        "Allow Local Access",
		Path:        "",
		Method:      "",
		SourceIP:    "127.0.0.1",
		UserGroup:   "",
		Action:      "allow",
		Priority:    100,
		Enabled:     true,
	}

	// 拒绝所有其他访问
	denyRule := &ACLRule{
		Name:        "Deny All Others",
		Path:        "",
		Method:      "",
		SourceIP:    "",
		UserGroup:   "",
		Action:      "deny",
		Priority:    1,
		Enabled:     true,
	}

	am.AddACLRule(localRule)
	am.AddACLRule(denyRule)

	log.Println("Default ACL rules created")
}

// 创建示例用户
func (am *AuthManager) CreateSampleUsers() {
	// 创建管理员用户
	am.AddUser("admin", "admin123", []string{"admin", "user"})
	
	// 创建普通用户
	am.AddUser("user", "user123", []string{"user"})
	
	// 创建只读用户
	am.AddUser("viewer", "viewer123", []string{"viewer"})

	log.Println("Sample users created")
}
