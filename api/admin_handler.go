package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"pansou/model"
	"pansou/service"
)

var adminService *service.AdminService

// InitAdminService 初始化管理端服务
func InitAdminService() {
	adminService = service.NewAdminService()
}

// AddDataHandler 添加搜索数据到MySQL和ES
func AddDataHandler(c *gin.Context) {
	var req model.AddDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "参数错误",
			"details": err.Error(),
		})
		return
	}

	// 调用服务层添加数据
	response, err := adminService.AddSearchData(&req)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "添加数据失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, response)
}

// BatchAddDataHandler 批量添加搜索数据
func BatchAddDataHandler(c *gin.Context) {
	var req model.BatchAddDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"error":   "参数错误",
			"details": err.Error(),
		})
		return
	}

	// 调用服务层批量添加数据
	response, err := adminService.BatchAddSearchData(&req)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "批量添加失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, response)
}

// GetDataHandler 查询搜索数据（分页）
func GetDataHandler(c *gin.Context) {
	keyword := c.Query("keyword")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	dataList, total, err := adminService.GetSearchDataByKeyword(keyword, page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "查询数据失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success":     true,
		"data":        dataList,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (int(total) + pageSize - 1) / pageSize,
	})
}

// DeleteDataHandler 删除搜索数据
func DeleteDataHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{
			"error": "无效的ID",
		})
		return
	}

	if err := adminService.DeleteSearchData(uint(id)); err != nil {
		c.JSON(500, gin.H{
			"error":   "删除失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// UpdateDataHandler 更新搜索数据
func UpdateDataHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{
			"error": "无效的ID",
		})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(400, gin.H{
			"error":   "参数错误",
			"details": err.Error(),
		})
		return
	}

	if err := adminService.UpdateSearchData(uint(id), updates); err != nil {
		c.JSON(500, gin.H{
			"error":   "更新失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "更新成功",
	})
}

// AdminStatsHandler 获取管理统计信息
func AdminStatsHandler(c *gin.Context) {
	// 这里可以添加统计逻辑，如数据总量、今日新增等
	c.JSON(200, gin.H{
		"success": true,
		"message": "统计信息接口",
		// 可以添加更多统计数据
	})
}
