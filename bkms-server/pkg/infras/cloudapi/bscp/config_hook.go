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

package bscp

import (
	"context"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// CreateHook 在项目下创建脚本
func (c *ConfigApiClient) CreateHook(ctx context.Context, req *CreateHookReq) (int64, error) {
	if err := req.Validate(); err != nil {
		return 0, errors.Wrap(err, "validate create hook req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_CreateHook2",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/hooks",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     req.BizID,
			"project_id": cast.ToString(req.ProjectID),
		}),
		bkapi.OptSetRequestBody(req),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return 0, errors.Wrapf(
			err, "call bscp create hook api, bizID: %s, projectID: %d, name: %s",
			req.BizID, req.ProjectID, req.Name,
		)
	}

	id := cast.ToInt64(mapx.Get(result, "data.id", 0))
	if id == 0 {
		return 0, errors.Errorf(
			"create hook returned empty id, bizID: %s, projectID: %d, name: %s",
			req.BizID, req.ProjectID, req.Name,
		)
	}

	return id, nil
}

// ListHooks 获取项目下脚本列表
func (c *ConfigApiClient) ListHooks(ctx context.Context, req *ListHooksReq) (*ListHooksResp, error) {
	opts := []define.OperationOption{
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     req.BizID,
			"project_id": cast.ToString(req.ProjectID),
		}),
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

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListHooks2",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/hooks",
		},
		opts...,
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"call bscp list hooks api, bizID: %s, projectID: %d",
			req.BizID,
			req.ProjectID,
		)
	}

	resp := &ListHooksResp{
		Count: cast.ToInt64(mapx.Get(result, "data.count", 0)),
	}

	for _, item := range mapx.GetList(result, "data.details") {
		if v, ok := item.(map[string]any); ok {
			detail := HookListItem{
				BoundNum:            cast.ToInt64(mapx.Get(v, "bound_num", 0)),
				ConfirmDelete:       cast.ToBool(mapx.Get(v, "confirm_delete", false)),
				PublishedRevisionID: cast.ToInt64(mapx.Get(v, "published_revision_id", 0)),
			}

			detail.Hook = Hook{
				ID:       cast.ToInt64(mapx.Get(v, "hook.id", 0)),
				Name:     mapx.GetStr(v, "hook.spec.name"),
				Type:     mapx.GetStr(v, "hook.spec.type"),
				Memo:     mapx.GetStr(v, "hook.spec.memo"),
				Creator:  mapx.GetStr(v, "hook.revision.creator"),
				Reviser:  mapx.GetStr(v, "hook.revision.reviser"),
				CreateAt: mapx.GetStr(v, "hook.revision.create_at"),
				UpdateAt: mapx.GetStr(v, "hook.revision.update_at"),
			}

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

// UpdateConfigHook 更新 App 绑定的前后置脚本
func (c *ConfigApiClient) UpdateConfigHook(ctx context.Context, req *UpdateConfigHookReq) error {
	if err := req.Validate(); err != nil {
		return errors.Wrap(err, "validate update config hook req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_UpdateConfigHook2",
			Method: "PUT",
			Path: "/api/v1/config/biz/{biz_id}/projects/{project_id}/envs/{env_id}" +
				"/apps/{app_id}/config_hooks",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     req.BizID,
			"project_id": cast.ToString(req.ProjectID),
			"env_id":     cast.ToString(req.EnvID),
			"app_id":     cast.ToString(req.AppID),
		}),
		bkapi.OptSetRequestBody(req),
	)

	if _, err := c.handleOperation(ctx, op); err != nil {
		return errors.Wrapf(
			err, "call bscp update config hook api, bizID: %s, projectID: %d, envID: %d, appID: %d",
			req.BizID, req.ProjectID, req.EnvID, req.AppID,
		)
	}

	return nil
}
