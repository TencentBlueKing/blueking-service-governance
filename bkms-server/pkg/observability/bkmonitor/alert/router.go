package alert

import "github.com/gin-gonic/gin"

// Handler 定义告警策略与告警事件 Gin 路由所需的视图方法。
type Handler interface {
	// 告警策略
	ListAlertStrategies(c *gin.Context)
	GetAlertStrategy(c *gin.Context)
	CreateAlertStrategy(c *gin.Context)
	UpdateAlertStrategy(c *gin.Context)
	DeleteAlertStrategy(c *gin.Context)
	SyncAlertStrategy(c *gin.Context)
	SwitchAlertStrategy(c *gin.Context)

	// 告警事件
	ListAlertEvents(c *gin.Context)
	ListAlertEventsByStrategy(c *gin.Context)
	GetAlertDetail(c *gin.Context)
}

// Register 注册告警策略与告警事件路由。
func Register(rg *gin.RouterGroup, h Handler) {
	// 告警策略（按应用维度）
	appGroup := rg.Group("/workspaces/:workspaceID/apps/:appID/bkmonitor")
	{
		// 查询告警策略列表
		appGroup.GET("/alert-strategies", h.ListAlertStrategies)
		// 查询应用下的告警事件列表
		appGroup.GET("/alerts", h.ListAlertEvents)
		// 查询单个告警策略详情
		appGroup.GET("/alert-strategies/:strategyID", h.GetAlertStrategy)
		// 创建告警策略
		appGroup.POST("/alert-strategies", h.CreateAlertStrategy)
		// 更新告警策略
		appGroup.PUT("/alert-strategies/:strategyID", h.UpdateAlertStrategy)
		// 删除告警策略
		appGroup.DELETE("/alert-strategies/:strategyID", h.DeleteAlertStrategy)
		// 同步告警策略到监控平台
		appGroup.POST("/alert-strategies/:strategyID/sync", h.SyncAlertStrategy)
		// 启停告警策略
		appGroup.POST("/alert-strategies/:strategyID/switch", h.SwitchAlertStrategy)
		// 查询指定策略下的告警事件列表
		appGroup.GET("/alert-strategies/:strategyID/alerts", h.ListAlertEventsByStrategy)
	}

	// 告警事件（按工作空间维度）
	wsGroup := rg.Group("/workspaces/:workspaceID/bkmonitor")
	{
		// 查询单个告警事件详情
		wsGroup.GET("/alerts/:alertID", h.GetAlertDetail)
	}
}
