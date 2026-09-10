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

// Package bscp 提供蓝鲸 bscp 服务的 API 调用封装
package bscp

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// ListUserBizs 获取用户有权限的业务（空间）列表
func (c *ApiClient) ListUserBizs(ctx context.Context) ([]Biz, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_user_space",
			Method: "GET",
			Path:   "/api/v1/auth/user/spaces",
		},
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrap(err, "call bscp list user bizs api")
	}

	// 通过请求的 WebAnnotations 获取用户有权限的业务列表
	hasPermBizIDs := set.NewStringSet()
	perms := mapx.GetMap(result, "web_annotations.perms")
	for key, val := range perms {
		if v, ok := val.(map[string]any); ok {
			if mapx.GetBool(v, "find_business_resource") {
				hasPermBizIDs.Add(key)
			}
		}
	}

	// 转换为 Biz 列表
	var bizs []Biz
	for _, ver := range mapx.GetList(result, "data.items") {
		if v, ok := ver.(map[string]any); ok {
			spaceID := mapx.GetStr(v, "space_id")
			// 只需要用户有权限的
			if !hasPermBizIDs.Has(spaceID) {
				continue
			}
			bizs = append(bizs, Biz{ID: spaceID, Name: mapx.GetStr(v, "space_name")})
		}
	}
	return bizs, nil
}

// GetBiz 获取指定业务信息
func (c *ApiClient) GetBiz(ctx context.Context, bizID string) (*Biz, error) {
	bizs, err := c.ListUserBizs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list user bizs")
	}

	// 按 id 查找
	for _, biz := range bizs {
		if biz.ID == bizID {
			return &biz, nil
		}
	}
	return nil, errors.Errorf("biz not found, bizID: %s", bizID)
}

// ListBizServices 获取业务下的服务列表
func (c *ApiClient) ListBizServices(ctx context.Context, bizID string) ([]Service, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_app",
			Method: "GET",
			Path:   "/api/v1/config/list/app/app/biz_id/{biz_id}",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": bizID}),
		bkapi.OptSetRequestQueryParam("all", "true"),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrap(err, "call bscp list biz services api")
	}

	// 转换为 Service 列表
	var svcs []Service
	for _, ver := range mapx.GetList(result, "data.details") {
		if v, ok := ver.(map[string]any); ok {
			svcs = append(svcs, Service{
				ID:         cast.ToString(mapx.Get(v, "id", 0)),
				Name:       mapx.GetStr(v, "spec.name"),
				Alias:      mapx.GetStr(v, "spec.alias"),
				Desc:       mapx.GetStr(v, "spec.memo"),
				ConfigType: ConfigType(mapx.GetStr(v, "spec.config_type")),
				DataType:   DataType(mapx.GetStr(v, "spec.data_type")),
			})
		}
	}
	return svcs, nil
}

// GetBizService 获取指定服务
func (c *ApiClient) GetBizService(ctx context.Context, bizID, svcID string) (*Service, error) {
	services, err := c.ListBizServices(ctx, bizID)
	if err != nil {
		return nil, errors.Wrapf(err, "list biz %s services", bizID)
	}

	// 按 id 查找
	for _, svc := range services {
		if svc.ID == svcID {
			return &svc, nil
		}
	}
	return nil, errors.Errorf("service not found, bizID: %s, svcID: %s", bizID, svcID)
}

// ListServiceVersions 获取服务下版本列表
func (c *ApiClient) ListServiceVersions(ctx context.Context, bizID, svcID string) (Versions, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_release",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/apps/{app_id}/releases",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": bizID, "app_id": svcID}),
		bkapi.OptSetRequestQueryParam("all", "true"),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrap(err, "call bscp list service versions api")
	}

	// 转换为 Service 列表
	var versions Versions
	for _, ver := range mapx.GetList(result, "data.details") {
		if v, ok := ver.(map[string]any); ok {
			versions = append(versions, Version{
				ID:              cast.ToString(mapx.Get(v, "id", 0)),
				Name:            mapx.GetStr(v, "spec.name"),
				Desc:            mapx.GetStr(v, "spec.memo"),
				IsFullyReleased: mapx.GetBool(v, "status.fully_released"),
				Creator:         mapx.GetStr(v, "revision.creator"),
				CreatedAt:       mapx.GetStr(v, "revision.create_at"),
			})
		}
	}
	return versions, nil
}

// ListServiceConfigs 获取服务下的配置项列表（统一处理文件型和键值型）
func (c *ApiClient) ListServiceConfigs(ctx context.Context, bizID, svcID, versionID string) ([]Config, error) {
	svc, err := c.GetBizService(ctx, bizID, svcID)
	if err != nil {
		return nil, errors.Wrap(err, "get service")
	}

	var cfgs []Config

	switch svc.ConfigType {
	case ConfigTypeFile:
		files, lErr := c.listServiceFiles(ctx, bizID, svcID, versionID)
		if lErr != nil {
			return nil, errors.Wrap(lErr, "list service files")
		}
		for i := range files {
			cfgs = append(cfgs, &files[i])
		}
	case ConfigTypeKV:
		kvs, lErr := c.listServiceKeyValues(ctx, bizID, svcID, versionID)
		if lErr != nil {
			return nil, errors.Wrap(lErr, "list service key-values")
		}
		for i := range kvs {
			cfgs = append(cfgs, &kvs[i])
		}
	default:
		return nil, errors.Errorf("unsupported config type: %s", svc.ConfigType)
	}

	return cfgs, nil
}

