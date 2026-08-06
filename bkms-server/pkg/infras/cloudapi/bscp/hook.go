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

// Package bscp 提供蓝鲸 bscp 服务的 API 调用封装 - 脚本（Hook）管理
package bscp

import (
	"context"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// CreateHook 创建脚本
func (c *ApiClient) CreateHook(ctx context.Context, req *CreateHookReq) (int64, error) {
	if err := req.Validate(); err != nil {
		return 0, errors.Wrap(err, "validate create hook req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_CreateHook",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/hooks",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": req.BizID}),
		bkapi.OptSetRequestBody(req),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return 0, errors.Wrapf(err, "call bscp create hook api, bizID: %s, name: %s", req.BizID, req.Name)
	}

	id := cast.ToInt64(mapx.Get(result, "data.id", 0))
	if id == 0 {
		return 0, errors.Errorf("create hook returned empty id, bizID: %s, name: %s", req.BizID, req.Name)
	}

	return id, nil
}

// DeleteHook 删除脚本
func (c *ApiClient) DeleteHook(ctx context.Context, req *DeleteHookReq) error {
	if err := req.Validate(); err != nil {
		return errors.Wrap(err, "validate delete hook req")
	}

	opts := []define.OperationOption{
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":  req.BizID,
			"hook_id": cast.ToString(req.HookID),
		}),
	}
	if req.Force {
		opts = append(opts, bkapi.OptSetRequestQueryParam("force", "true"))
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_DeleteHook",
			Method: "DELETE",
			Path:   "/api/v1/config/biz/{biz_id}/hooks/{hook_id}",
		},
		opts...,
	)

	_, err := c.handleOperation(ctx, op)
	if err != nil {
		return errors.Wrapf(err, "call bscp delete hook api, bizID: %s, hookID: %d", req.BizID, req.HookID)
	}

	return nil
}

// GetHook 获取脚本详情
func (c *ApiClient) GetHook(ctx context.Context, bizID string, hookID int64) (*Hook, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_GetHook",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/hooks/{hook_id}",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":  bizID,
			"hook_id": cast.ToString(hookID),
		}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(err, "call bscp get hook api, bizID: %s, hookID: %d", bizID, hookID)
	}

	hook := &Hook{
		ID:   cast.ToInt64(mapx.Get(result, "data.id", 0)),
		Name: mapx.GetStr(result, "data.spec.name"),
		Type: mapx.GetStr(result, "data.spec.type"),
		Memo: mapx.GetStr(result, "data.spec.memo"),
	}

	// 解析标签
	if tags, ok := mapx.Get(result, "data.spec.tags", nil).([]any); ok {
		for _, t := range tags {
			if str, sOk := t.(string); sOk {
				hook.Tags = append(hook.Tags, str)
			}
		}
	}

	// 解析 releases 信息
	hook.NotReleaseID = cast.ToInt64(mapx.Get(result, "data.spec.releases.notReleaseId", 0))

	// 解析 revision
	hook.Creator = mapx.GetStr(result, "data.revision.creator")
	hook.Reviser = mapx.GetStr(result, "data.revision.reviser")
	hook.CreateAt = mapx.GetStr(result, "data.revision.createAt")
	hook.UpdateAt = mapx.GetStr(result, "data.revision.updateAt")

	return hook, nil
}

// GetReleaseHook 获取版本绑定的前后置脚本
func (c *ApiClient) GetReleaseHook(ctx context.Context, bizID string, appID, releaseID int64) (*ReleaseHook, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_GetReleaseHook",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/apps/{app_id}/releases/{release_id}/hooks",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     bizID,
			"app_id":     cast.ToString(appID),
			"release_id": cast.ToString(releaseID),
		}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(
			err, "call bscp get release hook api, bizID: %s, appID: %d, releaseID: %d",
			bizID, appID, releaseID,
		)
	}

	releaseHook := new(ReleaseHook)

	// 解析前置脚本
	if mapx.Get(result, "data.preHook", nil) != nil {
		releaseHook.PreHook = &ReleaseHookDetail{
			HookID:           cast.ToInt64(mapx.Get(result, "data.preHook.hookId", 0)),
			HookName:         mapx.GetStr(result, "data.preHook.hookName"),
			HookRevisionID:   cast.ToInt64(mapx.Get(result, "data.preHook.hookRevisionId", 0)),
			HookRevisionName: mapx.GetStr(result, "data.preHook.hookRevisionName"),
			Type:             mapx.GetStr(result, "data.preHook.type"),
			Content:          mapx.GetStr(result, "data.preHook.content"),
		}
	}

	// 解析后置脚本
	if mapx.Get(result, "data.postHook", nil) != nil {
		releaseHook.PostHook = &ReleaseHookDetail{
			HookID:           cast.ToInt64(mapx.Get(result, "data.postHook.hookId", 0)),
			HookName:         mapx.GetStr(result, "data.postHook.hookName"),
			HookRevisionID:   cast.ToInt64(mapx.Get(result, "data.postHook.hookRevisionId", 0)),
			HookRevisionName: mapx.GetStr(result, "data.postHook.hookRevisionName"),
			Type:             mapx.GetStr(result, "data.postHook.type"),
			Content:          mapx.GetStr(result, "data.postHook.content"),
		}
	}

	return releaseHook, nil
}

