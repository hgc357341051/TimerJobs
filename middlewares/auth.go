package middlewares

import (
	"strings"
	"time"
	funcs "xiaohuAdmin/function"
	"xiaohuAdmin/global"
	"xiaohuAdmin/models/admins"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT声明结构体
// 用于存储用户身份信息和标准JWT字段
// swagger:model JWTClaims
type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTConfig JWT配置结构体
// 用于配置JWT密钥和过期时间
// swagger:model JWTConfig
type JWTConfig struct {
	SecretKey string
	Expire    time.Duration
}

var jwtConfig = &JWTConfig{
	SecretKey: "xiaohu_job_system_secret_key_2024",
	Expire:    24 * time.Hour, // 24小时过期
}

// GenerateToken 生成JWT令牌
func GenerateToken(user *admins.Admin) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtConfig.Expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "xiaohu_job_system",
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.SecretKey))
}

// ParseToken 解析JWT令牌
func ParseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtConfig.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// AuthMiddleware JWT认证中间件，强制要求认证
// 用于保护需要登录的接口
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			funcs.Unauthorized(c, "缺少认证令牌")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			funcs.Unauthorized(c, "无效的认证格式")
			c.Abort()
			return
		}

		// 提取令牌
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 解析令牌
		claims, err := ParseToken(tokenString)
		if err != nil {
			global.ZapLog.Error("JWT令牌解析失败", global.LogError(err))
			funcs.Unauthorized(c, "无效的认证令牌")
			c.Abort()
			return
		}

		// 检查用户是否存在且状态正常
		var user admins.Admin
		if err := global.DB.First(&user, claims.UserID).Error; err != nil {
			global.ZapLog.Error("用户不存在", global.LogError(err))
			funcs.Unauthorized(c, "用户不存在")
			c.Abort()
			return
		}

		if user.Status != 1 {
			funcs.Unauthorized(c, "用户已被禁用")
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("user", &user)

		// 记录访问日志
		global.ZapLog.Info("用户访问",
			global.LogField("user_id", claims.UserID),
			global.LogField("username", claims.Username),
			global.LogField("role", claims.Role),
			global.LogField("ip", c.ClientIP()),
			global.LogField("path", c.Request.URL.Path),
			global.LogField("method", c.Request.Method),
		)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选JWT认证中间件，不强制要求认证
// 用于部分接口可选登录
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := ParseToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		var user admins.Admin
		if err := global.DB.First(&user, claims.UserID).Error; err != nil {
			c.Next()
			return
		}

		if user.Status != 1 {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("user", &user)

		c.Next()
	}
}

// RoleMiddleware 角色权限中间件
// 用于限制接口访问角色
func RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			funcs.Forbidden(c, "需要认证")
			c.Abort()
			return
		}

		role := userRole.(string)
		hasRole := false
		for _, r := range roles {
			if r == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			global.ZapLog.Warn("权限不足",
				global.LogField("user_role", role),
				global.LogField("required_roles", roles),
				global.LogField("ip", c.ClientIP()),
				global.LogField("path", c.Request.URL.Path),
			)
			funcs.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminOnlyMiddleware 仅管理员中间件
func AdminOnlyMiddleware() gin.HandlerFunc {
	return RoleMiddleware("admin")
}

// GetCurrentUser 获取当前用户
func GetCurrentUser(c *gin.Context) *admins.Admin {
	user, exists := c.Get("user")
	if !exists {
		return nil
	}
	return user.(*admins.Admin)
}

// GetCurrentUserField 获取当前用户上下文字段
func GetCurrentUserField[T any](c *gin.Context, key string) (T, bool) {
	val, exists := c.Get(key)
	if !exists {
		var zero T
		return zero, false
	}
	casted, ok := val.(T)
	return casted, ok
}

// GetCurrentUserID 获取当前用户ID
func GetCurrentUserID(c *gin.Context) uint {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return userID.(uint)
}

// GetCurrentUsername 获取当前用户名
func GetCurrentUsername(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}
	return username.(string)
}

// GetCurrentRole 获取当前用户角色
func GetCurrentRole(c *gin.Context) string {
	role, exists := c.Get("role")
	if !exists {
		return ""
	}
	return role.(string)
}

// RateLimitMiddleware 简单的速率限制中间件
func RateLimitMiddleware(maxRequests int, window time.Duration) gin.HandlerFunc {
	// 这里可以实现更复杂的速率限制逻辑
	// 目前返回一个简单的中间件
	return func(c *gin.Context) {
		// TODO: 实现速率限制逻辑
		c.Next()
	}
}
