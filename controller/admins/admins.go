package admins

import (
	"time"

	"xiaohuAdmin/function"
	"xiaohuAdmin/global"
	"xiaohuAdmin/middlewares"
	"xiaohuAdmin/models/admins"

	"github.com/gin-gonic/gin"
)

type AdminController struct{}

// LoginRequest 管理员登录请求结构体
// 示例：{"username":"admin","password":"123456"}
type LoginRequest struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

// RegisterRequest 管理员注册请求结构体
// 示例：{"username":"admin","password":"123456","email":"admin@xx.com","role":"admin"}
type RegisterRequest struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
	Email    string `form:"email" json:"email"`
	Role     string `form:"role" json:"role"`
}

// UpdateProfileRequest 更新用户信息结构体
// 示例：{"email":"new@xx.com","role":"user"}
type UpdateProfileRequest struct {
	Email string `form:"email" json:"email"`
	Role  string `form:"role" json:"role"`
}

// AdminStatusRequest 管理员状态变更结构体
// 示例：{"id":1,"status":0}
type AdminStatusRequest struct {
	ID     uint `form:"id" json:"id" binding:"required"`
	Status int  `form:"status" json:"status" binding:"required"`
}

// AdminListRequest 管理员列表查询结构体
// 示例：{"page":1,"size":10}
type AdminListRequest struct {
	Page int `form:"page" json:"page"`
	Size int `form:"size" json:"size"`
}

// bindAndValidate 辅助函数，统一参数绑定和错误响应
func bindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBind(req); err != nil {
		function.No(c, "参数错误："+err.Error(), nil)
		return false
	}
	return true
}

// Login 管理员登录
// @Summary 管理员登录
// @Description 管理员登录接口
// @Tags 管理员
// @Accept json
// @Produce json
// @Param data body LoginRequest true "登录参数" 例：{"username":"admin","password":"123456"}
// @Success 200 {object} function.JsonData "登录成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Failure 401 {object} function.JsonData "用户名或密码错误"
// @Router /admin/login [post]
func (*AdminController) Login(c *gin.Context) {
	var req LoginRequest
	if !bindAndValidate(c, &req) {
		return
	}

	// 查找用户
	var admin admins.Admin
	if err := global.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		global.ZapLog.Warn("登录失败：用户不存在",
			global.LogField("username", req.Username),
			global.LogField("ip", c.ClientIP()))
		function.Unauthorized(c, "用户名或密码错误")
		return
	}

	// 检查用户状态
	if admin.Status != 1 {
		function.Unauthorized(c, "用户已被禁用")
		return
	}

	// 验证密码（这里使用明文密码，实际项目中应该使用加密密码）
	if admin.Password != req.Password {
		global.ZapLog.Warn("登录失败：密码错误",
			global.LogField("username", req.Username),
			global.LogField("ip", c.ClientIP()))
		function.Unauthorized(c, "用户名或密码错误")
		return
	}

	// 生成JWT令牌
	token, err := middlewares.GenerateToken(&admin)
	if err != nil {
		global.ZapLog.Error("生成JWT令牌失败", global.LogError(err))
		function.ServerError(c, "登录失败")
		return
	}

	// 更新最后登录时间
	global.DB.Model(&admin).Update("last_login", time.Now())

	global.ZapLog.Info("管理员登录成功",
		global.LogField("username", admin.Username),
		global.LogField("ip", c.ClientIP()))

	function.Ok(c, "登录成功", gin.H{
		"token": token,
		"user": gin.H{
			"id":       admin.ID,
			"username": admin.Username,
			"email":    admin.Email,
			"role":     admin.Role,
		},
	})
}

// Register 管理员注册
// @Summary 管理员注册
// @Description 注册新的管理员账户
// @Tags 管理员
// @Accept json
// @Produce json
// @Param data body RegisterRequest true "注册参数" 例：{"username":"admin","password":"123456","email":"admin@xx.com","role":"admin"}
// @Success 200 {object} function.JsonData "注册成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /admin/register [post]
func (*AdminController) Register(c *gin.Context) {
	var req RegisterRequest
	if !bindAndValidate(c, &req) {
		return
	}

	// 检查用户名是否已存在
	var existingAdmin admins.Admin
	if err := global.DB.Where("username = ?", req.Username).First(&existingAdmin).Error; err == nil {
		function.No(c, "用户名已存在", nil)
		return
	}

	// 验证邮箱格式
	if req.Email != "" {
		if err := function.ValidateEmail(req.Email); err != nil {
			function.No(c, "邮箱格式不正确", nil)
			return
		}
	}

	// 设置默认角色
	if req.Role == "" {
		req.Role = "admin"
	}

	// 创建管理员账户
	admin := admins.Admin{
		Username: req.Username,
		Password: req.Password, // 实际项目中应该加密
		Email:    req.Email,
		Role:     req.Role,
		Status:   1,
	}

	if err := global.DB.Create(&admin).Error; err != nil {
		global.ZapLog.Error("创建管理员失败", global.LogError(err))
		function.ServerError(c, "注册失败")
		return
	}

	global.ZapLog.Info("管理员注册成功",
		global.LogField("username", admin.Username),
		global.LogField("ip", c.ClientIP()))

	function.Ok(c, "注册成功", gin.H{
		"id":       admin.ID,
		"username": admin.Username,
		"email":    admin.Email,
		"role":     admin.Role,
	})
}

