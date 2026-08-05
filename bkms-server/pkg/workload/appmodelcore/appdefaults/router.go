package appdefaults

import "github.com/gin-gonic/gin"

// Handler contains views required by workspace application-default routes.
type Handler interface {
	// Resources rules
	ListResourcesRules(c *gin.Context)
	CreateResourcesRule(c *gin.Context)
	UpdateResourcesRule(c *gin.Context)
	DeleteResourcesRule(c *gin.Context)

	// Dev mode rules
	ListDevModeRules(c *gin.Context)
	CreateDevModeRule(c *gin.Context)
	UpdateDevModeRule(c *gin.Context)
	DeleteDevModeRule(c *gin.Context)
}

// Register registers workspace application-default routes.
func Register(rg *gin.RouterGroup, h Handler) {
	// Resources rules
	rg.GET("/workspaces/:workspaceID/app-spec/resources", h.ListResourcesRules)
	rg.POST("/workspaces/:workspaceID/app-spec/resources", h.CreateResourcesRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/resources/:ruleID", h.UpdateResourcesRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/resources/:ruleID", h.DeleteResourcesRule)

	// Dev mode rules
	rg.GET("/workspaces/:workspaceID/app-spec/dev-mode", h.ListDevModeRules)
	rg.POST("/workspaces/:workspaceID/app-spec/dev-mode", h.CreateDevModeRule)
	rg.PUT("/workspaces/:workspaceID/app-spec/dev-mode/:ruleID", h.UpdateDevModeRule)
	rg.DELETE("/workspaces/:workspaceID/app-spec/dev-mode/:ruleID", h.DeleteDevModeRule)
}
