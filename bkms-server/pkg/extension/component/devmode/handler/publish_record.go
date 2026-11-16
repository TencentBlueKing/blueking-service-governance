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
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	devmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	errInvalidRecordID = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument, "EXTENSION_DEVMODE_INVALID_RECORD_ID", "invalid publish record id",
	)
	errEnvsNotFound = bkerrs.Define(
		bkerrs.ErrCodeNotFound, "EXTENSION_DEVMODE_ENVS_NOT_FOUND", "environments not found",
	)
	errInvalidSinceTime = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument, "EXTENSION_DEVMODE_INVALID_SINCE_TIME", "invalid since time format",
	)
)

// CreateDevModePublishRecords 上报开发模式发布结果。
// CLI 在 publish 完成后调用本接口，将每个实例（pod）的发布结果写入数据库。
//
//	@ID			CreateDevModePublishRecords
//	@Summary	上报开发模式发布结果
//	@Tags		devmode
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		envName	path		string							true	"环境名称"
//	@Param		body	body		slz.CreatePublishRecordsInput	true	"发布结果上报请求体"
//	@Success	200		{object}	slz.CreatePublishRecordsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/devmode/{appID}/envs/{envName}/publish-records [post]
func (h *Handler) CreateDevModePublishRecords(c *gin.Context) {
	var uriInput slz.PublishRecordURIInput
	var input slz.CreatePublishRecordsInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := ginutils.BindJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	app, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	operator := auth.MustGetUser(ctx).ID

	records := lo.Map(input.Results, func(result slz.PublishInstanceResult, _ int) *devmode.PublishRecord {
		return &devmode.PublishRecord{
			AppID:      app.ID,
			EnvName:    uriInput.EnvName,
			Action:     devmode.RecordActionPublish,
			Instance:   result.Instance,
			BinaryName: input.BinaryName,
			FileSize:   input.FileSize,
			MD5:        input.MD5,
			Status:     devmode.RecordStatus(result.Status),
			Message:    result.Message,
			Operator:   operator,
		}
	})

	if err = store.Of(h.registry.DB, devmode.PublishRecordMeta).CreateMany(ctx, records); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create devmode publish records"))
		return
	}
	recordIDs := lo.Map(records, func(r *devmode.PublishRecord, _ int) string { return r.ID.Hex() })

	// 写入操作审计记录
	recordCtx := context.WithoutCancel(ctx)
	for _, result := range input.Results {
		go audit.AddOperationRecordAsync(
			recordCtx,
			audit.OperationTypePublish,
			audit.ResourceTypeInstance,
			result.Instance,
			audit.WithResult(publishAuditResult(result)),
			audit.WithDataAfter(map[string]any{
				"binaryName": input.BinaryName,
				"fileSize":   input.FileSize,
				"md5":        input.MD5,
				"message":    result.Message,
			}),
			audit.WithWorkspaceID(app.WorkspaceID),
			audit.WithAppID(app.ID),
			audit.WithEnvName(uriInput.EnvName),
		)
	}

	ginutils.OK(c, &slz.CreatePublishRecordsOutput{
		Data: &slz.CreatePublishRecordsData{RecordIDs: recordIDs},
	})
}

