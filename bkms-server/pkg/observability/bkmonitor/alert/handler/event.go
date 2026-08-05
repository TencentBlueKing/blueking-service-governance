package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	alertevent "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/event"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/serializer"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// ListAlertEvents 查询应用下的告警事件列表
//
//	@ID			ListAlertEvents
//	@Summary	查询应用下的告警事件列表
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		appID		path		string		true	"应用 ID"
//	@Param		envName		query		string		false	"环境名称"
//	@Param		status		query		[]string	false	"告警状态"
//	@Param		severity	query		[]int		false	"告警级别"
//	@Param		startTime	query		int			false	"开始时间"
//	@Param		endTime		query		int			false	"结束时间"
//	@Param		page		query		int			true	"页码，从 1 开始"
//	@Param		pageSize	query		int			true	"每页数量，仅支持 5/10/20/50/100"
//	@Param		alertID		query		string		false	"按告警 ID 过滤"
//	@Param		alertName	query		string		false	"按告警名称过滤"
//	@Param		description	query		string		false	"按告警内容过滤（映射到 query_string）"
//	@Param		strategyName	query		string		false	"按策略名称过滤"
//	@Param		eventID		query		string		false	"按事件 ID 过滤"
//	@Param		target		query		string		false	"按目标实例过滤"
//	@Param		ordering	query		[]string	false	"排序字段列表，默认 -create_time"
//	@Success	200			{object}	serializer.ListAlertEventsResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alerts [get]
func (h *Handler) ListAlertEvents(c *gin.Context) {
	var uriInput serializer.AlertStrategyAppURIInput
	var queryInput serializer.AppScopedAlertQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	queryInput.Normalize()

	ctx := c.Request.Context()
	app, err := validateAppInWorkspace(ctx, h.registry, uriInput.WorkspaceID, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ws, err := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}

	envName := strings.TrimSpace(queryInput.EnvName)
	if envName != "" {
		_, _, err = ginperm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, envName, ginperm.TypeView)
		if err != nil {
			bkerrs.AbortWithErr(c, err)
			return
		}
	}

	rules, err := h.alertStrategyService().ListByApp(ctx, app.WorkspaceID, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list alert strategies by app"))
		return
	}
	strategyIDs, remoteToBKMS := collectRemoteStrategyIDsForAppAlerts(rules, envName)
	if len(strategyIDs) == 0 {
		ginutils.OK(c, &serializer.ListAlertEventsResp{Data: &serializer.ListAlertEventsOutput{
			Count: 0, Results: []*serializer.AlertEventOutput{},
		}})
		return
	}

	operator := auth.MustGetUser(ctx).ID
	resp, err := alertevent.NewService().SearchByStrategyIDs(
		ctx, ws, operator, strategyIDs, queryInput.ToSearchInput(),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "search alerts by app"))
		return
	}

	results := lo.Map(resp.Alerts, func(a bkmapi.AlertEvent, _ int) *serializer.AlertEventOutput {
		return serializer.NewAlertEventOutput(a, remoteToBKMS[a.StrategyID])
	})
	ginutils.OK(c, &serializer.ListAlertEventsResp{Data: &serializer.ListAlertEventsOutput{
		Count:   resp.Total,
		Results: results,
	}})
}

