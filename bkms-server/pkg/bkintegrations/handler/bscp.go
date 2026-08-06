package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// ListBSCPBizs 获取用户的 BSCP 业务列表
//
//	@ID			ListBSCPBizs
//	@Summary	获取用户的 BSCP 业务列表
//	@Tags		bkintegrations-bscp
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.ListBSCPBizsOutput
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/bscp/bizs [get]
func (h *Handler) ListBSCPBizs(c *gin.Context) {
	ctx := c.Request.Context()
	client, err := bscp.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bscp client"))
		return
	}

	bizs, err := client.ListUserBizs(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "list bscp user bizs"))
		return
	}

	ginutils.OK(
		c,
		&serializer.ListBSCPBizsOutput{Data: lo.Map(bizs, func(item bscp.Biz, _ int) *serializer.BSCPBizOutput {
			return new(serializer.BSCPBizOutput).FromModel(item)
		})},
	)
}

// ListBSCPServices 获取 BSCP 业务下的服务列表
//
//	@ID			ListBSCPServices
//	@Summary	获取 BSCP 业务下的服务列表
//	@Tags		bkintegrations-bscp
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		bizID	path		string	true	"BSCP 业务 ID"
//	@Success	200		{object}	serializer.ListBSCPServicesOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/bscp/bizs/{bizID}/services [get]
func (h *Handler) ListBSCPServices(c *gin.Context) {
	var uriInput serializer.BSCPBizURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bscp.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bscp client"))
		return
	}

	services, err := client.ListBizServices(ctx, uriInput.BizID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "list bscp biz services"))
		return
	}

	ginutils.OK(
		c,
		&serializer.ListBSCPServicesOutput{
			Data: lo.Map(services, func(item bscp.Service, index int) *serializer.BSCPServiceOutput {
				return new(serializer.BSCPServiceOutput).FromModel(item)
			}),
		},
	)
}

// ListBSCPConfigs 获取 BSCP 服务下的配置列表
//
//	@ID			ListBSCPConfigs
//	@Summary	获取 BSCP 服务下的配置列表
//	@Tags		bkintegrations-bscp
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		bizID		path		string	true	"BSCP 业务 ID"
//	@Param		serviceID	path		string	true	"BSCP 服务 ID"
//	@Success	200			{object}	serializer.ListBSCPConfigsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/bscp/bizs/{bizID}/services/{serviceID}/configs [get]
func (h *Handler) ListBSCPConfigs(c *gin.Context) {
	var uriInput serializer.BSCPServiceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bscp.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bscp client"))
		return
	}

	versions, err := client.ListServiceVersions(ctx, uriInput.BizID, uriInput.ServiceID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "list bscp service versions"))
		return
	}

	ver := versions.LatestFullyReleased()
	if ver == nil {
		bkerrs.AbortWithErr(c, bkerrs.WrapBSCPNotFullyReleased(
			bkerrs.New(bkerrs.ErrCodeNotFound, "no fully released version"),
			uriInput.BizID,
			uriInput.ServiceID,
		))
		return
	}

	cfgs, err := client.ListServiceConfigs(ctx, uriInput.BizID, uriInput.ServiceID, ver.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "list bscp service configs"))
		return
	}

	ginutils.OK(
		c,
		&serializer.ListBSCPConfigsOutput{
			Data: lo.Map(cfgs, func(cfg bscp.Config, _ int) *serializer.BSCPConfigOutput {
				return &serializer.BSCPConfigOutput{
					ID:   cfg.ID(),
					Name: cfg.Name(),
					Desc: cfg.Desc(),
					Type: string(cfg.Type()),
				}
			}),
		},
	)
}

// GetBSCPConfig 获取 BSCP 配置项内容
//
//	@ID			GetBSCPConfig
//	@Summary	获取 BSCP 配置项内容
//	@Tags		bkintegrations-bscp
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		bizID		path		string	true	"BSCP 业务 ID"
//	@Param		serviceID	path		string	true	"BSCP 服务 ID"
//	@Param		configID	path		string	true	"BSCP 配置项 ID"
//	@Success	200			{object}	serializer.GetBSCPConfigOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/bscp/bizs/{bizID}/services/{serviceID}/configs/{configID} [get]
func (h *Handler) GetBSCPConfig(c *gin.Context) {
	var uriInput serializer.BSCPConfigURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bscp.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bscp client"))
		return
	}

	versions, err := client.ListServiceVersions(ctx, uriInput.BizID, uriInput.ServiceID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "list bscp service versions"))
		return
	}

	ver := versions.LatestFullyReleased()
	if ver == nil {
		bkerrs.AbortWithErr(c, bkerrs.WrapBSCPNotFullyReleased(
			bkerrs.New(bkerrs.ErrCodeNotFound, "no fully released version"),
			uriInput.BizID,
			uriInput.ServiceID,
		))
		return
	}

	cfg, err := client.GetServiceConfig(ctx, uriInput.BizID, uriInput.ServiceID, ver.ID, uriInput.ConfigID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "get bscp service config"))
		return
	}

	content, err := cfg.Content(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "get bscp config content"))
		return
	}

	biz, err := client.GetBiz(ctx, uriInput.BizID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "get bscp biz"))
		return
	}

	svc, err := client.GetBizService(ctx, uriInput.BizID, uriInput.ServiceID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapBSCPAPIError(err, "get bscp service"))
		return
	}

	ginutils.OK(c, &serializer.GetBSCPConfigOutput{Data: &serializer.BSCPConfigDetailOutput{
		ID:           cfg.ID(),
		Name:         cfg.Name(),
		Desc:         cfg.Desc(),
		Type:         string(cfg.Type()),
		Content:      content,
		BizID:        biz.ID,
		BizName:      biz.Name,
		ServiceID:    svc.ID,
		ServiceName:  svc.Name,
		ServiceAlias: svc.Alias,
		VersionID:    ver.ID,
		VersionName:  ver.Name,
	}})
}

func wrapBSCPAPIError(err error, msg string) error {
	if errors.Is(err, bscp.ErrNoPermission) {
		return bkerrs.WrapBSCPNoPermission(err, msg)
	}
	return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, msg)
}