// ListHooks 获取脚本列表
func (c *ApiClient) ListHooks(ctx context.Context, req *ListHooksReq) (*ListHooksResp, error) {
	opts := []define.OperationOption{
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": req.BizID}),
	}

	if req.Name != "" {
		opts = append(opts, bkapi.OptSetRequestQueryParam("name", req.Name))
	}
	if req.Tag != "" {
		opts = append(opts, bkapi.OptSetRequestQueryParam("tag", req.Tag))
	}
	if req.All {
		opts = append(opts, bkapi.OptSetRequestQueryParam("all", "true"))
	}
	if req.SearchKey != "" {
		opts = append(opts, bkapi.OptSetRequestQueryParam("searchKey", req.SearchKey))
	}
	if req.Start > 0 {
		opts = append(opts, bkapi.OptSetRequestQueryParam("start", cast.ToString(req.Start)))
	}
	if req.Limit > 0 {
		opts = append(opts, bkapi.OptSetRequestQueryParam("limit", cast.ToString(req.Limit)))
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListHooks",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/hooks",
		},
		opts...,
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(err, "call bscp list hooks api, bizID: %s", req.BizID)
	}

	resp := &ListHooksResp{
		Count: cast.ToInt64(mapx.Get(result, "data.count", 0)),
	}

	for _, item := range mapx.GetList(result, "data.details") {
		if v, ok := item.(map[string]any); ok {
			detail := HookListItem{
				BoundNum:            cast.ToInt64(mapx.Get(v, "boundNum", 0)),
				ConfirmDelete:       cast.ToBool(mapx.Get(v, "confirmDelete", false)),
				PublishedRevisionID: cast.ToInt64(mapx.Get(v, "publishedRevisionId", 0)),
			}

			// 解析 hook 信息
			detail.Hook = Hook{
				ID:       cast.ToInt64(mapx.Get(v, "hook.id", 0)),
				Name:     mapx.GetStr(v, "hook.spec.name"),
				Type:     mapx.GetStr(v, "hook.spec.type"),
				Memo:     mapx.GetStr(v, "hook.spec.memo"),
				Creator:  mapx.GetStr(v, "hook.revision.creator"),
				Reviser:  mapx.GetStr(v, "hook.revision.reviser"),
				CreateAt: mapx.GetStr(v, "hook.revision.createAt"),
				UpdateAt: mapx.GetStr(v, "hook.revision.updateAt"),
			}

			// 解析标签
			if tags, tOk := mapx.Get(v, "hook.spec.tags", nil).([]any); tOk {
				for _, t := range tags {
					if str, sOk := t.(string); sOk {
						detail.Hook.Tags = append(detail.Hook.Tags, str)
					}
				}
			}

			resp.Details = append(resp.Details, detail)
		}
	}

	return resp, nil
}

// UpdateConfigHook 更新服务绑定的前后置脚本
func (c *ApiClient) UpdateConfigHook(ctx context.Context, req *UpdateConfigHookReq) error {
	if err := req.Validate(); err != nil {
		return errors.Wrap(err, "validate update config hook req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_UpdateConfigHook",
			Method: "PUT",
			Path:   "/api/v1/config/biz/{biz_id}/apps/{app_id}/config_hooks",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id": req.BizID,
			"app_id": cast.ToString(req.AppID),
		}),
		bkapi.OptSetRequestBody(req),
	)

	if _, err := c.handleOperation(ctx, op); err != nil {
		return errors.Wrapf(
			err, "call bscp update config hook api, bizID: %s, appID: %d",
			req.BizID, req.AppID,
		)
	}

	return nil
}

// UpdateHook 更新脚本信息（标签、描述）
func (c *ApiClient) UpdateHook(ctx context.Context, req *UpdateHookReq) error {
	if err := req.Validate(); err != nil {
		return errors.Wrap(err, "validate update hook req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_UpdateHook",
			Method: "PUT",
			Path:   "/api/v1/config/biz/{biz_id}/hooks/{hook_id}",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":  req.BizID,
			"hook_id": cast.ToString(req.HookID),
		}),
		bkapi.OptSetRequestBody(req),
	)

	if _, err := c.handleOperation(ctx, op); err != nil {
		return errors.Wrapf(err, "call bscp update hook api, bizID: %s, hookID: %d", req.BizID, req.HookID)
	}

	return nil
}
