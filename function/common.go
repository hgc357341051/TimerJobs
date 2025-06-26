package function

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// JsonData JSON响应数据结构
type JsonData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// PageData 分页响应数据结构
type PageData struct {
	Code       int         `json:"code"`
	Msg        string      `json:"msg,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Total      int64       `json:"total,omitempty"`
	PagesTotal int64       `json:"pages_total,omitempty"`
	Page       int         `json:"page,omitempty"`
	PageSize   int         `json:"page_size,omitempty"`
}

// JsonRes 发送JSON响应
func JsonRes(c *gin.Context, code int, msg string, data interface{}) {
	c.AbortWithStatusJSON(http.StatusOK, JsonData{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

// JsonPage 发送分页JSON响应
func JsonPage(c *gin.Context, msg string, data interface{}, total, pagesTotal int64, page, pageSize int) {
	c.AbortWithStatusJSON(http.StatusOK, PageData{
		Code:       http.StatusOK,
		Msg:        msg,
		Data:       data,
		Total:      total,
		PagesTotal: pagesTotal,
		Page:       page,
		PageSize:   pageSize,
	})
}

// Ok 发送成功响应
func Ok(c *gin.Context, msg string, data interface{}) {
	JsonRes(c, http.StatusOK, msg, data)
}

// No 发送失败响应
func No(c *gin.Context, msg string, data interface{}) {
	JsonRes(c, http.StatusBadRequest, msg, data)
}

// Error 发送错误响应
func Error(c *gin.Context, code int, msg string, data interface{}) {
	JsonRes(c, code, msg, data)
}

// JsonEncode 将任意对象编码为JSON字符串
func JsonEncode(data interface{}) (string, error) {
	if data == nil {
		return "null", nil
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return "", errors.New("JSON编码失败: " + err.Error())
	}
	return string(bytes), nil
}

// JsonEncodeStr 将任意对象编码为JSON字符串（忽略错误）
func JsonEncodeStr(data interface{}) string {
	result, _ := JsonEncode(data)
	return result
}

// JsonDecode 将JSON字符串解析为指定结构或map
func JsonDecode(jsonStr string, target interface{}) error {
	if jsonStr == "" {
		return errors.New("JSON字符串为空")
	}

	if target == nil {
		return errors.New("目标对象为空")
	}

	return json.Unmarshal([]byte(jsonStr), target)
}

// 辅助函数：通用获取参数并转换类型
func getValue[T any](getter func(string) string, key string, defaultValue T, parse func(string) (T, error)) T {
	value := getter(key)
	if value == "" {
		return defaultValue
	}
	if parse == nil {
		return any(strings.TrimSpace(value)).(T)
	}
	parsed, err := parse(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// 简化后的 GetQuery* 系列
func GetQueryInt(c *gin.Context, key string, defaultValue int) int {
	return getValue(c.Query, key, defaultValue, strconv.Atoi)
}
func GetQueryString(c *gin.Context, key string, defaultValue string) string {
	return getValue[string](c.Query, key, defaultValue, nil)
}
func GetQueryBool(c *gin.Context, key string, defaultValue bool) bool {
	return getValue(c.Query, key, defaultValue, strconv.ParseBool)
}

// 简化后的 GetForm* 系列
func GetFormInt(c *gin.Context, key string, defaultValue int) int {
	return getValue(c.PostForm, key, defaultValue, strconv.Atoi)
}
func GetFormString(c *gin.Context, key string, defaultValue string) string {
	return getValue[string](c.PostForm, key, defaultValue, nil)
}
func GetFormBool(c *gin.Context, key string, defaultValue bool) bool {
	return getValue(c.PostForm, key, defaultValue, strconv.ParseBool)
}

// ValidateRequired 验证必填字段
func ValidateRequired(c *gin.Context, fields ...string) error {
	for _, field := range fields {
		value := c.PostForm(field)
		if strings.TrimSpace(value) == "" {
			return errors.New("字段 " + field + " 不能为空")
		}
	}
	return nil
}

// ValidateLength 验证字符串长度
func ValidateLength(value string, min, max int) error {
	length := len(strings.TrimSpace(value))
	if length < min {
		return errors.New("字符串长度不能少于 " + strconv.Itoa(min) + " 个字符")
	}
	if max > 0 && length > max {
		return errors.New("字符串长度不能超过 " + strconv.Itoa(max) + " 个字符")
	}
	return nil
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("邮箱不能为空")
	}

	// 简单的邮箱格式验证
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return errors.New("邮箱格式不正确")
	}

	return nil
}

// ValidatePhone 验证手机号格式
func ValidatePhone(phone string) error {
	if phone == "" {
		return errors.New("手机号不能为空")
	}

	// 简单的手机号格式验证（中国大陆）
	if len(phone) != 11 || !strings.HasPrefix(phone, "1") {
		return errors.New("手机号格式不正确")
	}

	return nil
}

// Unauthorized 发送未授权响应
func Unauthorized(c *gin.Context, msg string) {
	if msg == "" {
		msg = "未授权访问"
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, JsonData{
		Code: http.StatusUnauthorized,
		Msg:  msg,
		Data: nil,
	})
}

// Forbidden 发送禁止访问响应
func Forbidden(c *gin.Context, msg string) {
	if msg == "" {
		msg = "禁止访问"
	}
	c.AbortWithStatusJSON(http.StatusForbidden, JsonData{
		Code: http.StatusForbidden,
		Msg:  msg,
		Data: nil,
	})
}

// NotFound 发送未找到响应
func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = "资源未找到"
	}
	c.AbortWithStatusJSON(http.StatusNotFound, JsonData{
		Code: http.StatusNotFound,
		Msg:  msg,
		Data: nil,
	})
}

// ServerError 发送服务器错误响应
func ServerError(c *gin.Context, msg string) {
	if msg == "" {
		msg = "服务器内部错误"
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, JsonData{
		Code: http.StatusInternalServerError,
		Msg:  msg,
		Data: nil,
	})
}

// TestRetryFunction 测试重试功能的函数
// 这个函数会随机失败，用于测试重试机制
// func TestRetryFunction() string {
// 	rand.Seed(time.Now().UnixNano())
// 	if rand.Float64() < 0.7 {
// 		panic("模拟任务执行失败，用于测试重试机制")
// 	}
// 	return "任务执行成功！"
// }
