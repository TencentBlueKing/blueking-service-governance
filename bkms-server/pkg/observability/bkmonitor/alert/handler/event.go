/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
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
//	@Param		alertDisplayName	query		string		false	"按告警展示名称过滤"
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
	var targetEnv *envmodel.Environment
	if envName != "" {
		_, targetEnv, err = ginperm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, envName, ginperm.TypeView)
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
	strategyIDs, remoteToBKMS := collectRemoteStrategyIDsForAppAlerts(rules, queryInput.AlertDisplayName)
	if len(strategyIDs) == 0 {
		ginutils.OK(c, &serializer.ListAlertEventsResp{Data: &serializer.ListAlertEventsOutput{
			Count: 0, Results: []*serializer.AlertEventOutput{},
		}})
		return
	}

	operator := auth.MustGetUser(ctx).ID
	searchInput := queryInput.ToSearchInput()
	if targetEnv != nil {
		searchInput.ClusterID = targetEnv.Cluster.ClusterID
		searchInput.Namespace = targetEnv.Cluster.Namespace
	}
	resp, err := alertevent.NewService().SearchByStrategyIDs(ctx, ws, operator, strategyIDs, searchInput)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "search alerts by app"))
		return
	}

	results := lo.Map(resp.Alerts, func(a bkmapi.AlertEvent, _ int) *serializer.AlertEventOutput {
		meta := remoteToBKMS[a.StrategyID]
		return serializer.NewAlertEventOutput(a, meta.StrategyID, meta.AlertDisplayName)
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
//	@Param		alertDisplayName	query		string		false	"按告警展示名称过滤"
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
	// 当前接口已通过 strategyID 定位到单条 BKMS 本地策略；若调用方还额外传入 alertDisplayName，
	// 则可先用本地展示名做一次短路过滤。
	if alertDisplayName := strings.TrimSpace(queryInput.AlertDisplayName); alertDisplayName != "" &&
		!strings.Contains(rule.DisplayName, alertDisplayName) {
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
		return serializer.NewAlertEventOutput(a, rule.ID.Hex(), rule.DisplayName)
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

	ginutils.OK(c, serializer.NewGetAlertDetailResp(detail, extractAlertDisplayNameFromAlertDetail(detail)))
}

type alertEventBackfillInfo struct {
	StrategyID       string
	AlertDisplayName string
}

// collectRemoteStrategyIDsForAppAlerts 从应用下的本地告警策略中收集远端监控策略 ID。
// 若传入 alertDisplayName，则优先按本地展示名预过滤，避免前端依赖远端 alertName 命名规则。
// 这里默认同一 app 范围内，一个 remoteStrategyID 只应归属一个本地策略。
// 返回 remoteStrategyID 到事件列表回填信息的映射，供结果组装时回填本地策略 ID 与展示名。
func collectRemoteStrategyIDsForAppAlerts(
	rules []alertstrategy.AlertStrategy,
	alertDisplayName string,
) ([]int64, map[int64]alertEventBackfillInfo) {
	alertDisplayName = strings.TrimSpace(alertDisplayName)
	seen := make(map[int64]struct{})
	remoteToBKMS := make(map[int64]alertEventBackfillInfo)
	ids := make([]int64, 0)
	for _, rule := range rules {
		if alertDisplayName != "" && !strings.Contains(rule.DisplayName, alertDisplayName) {
			continue
		}
		for _, ref := range rule.RemoteRefs {
			if ref.RemoteStrategyID <= 0 {
				continue
			}
			if _, ok := seen[ref.RemoteStrategyID]; ok {
				continue
			}
			seen[ref.RemoteStrategyID] = struct{}{}
			remoteToBKMS[ref.RemoteStrategyID] = alertEventBackfillInfo{
				StrategyID:       rule.ID.Hex(),
				AlertDisplayName: rule.DisplayName,
			}
			ids = append(ids, ref.RemoteStrategyID)
		}
	}
	return ids, remoteToBKMS
}

// extractAlertDisplayNameFromAlertDetail 从监控详情原始名称中提取 BKMS 展示名。
// 当前远端监控策略名格式固定为 `策略名【应用名】`，返回前端展示时只返回策略名
func extractAlertDisplayNameFromAlertDetail(detail map[string]any) string {
	if detail == nil {
		return ""
	}
	for _, key := range []string{"alert_name", "alertName", "strategy_name", "strategyName"} {
		value, ok := detail[key]
		if !ok {
			continue
		}
		name, ok := value.(string)
		if !ok {
			continue
		}
		return extractAlertDisplayNameFromRemoteName(name)
	}
	return ""
}

func extractAlertDisplayNameFromRemoteName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if !strings.HasSuffix(name, "】") {
		return name
	}
	idx := strings.LastIndex(name, "【")
	if idx <= 0 {
		return name
	}
	return name[:idx]
}
