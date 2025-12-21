package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"pansou/config"
	"pansou/util"
)

// AdminAuthMiddleware 管理端JWT认证中间件
// 专门用于管理端接口，与客户端接口认证分离
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果未启用认证，返回错误（管理端强制要求认证）
		if !config.AppConfig.AuthEnabled {
			c.JSON(403, gin.H{
				"error": "管理端认证未启用，请在配置中启用认证功能",
				"code":  "ADMIN_AUTH_DISABLED",
			})
			c.Abort()
			return
		}

		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"error": "管理端访问需要认证：缺少认证令牌",
				"code":  "ADMIN_AUTH_TOKEN_MISSING",
			})
			c.Abort()
			return
		}

		// 解析Bearer token
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.JSON(401, gin.H{
				"error": "管理端访问需要认证：令牌格式错误",
				"code":  "ADMIN_AUTH_TOKEN_INVALID_FORMAT",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, bearerPrefix)

		// 验证token
		claims, err := util.ValidateToken(tokenString, config.AppConfig.AuthJWTSecret)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "管理端访问需要认证：令牌无效或已过期",
				"code":  "ADMIN_AUTH_TOKEN_INVALID",
			})
			c.Abort()
			return
		}

		// 可以在这里添加额外的管理员权限检查
		// 例如：检查用户是否有管理员角色
		// 这里简单实现：所有能登录的用户都可以访问管理端
		// 如果需要更细粒度的权限控制，可以在配置中添加admin_users字段

		// 将用户信息存入上下文
		c.Set("admin_username", claims.Username)
		c.Set("is_admin", true)
		c.Next()
	}
}

// AdminCORSMiddleware 管理端CORS中间件（可能需要不同的CORS配置）
func AdminCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 管理端可能需要更严格的CORS策略
		// 这里先使用与客户端相同的配置，可根据需要调整
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