// GetServiceConfig 获取指定的配置项
func (c *ApiClient) GetServiceConfig(ctx context.Context, bizID, svcID, versionID, id string) (Config, error) {
	cfgs, err := c.ListServiceConfigs(ctx, bizID, svcID, versionID)
	if err != nil {
		return nil, errors.Wrap(err, "list service configs")
	}

	for _, cfg := range cfgs {
		if cfg.ID() == id {
			return cfg, nil
		}
	}

	return nil, errors.Errorf("config %s in biz %s, svc %s, ver %s not found", id, bizID, svcID, versionID)
}

// listServiceFiles 获取服务下文件列表
func (c *ApiClient) listServiceFiles(ctx context.Context, bizID, svcID, versionID string) ([]File, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListReleasedConfigItems",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/apps/{app_id}/releases/{release_id}/config_items",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": bizID, "app_id": svcID, "release_id": versionID}),
		bkapi.OptSetRequestQueryParam("all", "true"),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrap(err, "call bscp list service configs api")
	}

	var files []File
	for _, cfg := range mapx.GetList(result, "data.details") {
		if v, ok := cfg.(map[string]any); ok {
			fileType := FileType(mapx.GetStr(v, "spec.file_type"))
			// 跳过非文本类型的文件（如：二进制）
			if fileType != FileTypeText {
				continue
			}
			files = append(files, File{
				name:      mapx.GetStr(v, "spec.name"),
				path:      mapx.GetStr(v, "spec.path"),
				desc:      mapx.GetStr(v, "spec.memo"),
				bizID:     bizID,
				svcID:     svcID,
				signature: cast.ToString(mapx.Get(v, "commit_spec.content.signature", 0)),
			})
		}
	}

	return files, nil
}

// getFileContent 获取 BSCP 上指定文件的内容（仅对文件型服务配置有效）
// 参数中的 signature 为文件签名，结合 bizID 和 svcID 可确定唯一文件
func (c *ApiClient) getFileContent(ctx context.Context, bizID, svcID, signature string) (content string, err error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "download_content",
			Method: "GET",
			Path:   "/api/v1/biz/{biz_id}/content/download",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": bizID}),
		// 该 BSCP API 通过 header 指定要下载的文件签名，以此为索引查找文件
		bkapi.OptSetRequestHeader("X-Bscp-App-Id", svcID),
		bkapi.OptSetRequestHeader("X-Bkapi-File-Content-Id", signature),
	)

	started := time.Now()
	defer metrics.ClientRequest("bscp", apiOperation.FullName(), started, &err)

	resultProvider := bkapi.NewUnmarshalResultProvider(
		func(body io.Reader, v any) error {
			content, rErr := io.ReadAll(body)
			if rErr != nil {
				return errors.Wrap(rErr, "read response body")
			}

			if len(content) == 0 {
				return errors.New("empty response content")
			}

			// 先尝试解析为 JSON，检查是否为错误响应
			result := make(map[string]any)
			if json.Unmarshal(content, &result) == nil {
				if errMsg := mapx.GetStr(result, "error.message"); errMsg != "" {
					return errors.New(errMsg)
				}
				return errors.New("unexpected JSON response format")
			}

			// 专用 ResultProvider，类型只会是 *string，直接赋值即可无需做类型检查
			*v.(*string) = string(content)
			return nil
		},
	)

	var result string
	if _, err = apiOperation.
		SetContext(ctx).
		SetResultProvider(resultProvider).
		SetResult(&result).
		Request(); err != nil {
		return "", errors.Wrap(err, "call bscp download content api")
	}

	return result, nil
}

// listServiceKeyValues 获取服务下键值对
func (c *ApiClient) listServiceKeyValues(ctx context.Context, bizID, svcID, versionID string) ([]KeyValue, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_released_kv",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/apps/{app_id}/releases/{release_id}/kvs",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": bizID, "app_id": svcID, "release_id": versionID}),
		bkapi.OptSetRequestQueryParam("all", "true"),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrap(err, "call bscp list service kvs api")
	}

	var kvs []KeyValue
	for _, cfg := range mapx.GetList(result, "data.details") {
		if v, ok := cfg.(map[string]any); ok {
			kvs = append(kvs, KeyValue{
				key:   mapx.GetStr(v, "spec.key"),
				value: mapx.GetStr(v, "spec.value"),
				desc:  mapx.GetStr(v, "spec.memo"),
			})
		}
	}

	return kvs, nil
}
