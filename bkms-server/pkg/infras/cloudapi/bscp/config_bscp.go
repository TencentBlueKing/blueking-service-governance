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
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// ListProjects 获取业务下的项目列表
func (c *ConfigApiClient) ListProjects(ctx context.Context, bizID string) ([]Project, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListProjects",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/projects/list",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": bizID}),
		bkapi.OptSetRequestBody(map[string]any{"all": true}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(err, "call bscp list projects api, bizID: %s", bizID)
	}

	var projects []Project
	for _, item := range mapx.GetList(result, "data.projects") {
		if v, ok := item.(map[string]any); ok {
			projects = append(projects, Project{
				ID: cast.ToInt64(mapx.Get(v, "id", 0)),
				Spec: ProjectSpec{
					Name:      mapx.GetStr(v, "spec.name"),
					Key:       mapx.GetStr(v, "spec.key"),
					IsDefault: mapx.GetBool(v, "spec.is_default"),
					EnvCount:  cast.ToInt64(mapx.Get(v, "spec.env_count", 0)),
					AppCount:  cast.ToInt64(mapx.Get(v, "spec.app_count", 0)),
				},
			})
		}
	}
	return projects, nil
}

// GetProjectByKey 根据 projectKey 获取项目信息
func (c *ConfigApiClient) GetProjectByKey(ctx context.Context, bizID, projectKey string) (*Project, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_GetProjectByKey",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/projects/query/by_key",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": bizID}),
		bkapi.OptSetRequestQueryParam("projectKey", projectKey),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(err, "call bscp get project by key api, bizID: %s, key: %s", bizID, projectKey)
	}

	id := cast.ToInt64(mapx.Get(result, "data.id", 0))
	if id == 0 {
		return nil, errors.Errorf("project not found, bizID: %s, key: %s", bizID, projectKey)
	}

	return &Project{
		ID: id,
		Spec: ProjectSpec{
			Name:      mapx.GetStr(result, "data.spec.name"),
			Key:       mapx.GetStr(result, "data.spec.key"),
			IsDefault: mapx.GetBool(result, "data.spec.is_default"),
		},
	}, nil
}

// ListEnvironments 获取项目下的环境列表
func (c *ConfigApiClient) ListEnvironments(
	ctx context.Context,
	bizID string,
	projectID int64,
) (*EnvironmentsResp, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListEnvironments",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/envs/list",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     bizID,
			"project_id": cast.ToString(projectID),
		}),
		bkapi.OptSetRequestBody(map[string]any{"all": true}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(err, "call bscp list environments api, bizID: %s, projectID: %d", bizID, projectID)
	}

	parseEnvs := func(key string) []Environment {
		var envs []Environment
		for _, item := range mapx.GetList(result, "data."+key) {
			if v, ok := item.(map[string]any); ok {
				envs = append(envs, Environment{
					ID: cast.ToInt64(mapx.Get(v, "id", 0)),
					Spec: EnvironmentSpec{
						Name: mapx.GetStr(v, "spec.name"),
						Type: mapx.GetStr(v, "spec.type"),
						Memo: mapx.GetStr(v, "spec.memo"),
					},
				})
			}
		}
		return envs
	}

	return &EnvironmentsResp{
		ProdEnvironments:    parseEnvs("prod_environments"),
		StagingEnvironments: parseEnvs("staging_environments"),
		TestEnvironments:    parseEnvs("test_environments"),
		DevEnvironments:     parseEnvs("dev_environments"),
	}, nil
}

// CreateEnvironment 在项目下创建环境
func (c *ConfigApiClient) CreateEnvironment(ctx context.Context, req *CreateEnvironmentReq) (*Environment, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate create environment req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_CreateEnvironment",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/envs",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     req.BizID,
			"project_id": cast.ToString(req.ProjectID),
		}),
		bkapi.OptSetRequestBody(req),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(
			err, "call bscp create environment api, bizID: %s, projectID: %d, name: %s",
			req.BizID, req.ProjectID, req.Name,
		)
	}

	id := cast.ToInt64(mapx.Get(result, "data.id", 0))
	if id == 0 {
		return nil, errors.Errorf(
			"create environment returned empty id, bizID: %s, projectID: %d, name: %s",
			req.BizID, req.ProjectID, req.Name,
		)
	}

	return &Environment{
		ID:   id,
		Spec: EnvironmentSpec{Name: req.Name, Type: req.Type, Memo: req.Memo},
	}, nil
}

// CreateApp 在项目环境下创建 BSCP App
func (c *ConfigApiClient) CreateApp(ctx context.Context, req *CreateAppReq) (*App, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate create app req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_CreateApp2",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/envs/{env_id}/apps",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     req.BizID,
			"project_id": cast.ToString(req.ProjectID),
			"env_id":     cast.ToString(req.EnvID),
		}),
		bkapi.OptSetRequestBody(req),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(
			err, "call bscp create app api, bizID: %s, projectID: %d, envID: %d, name: %s",
			req.BizID, req.ProjectID, req.EnvID, req.Name,
		)
	}

	id := cast.ToString(mapx.Get(result, "data.id", 0))
	if id == "" || id == "0" {
		return nil, errors.Errorf(
			"create app returned empty id, bizID: %s, projectID: %d, envID: %d, name: %s",
			req.BizID, req.ProjectID, req.EnvID, req.Name,
		)
	}

	return &App{
		ID:         id,
		Name:       req.Name,
		Alias:      req.Alias,
		Desc:       req.Memo,
		ConfigType: req.ConfigType,
		DataType:   req.DataType,
	}, nil
}

// ListEnvApps 获取项目环境下的 App 列表
func (c *ConfigApiClient) ListEnvApps(ctx context.Context, bizID string, projectID, envID int64) ([]App, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListAppsBySpaceRest2",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/envs/{env_id}/apps/list",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     bizID,
			"project_id": cast.ToString(projectID),
			"env_id":     cast.ToString(envID),
		}),
		bkapi.OptSetRequestBody(map[string]any{"all": true}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(
			err, "call bscp list env apps api, bizID: %s, projectID: %d, envID: %d",
			bizID, projectID, envID,
		)
	}

	var apps []App
	for _, ver := range mapx.GetList(result, "data.details") {
		if v, ok := ver.(map[string]any); ok {
			apps = append(apps, App{
				ID:         cast.ToString(mapx.Get(v, "id", 0)),
				Name:       mapx.GetStr(v, "spec.name"),
				Alias:      mapx.GetStr(v, "spec.alias"),
				Desc:       mapx.GetStr(v, "spec.memo"),
				ConfigType: ConfigType(mapx.GetStr(v, "spec.config_type")),
				DataType:   DataType(mapx.GetStr(v, "spec.data_type")),
			})
		}
	}
	return apps, nil
}

// GetOrCreateApp 获取或创建 BSCP App（幂等，按名称匹配）
func (c *ConfigApiClient) GetOrCreateApp(ctx context.Context, req *CreateAppReq) (*App, error) {
	services, err := c.ListEnvApps(ctx, req.BizID, req.ProjectID, req.EnvID)
	if err != nil {
		return nil, err
	}

	for _, svc := range services {
		if svc.Name == req.Name {
			return &svc, nil
		}
	}

	return c.CreateApp(ctx, req)
}
