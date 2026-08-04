package appdefaults

import "github.com/gin-gonic/gin"

// Handler contains views required by workspace application-default routes.
type Handler interface {
	ListResourcesRules(c *gin.Context)
	CreateResourcesRule(c *gin.Context)
	UpdateResourcesRule(c *gin.Context)
	DeleteResourcesRule(c *gin.Context)

	ListUpdateStrategyRules(c *gin.Context)
	CreateUpdateStrategyRule(c *gin.Context)
	UpdateUpdateStrategyRule(c *gin.Context)
	DeleteUpdateStrategyRule(c *gin.Context)

	ListDevModeRules(c *gin.Context)
	CreateDevModeRule(c *gin.Context)
	UpdateDevModeRule(c *gin.Context)
	DeleteDevModeRule(c *gin.Context)

	ListLifecycleRules(c *gin.Context)
	CreateLifecycleRule(c *gin.Context)
	UpdateLifecycleRule(c *gin.Context)
	DeleteLifecycleRule(c *gin.Context)

	ListProbeRules(c *gin.Context)
	CreateProbeRule(c *gin.Context)
	UpdateProbeRule(c *gin.Context)
	DeleteProbeRule(c *gin.Context)

	ListLabelsRules(c *gin.Context)
	CreateLabelsRule(c *gin.Context)
	UpdateLabelsRule(c *gin.Context)
	DeleteLabelsRule(c *gin.Context)

	ListAnnotationsRules(c *gin.Context)
	CreateAnnotationsRule(c *gin.Context)
	UpdateAnnotationsRule(c *gin.Context)
	DeleteAnnotationsRule(c *gin.Context)

	ListTkeRouteEniRules(c *gin.Context)
	CreateTkeRouteEniRule(c *gin.Context)
	UpdateTkeRouteEniRule(c *gin.Context)
	DeleteTkeRouteEniRule(c *gin.Context)
}

// Register registers workspace application-default routes.
func Register(rg *gin.RouterGroup, h Handler) {
	rg.GET("/workspaces/:workspaceID/app-spec/resources", h.ListResourcesRules)
	rg.POST("/workspaces/:workspaceID/app-spec/resources", h.CreateResourcesRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/resources/:ruleID", h.UpdateResourcesRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/resources/:ruleID", h.DeleteResourcesRule)

	rg.GET("/workspaces/:workspaceID/app-spec/update-strategy", h.ListUpdateStrategyRules)
	rg.POST("/workspaces/:workspaceID/app-spec/update-strategy", h.CreateUpdateStrategyRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/update-strategy/:ruleID", h.UpdateUpdateStrategyRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/update-strategy/:ruleID", h.DeleteUpdateStrategyRule)

	rg.GET("/workspaces/:workspaceID/app-spec/dev-mode", h.ListDevModeRules)
	rg.POST("/workspaces/:workspaceID/app-spec/dev-mode", h.CreateDevModeRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/dev-mode/:ruleID", h.UpdateDevModeRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/dev-mode/:ruleID", h.DeleteDevModeRule)

	rg.GET("/workspaces/:workspaceID/app-spec/lifecycle", h.ListLifecycleRules)
	rg.POST("/workspaces/:workspaceID/app-spec/lifecycle", h.CreateLifecycleRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/lifecycle/:ruleID", h.UpdateLifecycleRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/lifecycle/:ruleID", h.DeleteLifecycleRule)

	rg.GET("/workspaces/:workspaceID/app-spec/probe", h.ListProbeRules)
	rg.POST("/workspaces/:workspaceID/app-spec/probe", h.CreateProbeRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/probe/:ruleID", h.UpdateProbeRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/probe/:ruleID", h.DeleteProbeRule)

	rg.GET("/workspaces/:workspaceID/app-spec/labels", h.ListLabelsRules)
	rg.POST("/workspaces/:workspaceID/app-spec/labels", h.CreateLabelsRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/labels/:ruleID", h.UpdateLabelsRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/labels/:ruleID", h.DeleteLabelsRule)

	rg.GET("/workspaces/:workspaceID/app-spec/annotations", h.ListAnnotationsRules)
	rg.POST("/workspaces/:workspaceID/app-spec/annotations", h.CreateAnnotationsRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/annotations/:ruleID", h.UpdateAnnotationsRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/annotations/:ruleID", h.DeleteAnnotationsRule)

	rg.GET("/workspaces/:workspaceID/app-spec/tke-route-eni", h.ListTkeRouteEniRules)
	rg.POST("/workspaces/:workspaceID/app-spec/tke-route-eni", h.CreateTkeRouteEniRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/tke-route-eni/:ruleID", h.UpdateTkeRouteEniRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/tke-route-eni/:ruleID", h.DeleteTkeRouteEniRule)
}
