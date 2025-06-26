package middlewares

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"xiaohuAdmin/global"

	"github.com/gin-gonic/gin"
)

// IPControl IP访问控制中间件
// 用于根据白名单/黑名单限制接口访问
func IPControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取IP控制配置（只获取一次，提升效率）
		enabled, whitelist, blacklist := global.GetIPControlConfig()

		// 检查是否启用IP控制
		if !enabled {
			c.Next()
			return
		}

		// 获取客户端IP
		clientIP := getClientIP(c)

		// 检查白名单（优先级高）
		if isIPInList(clientIP, whitelist) {
			c.Next()
			return
		}

		// 检查黑名单
		if isIPInList(clientIP, blacklist) {
			global.ZapLog.Warn("IP访问被拒绝（黑名单）",
				global.LogField("ip", clientIP),
				global.LogField("path", c.Request.URL.Path),
				global.LogField("method", c.Request.Method))

			c.JSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "访问被拒绝：IP地址在黑名单中",
				"data": nil,
			})
			c.Abort()
			return
		}

		// 严格白名单模式：如果IP不在白名单中，直接拒绝
		global.ZapLog.Warn("IP访问被拒绝（不在白名单中）",
			global.LogField("ip", clientIP),
			global.LogField("path", c.Request.URL.Path),
			global.LogField("method", c.Request.Method))

		c.JSON(http.StatusForbidden, gin.H{
			"code": 403,
			"msg":  "访问被拒绝：IP地址不在白名单中",
			"data": nil,
		})
		c.Abort()
	}
}

// getClientIP 获取客户端真实IP
// 按优先级从常见代理头或 RemoteAddr 获取
func getClientIP(c *gin.Context) string {
	// 按优先级获取IP
	ipHeaders := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Client-IP",
		"CF-Connecting-IP", // Cloudflare
		"True-Client-IP",   // Akamai
	}

	for _, header := range ipHeaders {
		if ip := c.GetHeader(header); ip != "" {
			// 处理多个IP的情况（如X-Forwarded-For: client, proxy1, proxy2）
			if strings.Contains(ip, ",") {
				ips := strings.Split(ip, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}
			return strings.TrimSpace(ip)
		}
	}

	// 如果没有代理头，使用RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// isIPInList 检查IP是否在指定列表中（支持CIDR）
func isIPInList(ip string, list []string) bool {
	for _, rule := range list {
		if isIPMatch(ip, rule) {
			return true
		}
	}
	return false
}

// isIPMatch 检查IP是否匹配（支持CIDR格式）
func isIPMatch(clientIP, ruleIP string) bool {
	// 如果规则包含斜杠，说明是CIDR格式
	if strings.Contains(ruleIP, "/") {
		_, ipNet, err := net.ParseCIDR(ruleIP)
		if err != nil {
			return false
		}

		ip := net.ParseIP(clientIP)
		if ip == nil {
			return false
		}

		return ipNet.Contains(ip)
	}

	// 直接IP比较
	return clientIP == ruleIP
}

// GetIPControlStatus 获取IP控制状态信息
func GetIPControlStatus() map[string]interface{} {
	enabled, whitelist, blacklist := global.GetIPControlConfig()
	return map[string]interface{}{
		"enabled":   enabled,
		"whitelist": whitelist,
		"blacklist": blacklist,
	}
}

// AddToWhitelist 添加IP到白名单
// 校验格式并避免重复
func AddToWhitelist(ip string) error {
	enabled, whitelist, blacklist := global.GetIPControlConfig()

	// 检查是否已存在
	for _, existingIP := range whitelist {
		if existingIP == ip {
			return fmt.Errorf("IP %s 已存在于白名单中", ip)
		}
	}

	// 验证IP格式
	if strings.Contains(ip, "/") {
		_, _, err := net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("无效的CIDR格式: %v", err)
		}
	} else {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("无效的IP地址: %s", ip)
		}
	}

	whitelist = append(whitelist, ip)
	global.UpdateIPControlConfig(enabled, whitelist, blacklist)

	return nil
}

// RemoveFromWhitelist 从白名单中移除IP
func RemoveFromWhitelist(ip string) error {
	enabled, whitelist, blacklist := global.GetIPControlConfig()

	for i, existingIP := range whitelist {
		if existingIP == ip {
			whitelist = append(whitelist[:i], whitelist[i+1:]...)
			global.UpdateIPControlConfig(enabled, whitelist, blacklist)
			return nil
		}
	}

	return fmt.Errorf("IP %s 不在白名单中", ip)
}

// AddToBlacklist 添加IP到黑名单
func AddToBlacklist(ip string) error {
	enabled, whitelist, blacklist := global.GetIPControlConfig()

	// 检查是否已存在
	for _, existingIP := range blacklist {
		if existingIP == ip {
			return fmt.Errorf("IP %s 已存在于黑名单中", ip)
		}
	}

	// 验证IP格式
	if strings.Contains(ip, "/") {
		_, _, err := net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("无效的CIDR格式: %v", err)
		}
	} else {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("无效的IP地址: %s", ip)
		}
	}

	blacklist = append(blacklist, ip)
	global.UpdateIPControlConfig(enabled, whitelist, blacklist)

	return nil
}

// RemoveFromBlacklist 从黑名单中移除IP
func RemoveFromBlacklist(ip string) error {
	enabled, whitelist, blacklist := global.GetIPControlConfig()

	for i, existingIP := range blacklist {
		if existingIP == ip {
			blacklist = append(blacklist[:i], blacklist[i+1:]...)
			global.UpdateIPControlConfig(enabled, whitelist, blacklist)
			return nil
		}
	}

	return fmt.Errorf("IP %s 不在黑名单中", ip)
}
