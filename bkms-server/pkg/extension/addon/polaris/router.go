package polaris

import "github.com/gin-gonic/gin"

// PolarisConfigHandler contains views required by polaris-config Gin routes.
type PolarisConfigHandler interface {
	ListAppPolarisConfigs(c *gin.Context)
	CreateAppPolarisConfig(c *gin.Context)
	PatchAppPolarisConfig(c *gin.Context)
	DeleteAppPolarisConfig(c *gin.Context)
	ListAppPolarisConfigVars(c *gin.Context)
	ValidateAppPolarisConfig(c *gin.Context)
	PutEnvWeight(c *gin.Context)
}

// Register registers Gin polaris-config routes.
func Register(rg *gin.RouterGroup, h PolarisConfigHandler) {
	// 获取应用的北极星配置列表
	rg.GET("/apps/:appID/deps/polaris-configs", h.ListAppPolarisConfigs)
	// 创建北极星配置
	rg.POST("/apps/:appID/deps/polaris-configs", h.CreateAppPolarisConfig)
	// 更新北极星配置
	rg.PATCH("/apps/:appID/deps/polaris-configs/:configName", h.PatchAppPolarisConfig)
	// 删除北极星配置
	rg.DELETE("/apps/:appID/deps/polaris-configs/:configName", h.DeleteAppPolarisConfig)

	// 获取北极星配置变量列表
	rg.GET("/apps/:appID/deps/polaris-configs/:configName/vars", h.ListAppPolarisConfigVars)
	// 校验北极星配置（创建前预校验）
	rg.POST("/apps/:appID/deps/polaris-configs/validate", h.ValidateAppPolarisConfig)

	// 更新已部署环境的北极星实例权重
	rg.PUT("/apps/:appID/deps/polaris-configs/:configName/envs/:envName/weight", h.PutEnvWeight)
}