// ListDevModePublishRecords 获取开发模式发布记录列表。
// envNames 与 envTypes 二选一，跨环境统一分页，按创建时间倒序返回。
//
//	@ID			ListDevModePublishRecords
//	@Summary	获取开发模式发布记录列表
//	@Tags		devmode
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string									true	"应用 ID"
//	@Param		envNames	query		[]string								false	"环境名称，与 envTypes 二选一"
//	@Param		envTypes	query		[]string								false	"环境类型，与 envNames 二选一"
//
//	@Param		keyword		query		string									false	"搜索关键字"
//	@Param		page		query		int										true	"分页页码（从 1 开始）"
//	@Param		pageSize	query		int										true	"分页大小"
//	@Param		since		query		string									false	"只返回创建时间不早于该时间的记录（RFC3339）"
//	@Param		statuses	query		[]string								false	"状态过滤"
//
//	@Success	200			{object}	slz.ListPublishRecordsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/devmode/{appID}/publish-records [get]
func (h *Handler) ListDevModePublishRecords(c *gin.Context) {
	var uriInput slz.PublishRecordAppURIInput
	var input slz.ListPublishRecordsByScopeInput
	if err := ginutils.BindURIQuery(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := input.Validate(); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()

	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	since, err := parseSince(input.Since)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	opts := []store.Opt{devmode.KeywordMatches(input.Keyword), devmode.NewestFirst()}
	if since != nil {
		opts = append(opts, devmode.CreatedSince(*since))
	}
	if len(input.Statuses) > 0 {
		opts = append(opts, store.Where("status", bson.M{"$in": input.Statuses}))
	}

	// 按记录 ID 查询时无需解析环境，直接限定 appID + _id
	if len(input.RecordIDs) > 0 {
		ids := make([]bson.ObjectID, 0, len(input.RecordIDs))
		for _, raw := range input.RecordIDs {
			id, parseErr := bson.ObjectIDFromHex(raw)
			if parseErr != nil {
				bkerrs.AbortWithErr(c, errInvalidRecordID.With("recordID", raw).WithCause(parseErr))
				return
			}
			ids = append(ids, id)
		}
		opts = append(opts, store.Where("appID", app.ID), store.Where("_id", bson.M{"$in": ids}))
		h.respondPublishRecords(c, ctx, opts, input.Page, input.PageSize)
		return
	}

	envs, err := store.Of(h.registry.DB, envmodel.EnvMeta).
		List(ctx, envmodel.InWorkspace(app.WorkspaceID), envmodel.AvailableToApp(app.ID), envmodel.NewestFirst())
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list envs of app %s", app.ID,
		))
		return
	}
	var envNames []string
	if len(input.EnvNames) > 0 {
		visible := lo.SliceToMap(envs, func(env envmodel.Environment) (string, struct{}) {
			return env.Name, struct{}{}
		})
		notFound := lo.Filter(input.EnvNames, func(name string, _ int) bool {
			_, ok := visible[name]
			return !ok
		})
		if len(notFound) > 0 {
			bkerrs.AbortWithErr(c, errEnvsNotFound.With("envs", strings.Join(notFound, ", ")))
			return
		}
		envNames = input.EnvNames
	} else {
		envNames = lo.FilterMap(envs, func(env envmodel.Environment, _ int) (string, bool) {
			return env.Name, lo.Contains(input.EnvTypes, env.Type)
		})
	}
	if len(envNames) == 0 {
		ginutils.OK(c, slz.ListPublishRecordsOutput{
			Data: slz.PaginatedPublishRecords{Results: []*slz.PublishRecordOutputObj{}},
		})
		return
	}

	opts = append(opts, devmode.ByAppEnvs(app.ID, envNames))
	h.respondPublishRecords(c, ctx, opts, input.Page, input.PageSize)
}

// respondPublishRecords 按条件分页查询并输出发布记录
func (h *Handler) respondPublishRecords(
	c *gin.Context, ctx context.Context, opts []store.Opt, page, pageSize int64,
) {
	result, err := store.Of(h.registry.DB, devmode.PublishRecordMeta).ListPage(ctx, page, pageSize, opts...)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list devmode publish records"))
		return
	}

	outputRecords := lo.Map(result.Items, func(record devmode.PublishRecord, _ int) *slz.PublishRecordOutputObj {
		return new(slz.PublishRecordOutputObj).FromModel(record)
	})
	ginutils.OK(c, slz.ListPublishRecordsOutput{
		Data: slz.PaginatedPublishRecords{Count: result.Total, Results: outputRecords},
	})
}

// parseSince 解析 since 查询参数（RFC3339），为空时返回 nil
func parseSince(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	since, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, errInvalidSinceTime.With("since", raw).WithCause(err)
	}
	return &since, nil
}

// publishAuditResult 将单个实例的发布状态映射为审计结果。
func publishAuditResult(result slz.PublishInstanceResult) audit.Result {
	if result.Status == string(devmode.RecordStatusFailed) {
		return audit.ResultFailed
	}
	return audit.ResultSuccess
}