// GetProfile 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的信息
// @Tags 管理员
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} function.JsonData "获取成功"
// @Router /admin/profile [get]
func (*AdminController) GetProfile(c *gin.Context) {
	user := middlewares.GetCurrentUser(c)
	if user == nil {
		function.Unauthorized(c, "用户未登录")
		return
	}

	function.Ok(c, "获取成功", gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"status":     user.Status,
		"last_login": user.LastLogin,
		"created_at": user.CreatedAt,
	})
}

// UpdateProfile 更新用户信息
// @Summary 更新用户信息
// @Description 更新当前登录用户的信息
// @Tags 管理员
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body UpdateProfileRequest true "更新参数" 例：{"email":"new@xx.com","role":"user"}
// @Success 200 {object} function.JsonData "更新成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /admin/profile [post]
func (*AdminController) UpdateProfile(c *gin.Context) {
	user := middlewares.GetCurrentUser(c)
	if user == nil {
		function.Unauthorized(c, "用户未登录")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		function.No(c, "参数错误："+err.Error(), nil)
		return
	}

	updates := make(map[string]interface{})

	if req.Email != "" {
		if err := function.ValidateEmail(req.Email); err != nil {
			function.No(c, "邮箱格式不正确", nil)
			return
		}
		updates["email"] = req.Email
	}

	if req.Password != "" {
		updates["password"] = req.Password // 实际项目中应该加密
	}

	if len(updates) == 0 {
		function.No(c, "没有需要更新的字段", nil)
		return
	}

	if err := global.DB.Model(user).Updates(updates).Error; err != nil {
		global.ZapLog.Error("更新用户信息失败", global.LogError(err))
		function.ServerError(c, "更新失败")
		return
	}

	global.ZapLog.Info("用户信息更新成功",
		global.LogField("username", user.Username),
		global.LogField("ip", c.ClientIP()))

	function.Ok(c, "更新成功", nil)
}

// GetAdminList 获取管理员列表
// @Summary 管理员列表
// @Description 分页获取管理员列表
// @Tags 管理员
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} function.PageData "分页数据"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /admin/list [get]
func (*AdminController) GetAdminList(c *gin.Context) {
	var req AdminListRequest
	if !bindAndValidate(c, &req) {
		return
	}

	// 获取管理员列表
	var adminList []admins.Admin
	var total int64

	// 获取总数
	global.DB.Model(&admins.Admin{}).Count(&total)

	// 获取分页数据
	if err := global.DB.Offset((req.Page - 1) * req.Size).Limit(req.Size).Find(&adminList).Error; err != nil {
		global.ZapLog.Error("获取管理员列表失败", global.LogError(err))
		function.ServerError(c, "获取列表失败")
		return
	}

	function.Ok(c, "获取成功", function.PageData{
		Total:    total,
		Page:     req.Page,
		PageSize: req.Size,
		Data:     adminList,
	})
}

// UpdateAdminStatus 更新管理员状态
// @Summary 修改管理员状态
// @Description 启用/禁用管理员
// @Tags 管理员
// @Accept json
// @Produce json
// @Param data body AdminStatusRequest true "状态参数" 例：{"id":1,"status":0}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /admin/status [post]
func (*AdminController) UpdateAdminStatus(c *gin.Context) {
	var req AdminStatusRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if req.ID == 0 {
		function.No(c, "管理员ID不能为空", nil)
		return
	}

	if req.Status != 0 && req.Status != 1 {
		function.No(c, "状态值无效", nil)
		return
	}

	var admin admins.Admin
	if err := global.DB.First(&admin, req.ID).Error; err != nil {
		function.NotFound(c, "管理员不存在")
		return
	}

	if err := global.DB.Model(&admin).Update("status", req.Status).Error; err != nil {
		global.ZapLog.Error("更新管理员状态失败", global.LogError(err))
		function.ServerError(c, "更新失败")
		return
	}

	statusText := "启用"
	if req.Status == 0 {
		statusText = "禁用"
	}

	global.ZapLog.Info("管理员状态更新成功",
		global.LogField("admin_id", req.ID),
		global.LogField("status", statusText))

	function.Ok(c, "状态更新成功", nil)
}

// DeleteAdmin 删除管理员
// @Summary 删除管理员
// @Description 删除指定管理员
// @Tags 管理员
// @Accept json
// @Produce json
// @Param id query int true "管理员ID"
// @Success 200 {object} function.JsonData "删除成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /admin/delete [post]
func (*AdminController) DeleteAdmin(c *gin.Context) {
	id := function.GetQueryInt(c, "id", 0)

	if id == 0 {
		function.No(c, "管理员ID不能为空", nil)
		return
	}

	currentUser := middlewares.GetCurrentUser(c)
	if currentUser != nil && currentUser.ID == uint(id) {
		function.No(c, "不能删除自己的账户", nil)
		return
	}

	var admin admins.Admin
	if err := global.DB.First(&admin, id).Error; err != nil {
		function.NotFound(c, "管理员不存在")
		return
	}

	if err := global.DB.Delete(&admin).Error; err != nil {
		global.ZapLog.Error("删除管理员失败", global.LogError(err))
		function.ServerError(c, "删除失败")
		return
	}

	global.ZapLog.Info("管理员删除成功",
		global.LogField("admin_id", id),
		global.LogField("username", admin.Username))

	function.Ok(c, "删除成功", nil)
}

func (*AdminController) IndexHtml(c *gin.Context) {
	c.HTML(200, "index/index.html", gin.H{
		"title": "Hello Gin",
		"name":  "xiaohu",
		"html":  "",
	})
}