// ListAlertEventsByStrategy 查询单条告警策略关联的告警事件
//
//	@ID			ListAlertEventsByStrategy
//	@Summary	查询规则关联的告警事件
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		strategyID	path		string	true	"本地策略 ID"
//	@Param		status		query		[]string	false	"告警状态"
//	@Param		severity	query		[]int		false	"告警级别"
//	@Param		startTime	query		int			false	"开始时间"
//	@Param		endTime		query		int			false	"结束时间"
//	@Param		page		query		int			true	"页码，从 1 开始"
//	@Param		pageSize	query		int			true	"每页数量，仅支持 5/10/20/50/100"
//	@Param		alertID		query		string		false	"按告警 ID 过滤"
//	@Param		alertName	query		string		false	"按告警名称过滤"
//	@Param		description	query		string		false	"按告警内容过滤（映射到 query_string）"
//	@Param		strategyName	query		string		false	"按策略名称过滤"
//	@Param		eventID		query		string		false	"按事件 ID 过滤"
//	@Param		target		query		string		false	"按目标实例过滤"
//	@Param		ordering	query		[]string	false	"排序字段列表，默认 -create_time"
//	@Success	200			{object}	serializer.ListAlertEventsResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/alerts [get]
func (h *Handler) ListAlertEventsByStrategy(c *gin.Context) {
	var uriInput serializer.AlertStrategyURIInput
	var queryInput serializer.AlertQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	queryInput.Normalize()

	ctx := c.Request.Context()
	app, err := validateAppInWorkspace(ctx, h.registry, uriInput.WorkspaceID, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ws, err := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}

	strategyObjID, err := bson.ObjectIDFromHex(uriInput.StrategyID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "invalid strategy ID"))
		return
	}

	rule, err := getAlertStrategyInApp(ctx, h.registry, strategyObjID, uriInput.WorkspaceID, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapAlertStrategyLookupErr(err))
		return
	}

	strategyIDs := lo.Uniq(lo.Map(rule.RemoteRefs, func(ref alertstrategy.RemoteStrategyRef, _ int) int64 {
		return ref.RemoteStrategyID
	}))
	if len(strategyIDs) == 0 {
		ginutils.OK(c, &serializer.ListAlertEventsResp{Data: &serializer.ListAlertEventsOutput{
			Count: 0, Results: []*serializer.AlertEventOutput{},
		}})
		return
	}

	operator := auth.MustGetUser(ctx).ID
	resp, err := alertevent.NewService().SearchByStrategyIDs(
		ctx, ws, operator, strategyIDs, queryInput.ToSearchInput(),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "search alerts by strategy"))
		return
	}

	results := lo.Map(resp.Alerts, func(a bkmapi.AlertEvent, _ int) *serializer.AlertEventOutput {
		return serializer.NewAlertEventOutput(a, rule.ID.Hex())
	})
	ginutils.OK(c, &serializer.ListAlertEventsResp{Data: &serializer.ListAlertEventsOutput{
		Count:   resp.Total,
		Results: results,
	}})
}

// GetAlertDetail 查询单条告警详情
//
//	@ID			GetAlertDetail
//	@Summary	查询单条告警详情
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		alertID		path		string	true	"告警 ID"
//	@Success	200			{object}	serializer.GetAlertDetailResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkmonitor/alerts/{alertID} [get]
func (h *Handler) GetAlertDetail(c *gin.Context) {
	var uriInput serializer.AlertDetailURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	detail, err := alertevent.NewService().GetDetail(ctx, ws, auth.MustGetUser(ctx).ID, uriInput.AlertID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get alert detail"))
		return
	}

	ginutils.OK(c, &serializer.GetAlertDetailResp{Data: detail})
}

// collectRemoteStrategyIDsForAppAlerts 从应用下的本地告警策略中收集远端监控策略 ID。
// 这里默认同一 app 范围内，一个 remoteStrategyID 只应归属一个本地策略。
// 返回 remoteStrategyID 到 BKMS 本地策略 ID 的映射，供事件列表回填本地策略 ID。
func collectRemoteStrategyIDsForAppAlerts(
	rules []alertstrategy.AlertStrategy,
	envName string,
) ([]int64, map[int64]string) {
	seen := make(map[int64]struct{})
	remoteToBKMS := make(map[int64]string)
	ids := make([]int64, 0)
	for _, rule := range rules {
		for _, ref := range rule.RemoteRefs {
			if envName != "" && ref.EnvName != envName {
				continue
			}
			if ref.RemoteStrategyID <= 0 {
				continue
			}
			if _, ok := seen[ref.RemoteStrategyID]; ok {
				continue
			}
			seen[ref.RemoteStrategyID] = struct{}{}
			remoteToBKMS[ref.RemoteStrategyID] = rule.ID.Hex()
			ids = append(ids, ref.RemoteStrategyID)
		}
	}
	return ids, remoteToBKMS
}
