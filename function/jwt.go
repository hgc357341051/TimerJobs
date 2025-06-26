package function

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig JWT 配置结构体
// swagger:model JWTConfig
// 示例：{"secret_key":"your-secret-key","expire_time":7200,"refresh_time":86400,"issuer":"xiaohu-admin","audience":"xiaohu-users"}
type JWTConfig struct {
	SecretKey   string        `json:"secret_key"`
	ExpireTime  time.Duration `json:"expire_time"`
	RefreshTime time.Duration `json:"refresh_time"`
	Issuer      string        `json:"issuer"`
	Audience    string        `json:"audience"`
}

// 默认配置（生产环境请务必更换 SecretKey！）
var defaultJWTConfig = JWTConfig{
	SecretKey:   "your-secret-key-change-this-in-production", // 生产环境请更换
	ExpireTime:  2 * time.Hour,
	RefreshTime: 24 * time.Hour,
	Issuer:      "xiaohu-admin",
	Audience:    "xiaohu-users",
}

// JWTClaims JWT声明结构体
// swagger:model JWTClaims
// 示例：{"user_id":1,"username":"admin","role":"admin"}
type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateSecretKey 生成安全的密钥
func GenerateSecretKey(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成密钥失败: %v", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateToken 生成JWT Token
func GenerateToken(userID uint, username, role string) (string, error) {
	return GenerateTokenWithConfig(userID, username, role, defaultJWTConfig)
}

// GenerateTokenWithConfig 使用自定义配置生成JWT Token
func GenerateTokenWithConfig(userID uint, username, role string, config JWTConfig) (string, error) {
	now := time.Now()

	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(config.ExpireTime)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    config.Issuer,
			Audience:  []string{config.Audience},
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.SecretKey))
}

// ParseToken 解析JWT Token
func ParseToken(tokenString string) (*JWTClaims, error) {
	return ParseTokenWithConfig(tokenString, defaultJWTConfig)
}

// ParseTokenWithConfig 使用自定义配置解析JWT Token
func ParseTokenWithConfig(tokenString string, config JWTConfig) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名方法: %v", token.Header["alg"])
		}
		return []byte(config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析token失败: %v", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("无效的token")
}

// ValidateToken 验证Token是否有效
func ValidateToken(tokenString string) (*JWTClaims, error) {
	return ParseToken(tokenString)
}

// RefreshToken 刷新Token
func RefreshToken(tokenString string) (string, error) {
	return RefreshTokenWithConfig(tokenString, defaultJWTConfig)
}

// RefreshTokenWithConfig 使用自定义配置刷新Token
func RefreshTokenWithConfig(tokenString string, config JWTConfig) (string, error) {
	claims, err := ParseTokenWithConfig(tokenString, config)
	if err != nil {
		return "", fmt.Errorf("解析原token失败: %v", err)
	}

	// 检查是否在刷新时间范围内
	now := time.Now()
	if claims.ExpiresAt.Time.Sub(now) > config.RefreshTime {
		return "", fmt.Errorf("token还未到刷新时间")
	}

	// 生成新token
	return GenerateTokenWithConfig(claims.UserID, claims.Username, claims.Role, config)
}

// ExtractTokenFromHeader 从请求头中提取token
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("授权头为空")
	}
	// 支持 "Bearer token" 和 "token" 格式
	if strings.HasPrefix(authHeader, "Bearer ") {
		return authHeader[7:], nil
	}
	return authHeader, nil
}

// GetTokenInfo 获取Token信息
func GetTokenInfo(tokenString string) (map[string]interface{}, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
		"exp":      claims.ExpiresAt.Time,
		"iat":      claims.IssuedAt.Time,
		"iss":      claims.Issuer,
		"aud":      claims.Audience,
	}, nil
}
