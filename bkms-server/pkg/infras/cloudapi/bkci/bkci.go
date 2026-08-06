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

// Package bkci provides api client to devops（蓝盾）
package bkci

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/tof"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

var (
	// ObjectNotFound 蓝盾对象不存在（用于 http status code 为 404 的情况）
	ObjectNotFound = errors.New("bkci object not found")
	// ProjectAlreadyExist 蓝盾项目已存在
	ProjectAlreadyExist = errors.New("bkci project already exist")
	// RepoAlreadyExist 蓝盾代码库已存在
	RepoAlreadyExist = errors.New("bkci repo already exist")
	// BuildLogExpired 构建日志已过期
	BuildLogExpired = errors.New("bkci build log expired")
	// BuildLogCleaned 构建日志已清理
	BuildLogCleaned = errors.New("bkci build log cleaned")
	// BuildLogQueryFailed 构建日志查询异常
	BuildLogQueryFailed = errors.New("bkci build log query failed")
)

// 引用自蓝盾
// https://github.com/TencentBlueKing/bk-ci/blob/master/src/backend/ci/core/log/api-log/src/main/kotlin/com/tencent/
// devops/common/log/pojo/enums/LogStatus.kt
const (
	// buildLogStatusSucceed 查询成功，返回的日志内容可直接消费。
	buildLogStatusSucceed = 0
	// buildLogStatusEmpty 日志为空，不会再有可拉取日志，按日志终态处理。
	buildLogStatusEmpty = 1
	// buildLogStatusExpired 日志已过期，对应 BKCI 原始状态码 CLEAN(2)。
	buildLogStatusExpired = 2
	// buildLogStatusCleaned 日志已清理，对应 BKCI 原始状态码 CLOSED(3)。
	buildLogStatusCleaned = 3
	// buildLogStatusFail 查询异常，BKCI 未能正常返回日志结果（BKCI 文档中定义为 999）。
	buildLogStatusFail = 999
)

// ApiClient 蓝盾 API 客户端（用户态）
type ApiClient struct {
	define.BkApiClient
	user auth.User
}

var _ Client = &ApiClient{}

// New 创建 BkciClient，根据配置返回真实客户端或 stub 客户端
func New(user auth.User) (Client, error) {
	// 测试时使用 stub 客户端
	if config.G.Development.UseStubBkCI {
		log.InfoNoContext("use stub bkci client according to config")
		return NewStub(user), nil
	}

	authorization, _ := json.Marshal(map[string]string{
		"bk_username":       user.ID,
		user.Cred.CredKey(): user.Cred.CredValue(),
	})
	client, err := bkapi.NewBkApiClient("devops", bkapi.ClientConfig{
		BkApiUrlTmpl: config.G.BkPlatUrls.BkApiUrlTmpl,
		Stage:        config.G.BkApiStages.BkCI,
		ClientOptions: []define.BkApiClientOption{
			bkapi.OptSetRequestHeader("x-bkapi-authorization", string(authorization)),
			bkapi.OptJsonResultProvider(),
			bkapi.OptJsonBodyProvider(),
			bkapi.OptTimeout(60 * time.Second),
		},
	})
	if err != nil {
		return nil, err
	}
	return &ApiClient{client, user}, nil
}

// ------------------------------------------ 蓝盾项目相关 API ------------------------------------------

// ListProjects 获取蓝盾项目列表
func (c *ApiClient) ListProjects(ctx context.Context) ([]Project, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_project_list",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/project_list",
		},
	)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, d := range mapx.GetList(result, "data") {
		if p, ok := d.(map[string]any); ok {
			projects = append(projects, Project{
				ID:            mapx.GetStr(p, "projectId"),
				Code:          mapx.GetStr(p, "projectCode"),
				Name:          mapx.GetStr(p, "projectName"),
				Creator:       mapx.GetStr(p, "creator"),
				HasManagePerm: mapx.GetBool(p, "managePermission"),
			})
		}
	}

	return projects, nil
}

// GetProject 获取蓝盾项目（通过项目 Code）
func (c *ApiClient) GetProject(ctx context.Context, projectCode string) (*Project, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_project_get",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/{projectCode}",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectCode": projectCode,
		}),
	)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	return &Project{
		ID:            mapx.GetStr(result, "data.projectId"),
		Code:          projectCode,
		Name:          mapx.GetStr(result, "data.projectName"),
		Creator:       mapx.GetStr(result, "data.creator"),
		HasManagePerm: mapx.GetBool(result, "data.managePermission"),
	}, nil
}

// CreateProject 创建蓝盾项目
// 注意：社区版不需要 obsProductID & obsProductName
func (c *ApiClient) CreateProject(
	ctx context.Context,
	projectCode, obsProductID, obsProductName string,
	userOrg *tof.Organization,
) error {
	// 构建请求体
	body := map[string]any{
		"projectName": projectCode,
		"englishName": projectCode,
		// projectType、description 为默认值，且蓝盾侧不能为空。
		// projectType = 5 表示 支撑产品
		"projectType": "5",
		"description": "蓝鲸服务治理",
	}
	// 仅当运营产品 ID 与名称均非空时才设置（社区版无这些信息）
	if obsProductID != "" && obsProductName != "" {
		body["productId"] = obsProductID
		body["productName"] = obsProductName
	}
	// 仅当有提供用户组织信息时，才设置相应的参数
	if userOrg != nil {
		body["bgId"] = userOrg.BgID
		body["bgName"] = userOrg.BgName
		body["deptId"] = userOrg.DeptID
		body["deptName"] = userOrg.DeptName
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_project_create",
			Method: "POST",
			Path:   "/v4/apigw-user/projects/project_create",
		},
		bkapi.OptSetRequestBody(body),
	)

	return c.handleOperationWithoutResult(ctx, apiOperation)
}

// ------------------------------------------ 蓝盾代码库 & OAuth API ------------------------------------------

// ListOAuthGitProjects 获取用户有 OAuth 授权给蓝盾的 Git 项目列表
func (c *ApiClient) ListOAuthGitProjects(ctx context.Context, projectCode, keyword string) ([]GitProject, error) {
	params := map[string]string{"projectId": projectCode}
	if keyword != "" {
		params["search"] = keyword
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v2_user_git_get_project",
			Method: "GET",
			Path:   "/v2/apigw-user/git/getProject",
		},
	).SetQueryParams(params)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	var projects []GitProject
	for _, d := range mapx.GetList(result, "data.project") {
		if p, ok := d.(map[string]any); ok {
			projects = append(projects, GitProject{
				ID:    mapx.GetStr(p, "id"),
				Name:  mapx.GetStr(p, "name"),
				Alias: mapx.GetStr(p, "nameWithNameSpace"),
				Url:   mapx.GetStr(p, "httpUrl"),
			})
		}
	}

	return projects, nil
}

// GetOAuthUrl 获取用户授权 Git 项目给蓝盾的 OAuth 授权地址
func (c *ApiClient) GetOAuthUrl(ctx context.Context, projectCode string) (string, error) {
	apiOperation := c.NewOperation(
		// 与 ListOAuthGitProjects 共用一个 API，但是获取的不同数据字段（当 Git 项目列表为空时，会提示用户进行授权）
		bkapi.OperationConfig{
			Name:   "v2_user_git_get_project",
			Method: "GET",
			Path:   "/v2/apigw-user/git/getProject",
		},
	).SetQueryParams(map[string]string{"projectId": projectCode})

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return "", err
	}

	return mapx.GetStr(result, "data.url"), nil
}

// ------------------------------------------ 蓝盾凭证管理 API ------------------------------------------

// CreateCredential 创建蓝盾凭证（目前只支持创建 用户名 - 密码类型的）
func (c *ApiClient) CreateCredential(
	ctx context.Context, projectCode, credentialID, credentialDesc, username, password string,
) error {
	body := map[string]string{
		"credentialType":   "USERNAME_PASSWORD",
		"credentialId":     credentialID,
		"credentialName":   credentialID,
		"credentialRemark": credentialDesc,
		"v1":               username,
		"v2":               password,
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_credential_create",
			Method: "POST",
			Path:   "v4/apigw-user/projects/{projectId}/credentials/credential",
		},
		bkapi.OptSetRequestPathParams(
			map[string]string{"projectId": projectCode},
		),
		bkapi.OptSetRequestBody(body),
	)

	return c.handleOperationWithoutResult(ctx, apiOperation)
}

// ------------------------------------------ 蓝盾流水线 API ------------------------------------------

// ListPipelines 获取流水线列表
// 注意：该结构提供的 Pipeline 中 Variables 字段为空，如有需要可通过 GetPipeline 获取
func (c *ApiClient) ListPipelines(
	ctx context.Context, projectCode, keyword string, page, pageSize int64,
) (int64, []Pipeline, error) {
	params := map[string]string{
		"page":     cast.ToString(page),
		"pageSize": cast.ToString(pageSize),
	}
	// 支持按照名称搜索
	if keyword != "" {
		params["pipelineName"] = keyword
	}
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_pipeline_paging_search_by_name",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/{projectId}/pipelines/paging_search_by_name",
		},
		bkapi.OptSetRequestPathParams(
			map[string]string{"projectId": projectCode},
		),
	).SetQueryParams(params)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return 0, nil, err
	}

	// 获取总流水线数量
	total := cast.ToInt64(mapx.Get(result, "data.count", 0))
	// 转换成 Pipeline 列表
	var pipelines []Pipeline
	for _, d := range mapx.GetList(result, "data.records") {
		p, ok := d.(map[string]any)
		if !ok {
			return 0, nil, errors.New("invalid pipeline data (not map[string]any type)")
		}
		// 组装数据
		pipelines = append(pipelines, Pipeline{
			ID:          mapx.GetStr(p, "pipelineId"),
			Name:        mapx.GetStr(p, "pipelineName"),
			Description: mapx.GetStr(p, "pipelineDesc"),
			Version:     cast.ToInt64(mapx.Get(p, "version", 0)),
			Creator:     mapx.GetStr(p, "creator"),
			Updater:     mapx.GetStr(p, "lastModifyUser"),
			CreatedAt:   time.UnixMilli(cast.ToInt64(mapx.Get(p, "createTime", 0))),
			UpdatedAt:   time.UnixMilli(cast.ToInt64(mapx.Get(p, "updateTime", 0))),
		})
	}

	return total, pipelines, nil
}

// GetPipeline 获取流水线详情
func (c *ApiClient) GetPipeline(ctx context.Context, projectCode, pipelineID string) (*Pipeline, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_pipeline_get",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/{projectId}/pipelines/pipeline",
		},
		bkapi.OptSetRequestPathParams(
			map[string]string{"projectId": projectCode},
		),
	).SetQueryParams(map[string]string{"pipelineId": pipelineID})

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	pipelineDetail := mapx.GetMap(result, "data")
	variables, err := parsePipelineVariables(pipelineDetail)
	if err != nil {
		return nil, errors.Wrap(err, "parse pipeline variables")
	}

	// 由于蓝盾流水线的详情（Retrieve）接口可用字段较少，缺失部分信息（如：更新人，创建/更新时间等）
	// 因此采用 list 接口获取基本信息 + retrieve 接口获取变量并组装到一起的方式实现 GetPipeline
	pipelineName := mapx.GetStr(pipelineDetail, "name")
	pipeline, err := c.getPipelineByName(ctx, projectCode, pipelineName)
	if err != nil {
		return nil, err
	}
	pipeline.Variables = variables

	return pipeline, nil
}

// getPipelineByName 通过名称获取流水线（不含变量信息）
func (c *ApiClient) getPipelineByName(ctx context.Context, projectCode, pipelineName string) (*Pipeline, error) {
	_, pipelines, err := c.ListPipelines(ctx, projectCode, pipelineName, 1, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "list pipelines with name %s", pipelineName)
	}
	if len(pipelines) == 0 {
		return nil, errors.Errorf("pipeline with name %s in project %s not found", pipelineName, projectCode)
	}

	return &pipelines[0], nil
}

// CreatePipeline 创建蓝盾流水线，返回流水线 ID（p-[a-z0-9]{32}）
func (c *ApiClient) CreatePipeline(
	ctx context.Context,
	projectCode, name, description string,
	stages []map[string]any,
) (string, error) {
	body := map[string]any{
		"name":            name,
		"desc":            description,
		"stages":          stages,
		"pipelineCreator": c.user.ID,
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_pipeline_create",
			Method: "POST",
			Path:   "/v4/apigw-user/projects/{projectId}/pipelines/pipeline",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
		bkapi.OptSetRequestBody(body),
	)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return "", err
	}

	// 从响应中提取流水线 ID
	pipelineID := mapx.GetStr(result, "data.id")
	return pipelineID, nil
}

// UpdatePipeline 更新蓝盾流水线
func (c *ApiClient) UpdatePipeline(
	ctx context.Context,
	projectCode, pipelineID, name, description string,
	stages []map[string]any,
) error {
	if projectCode == "" {
		return errors.New("projectCode is required to update bkci pipeline")
	}
	if pipelineID == "" {
		return errors.New("pipelineID is required to update bkci pipeline")
	}
	if name == "" {
		return errors.New("pipeline name is required to update bkci pipeline")
	}
	if len(stages) == 0 {
		return errors.New("pipeline stages are required to update bkci pipeline")
	}

	// body 只承载流水线 Model（name/desc/stages），pipelineId 通过 QueryParam 传递
	body := map[string]any{
		"name":   name,
		"desc":   description,
		"stages": stages,
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_pipeline_edit",
			Method: "PUT",
			Path:   "/v4/apigw-user/projects/{projectId}/pipelines/pipeline",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
		bkapi.OptSetRequestBody(body),
	).SetQueryParams(map[string]string{
		"pipelineId": pipelineID,
	})

	if err := c.handleOperationWithoutResult(ctx, apiOperation); err != nil {
		return errors.Wrapf(err, "update bkci pipeline %s in project %s", pipelineID, projectCode)
	}
	return nil
}

// CreatePipelineBuild 创建蓝盾流水线构建，返回构建引用信息（包含构建 ID: b-[a-z0-9]{32}）
func (c *ApiClient) CreatePipelineBuild(
	ctx context.Context, projectCode, pipelineID string, pipelineParams map[string]string,
) (*PipelineBuildReference, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_build_start",
			Method: "POST",
			Path:   "/v4/apigw-user/projects/{projectId}/build_start",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
		bkapi.OptSetRequestQueryParams(map[string]string{
			"pipelineId": pipelineID,
		}),
		bkapi.OptSetRequestBody(pipelineParams),
	)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	// 从响应中提取构建信息
	return &PipelineBuildReference{
		ID:  mapx.GetStr(result, "data.id"),
		Num: cast.ToString(mapx.Get(result, "data.num", "")),
	}, nil
}

// GetPipelineBuildState 获取蓝盾流水线构建状态
func (c *ApiClient) GetPipelineBuildState(
	ctx context.Context, projectCode, pipelineID, buildID string,
) (*PipelineBuildState, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_build_status",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/{projectId}/build_status",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
	).SetQueryParams(map[string]string{
		"pipelineId": pipelineID,
		"buildId":    buildID,
	})

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	// 提取 variables 字段
	variables := map[string]string{}
	if vars := mapx.GetMap(result, "data.variables"); vars != nil {
		for k, v := range vars {
			variables[k] = cast.ToString(v)
		}
	}

	// 组装流水线构建状态
	return &PipelineBuildState{
		PipelineID:  pipelineID,
		BuildID:     buildID,
		BuildNum:    cast.ToString(mapx.Get(result, "data.buildNum", "")),
		UserID:      mapx.GetStr(result, "data.userId"),
		Status:      mapx.GetStr(result, "data.status"),
		StartTime:   cast.ToInt64(mapx.Get(result, "data.startTime", 0)),
		EndTime:     cast.ToInt64(mapx.Get(result, "data.endTime", 0)),
		TotalTime:   cast.ToInt64(mapx.Get(result, "data.totalTime", 0)),
		ExecuteTime: cast.ToInt64(mapx.Get(result, "data.executeTime", 0)),
		Variables:   variables,
	}, nil
}

// ListPipelineRepoRefProperties 获取流水线启动表单中的代码分支/Tag字段列表
func (c *ApiClient) ListPipelineRepoRefProperties(
	ctx context.Context, projectCode, pipelineID string,
) ([]RepoRefProperty, error) {
	startInfo, err := c.getPipelineStartupInfo(ctx, projectCode, pipelineID)
	if err != nil {
		return nil, err
	}

	return parseRepoRefProperties(startInfo)
}

// ListPipelineRepoRefOptions 获取某个代码分支/Tag字段的可选项
//
// BKCI 的 build_manual_startup_options 接口不能仅凭 propertyID 查询选项，
// 它要求请求体中携带 startup_info 里对应字段的原始 property 定义。
// 因此这里需要先查询一次 build_manual_startup_info，再将匹配到的 property
// 原文透传给 options 接口。
//
// 当前没有采用“由前端回传 property 再直接查 options”的方案，是为了避免将
// BKCI 上游字段结构直接暴露到本服务 API 契约中，减少前后端对上游细节的耦合。
func (c *ApiClient) ListPipelineRepoRefOptions(
	ctx context.Context, projectCode, pipelineID, propertyID, search string,
) ([]PipelineVariableOption, error) {
	property, err := c.getPipelineRepoRefProperty(ctx, projectCode, pipelineID, propertyID)
	if err != nil {
		return nil, err
	}

	return c.listPipelineRepoRefOptionsByProperty(ctx, projectCode, pipelineID, property, search)
}

// getPipelineStartupInfo 获取 BKCI 流水线启动表单原始数据。
// 返回原始 data map，供上层继续解析 RepoRef 字段或透传 property 给 options 接口。
func (c *ApiClient) getPipelineStartupInfo(
	ctx context.Context, projectCode, pipelineID string,
) (map[string]any, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_build_startInfo",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/{projectId}/build_manual_startup_info",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
	).SetQueryParams(map[string]string{
		"pipelineId": pipelineID,
	})

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	return mapx.GetMap(result, "data"), nil
}

// getPipelineRepoRefProperty 从 BKCI 流水线启动表单中获取指定 ID 的 RepoRef 属性定义。
func (c *ApiClient) getPipelineRepoRefProperty(
	ctx context.Context, projectCode, pipelineID, propertyID string,
) (map[string]any, error) {
	startInfo, err := c.getPipelineStartupInfo(ctx, projectCode, pipelineID)
	if err != nil {
		return nil, err
	}

	// 仅筛选 type 为 git_ref、svn_tag 或 repo_ref 的属性，返回上游原始 property 定义，
	// 供后续查询分支/Tag 动态可选项时透传给 BKCI 的 startup_options 接口。
	for _, item := range mapx.GetList(startInfo, "properties") {
		property, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("invalid repo ref property (not map[string]any type): %v", item)
		}
		propertyType := mapx.GetStr(property, "type")
		if !isRepoRefPropertyType(propertyType) {
			continue
		}
		if mapx.GetStr(property, "id") == propertyID {
			return property, nil
		}
	}

	return nil, errors.Errorf("repo ref property %s not found", propertyID)
}

// listPipelineRepoRefOptionsByProperty 使用 startup_info 中的原始 property 定义查询动态选项。
// BKCI 的 startup_options 接口依赖完整 property 请求体，因此这里直接透传上游原文。
func (c *ApiClient) listPipelineRepoRefOptionsByProperty(
	ctx context.Context, projectCode, pipelineID string, property map[string]any, search string,
) ([]PipelineVariableOption, error) {
	queryParams := map[string]string{
		"pipelineId": pipelineID,
	}
	if search != "" {
		queryParams["search"] = search
	}
	optionsOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_build_startOptions",
			Method: "POST",
			Path:   "/v4/apigw-user/projects/{projectId}/build_manual_startup_options",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
		bkapi.OptSetRequestQueryParams(queryParams),
		bkapi.OptSetRequestBody(property),
	)

	optionsResult, err := c.handleOperation(ctx, optionsOperation)
	if err != nil {
		return nil, err
	}

	options := make([]PipelineVariableOption, 0, len(mapx.GetList(optionsResult, "data")))
	for _, item := range mapx.GetList(optionsResult, "data") {
		option, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("invalid repo ref option (not map[string]any type): %v", item)
		}
		options = append(options, PipelineVariableOption{
			Key:   mapx.GetStr(option, "key"),
			Value: mapx.GetStr(option, "value"),
		})
	}

	return options, nil
}

// ------------------------------------------ 蓝盾代码库管理 API ------------------------------------------

// ListRepository 获取蓝盾仓库列表
// repoType 可选值："", CODE_SVN, CODE_GIT, CODE_GITLAB, GITHUB, CODE_TGIT, CODE_P4
func (c *ApiClient) ListRepository(
	ctx context.Context, projectCode, repoType string, page, pageSize int64,
) (int64, []Repository, error) {
	params := map[string]string{
		"page":     cast.ToString(page),
		"pageSize": cast.ToString(pageSize),
	}
	if repoType != "" {
		params["repositoryType"] = repoType
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_repository_list",
			Method: "GET",
			Path:   "/v4/apigw-user/repositories/projects/{projectId}/repository_info_list",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
	).SetQueryParams(params)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return 0, nil, err
	}

	// 获取总仓库数量
	total := cast.ToInt64(mapx.Get(result, "data.count", 0))
	// 转换成 Repository 列表
	var repositories []Repository
	for _, d := range mapx.GetList(result, "data.records") {
		r, ok := d.(map[string]any)
		if !ok {
			return 0, nil, errors.New("invalid repository data (not map[string]any type)")
		}
		// 组装数据
		repositories = append(repositories, Repository{
			ID:        mapx.GetStr(r, "repositoryHashId"),
			Alias:     mapx.GetStr(r, "aliasName"),
			Url:       mapx.GetStr(r, "url"),
			Type:      mapx.GetStr(r, "type"),
			UpdatedAt: time.Unix(cast.ToInt64(mapx.Get(r, "updatedTime", 0)), 0),
		})
	}

	return total, repositories, nil
}

// GetRepository 获取蓝盾代码库详情
func (c *ApiClient) GetRepository(
	ctx context.Context, projectCode, repoHashID string,
) (*Repository, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_repository_get",
			Method: "GET",
			Path:   "/v4/apigw-user/repositories/projects/{projectId}/repository/{repositoryId}",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId":    projectCode,
			"repositoryId": repoHashID,
		}),
	)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	// 解析代码库数据
	return &Repository{
		ID:        mapx.GetStr(result, "data.repositoryHashId"),
		Alias:     mapx.GetStr(result, "data.aliasName"),
		Url:       mapx.GetStr(result, "data.url"),
		Type:      mapx.GetStr(result, "data.type"),
		UpdatedAt: time.Unix(cast.ToInt64(mapx.Get(result, "data.updatedTime", 0)), 0),
	}, nil
}

// CreateRepository 创建蓝盾代码库，返回代码库 Hash ID（目前只支持 codeGit + OAuth）
func (c *ApiClient) CreateRepository(
	ctx context.Context, projectCode, repoURL, repoAlias string,
) (string, error) {
	body := map[string]any{
		"@type":       "codeGit",
		"aliasName":   repoAlias,
		"url":         repoURL,
		"authType":    "OAUTH",
		"projectName": repoAlias,
		"userName":    c.user.ID,
		// OAUTH 是不需要指定凭证的，但是蓝盾 API 有做检查
		"credentialId": "",
	}

	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_repository_create",
			Method: "POST",
			Path:   "/v4/apigw-user/repositories/projects/{projectId}/repository",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
		bkapi.OptSetRequestBody(body),
	)

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return "", err
	}

	// 从响应中提取代码库 ID
	repoID := mapx.GetStr(result, "data.hashId")
	return repoID, nil
}

// ------------------------------------------ 蓝盾构建日志 API ------------------------------------------

// GetInitBuildLog 获取构建日志初始内容，首屏日志拉取，具体返回日志数量由bkci决定，请求不能传递需要拉取数量
// 根据返回 hasMore 和 finished 确定是否根据getMore接口再次拉取
func (c *ApiClient) GetInitBuildLog(
	ctx context.Context, projectCode, pipelineID, buildID string,
) (*BuildLog, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_log_init",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/{projectId}/logs/init_logs",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
	).SetQueryParams(map[string]string{
		"pipelineId": pipelineID,
		"buildId":    buildID,
	})

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	buildLog, err := parseBuildLog(ctx, result)
	if err != nil {
		return nil, errors.Wrap(err, "parse init build log")
	}
	return buildLog, nil
}

// GetMoreBuildLogs 获取指定行号之后的构建日志（增量拉取，用于 SSE 续传）。
// batchSize 由调用方控制查询窗口，客户端内部会映射为 BKCI 的 end=start+batchSize 参数
func (c *ApiClient) GetMoreBuildLogs(
	ctx context.Context, projectCode, pipelineID, buildID string, start, batchSize int64,
) (*BuildLog, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_log_more",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/{projectId}/logs/more_logs",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
	).SetQueryParams(map[string]string{
		"pipelineId": pipelineID,
		"buildId":    buildID,
		"start":      cast.ToString(start),
		"end":        cast.ToString(start + batchSize),
		// 是否正序输出
		"fromStart": "true",
	})

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	buildLog, err := parseBuildLog(ctx, result)
	if err != nil {
		return nil, errors.Wrap(err, "parse incremental build log")
	}
	return buildLog, nil
}

// DownloadBuildLogs 下载构建日志原始流（适用于下载场景）
func (c *ApiClient) DownloadBuildLogs(
	ctx context.Context, projectCode, pipelineID, buildID string,
) (io.ReadCloser, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v4_user_log_download",
			Method: "GET",
			Path:   "/v4/apigw-user/projects/{projectId}/logs/download_logs",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"projectId": projectCode,
		}),
		bkapi.OptSetRequestHeader("Content-Type", "application/json"),
	).SetQueryParams(map[string]string{
		"pipelineId": pipelineID,
		"buildId":    buildID,
	})

	return c.handleStreamOperation(ctx, apiOperation)
}

// parseBuildLog 从 BKCI 响应中解析构建日志
func parseBuildLog(ctx context.Context, result map[string]any) (*BuildLog, error) {
	rawData, exists := result["data"]
	if !exists || rawData == nil {
		return &BuildLog{}, nil
	}

	data, ok := rawData.(map[string]any)
	if !ok {
		return nil, errors.Errorf("bkci build log payload data invalid type: %T", rawData)
	}

	var buildLog BuildLog
	if err := mapstructure.Decode(data, &buildLog); err != nil {
		log.Errorf(ctx, "parse build log failed: %v", err)
		return nil, errors.Wrap(err, "decode build log payload")
	}

	switch buildLog.Status {
	case buildLogStatusSucceed, buildLogStatusEmpty:
		return &buildLog, nil
	case buildLogStatusExpired:
		return nil, wrapBuildLogStatusError(BuildLogExpired, buildLog.Message)
	case buildLogStatusCleaned:
		return nil, wrapBuildLogStatusError(BuildLogCleaned, buildLog.Message)
	case buildLogStatusFail:
		return nil, wrapBuildLogStatusError(BuildLogQueryFailed, buildLog.Message)
	default:
		return nil, errors.Errorf(
			"bkci build log returned unknown status: %d, message: %s",
			buildLog.Status, buildLog.Message,
		)
	}
}

func wrapBuildLogStatusError(base error, message string) error {
	if message == "" {
		return base
	}
	return errors.Wrap(base, message)
}

// handleOperation 发起请求并检查结果，返回响应体 & 错误
func (c *ApiClient) handleOperation(
	ctx context.Context, apiOperation define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	opName := apiOperation.FullName()
	defer metrics.ReportClientRequestMetric("bkci", apiOperation.FullName(), started, &err)

	ctx, span := apm.StartClientSpan(ctx, "bkci", opName)
	resp, err := apiOperation.SetContext(ctx).SetResult(&result).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// HTTP 状态码检查
	// 404 错误 -> 蓝盾资源不存在
	if resp.StatusCode == http.StatusNotFound {
		return nil, ObjectNotFound
	}
	// 根据返回码判断是否失败
	// 蓝鲸网关错误码字段是 Code，蓝盾错误码字段是 Status
	code, status := cast.ToInt(result["code"]), cast.ToInt(result["status"])
	// 针对特定的蓝盾错误码，抛出特定的错误
	switch status {
	case projectNameExistStatus:
		return nil, ProjectAlreadyExist
	case repoAliasExistStatus:
		return nil, RepoAlreadyExist
	}

	// 其他 HTTP 错误 -> 蓝盾 API 错误（从 result 中多字段兜底提取 message）
	if !httpresp.IsSuccess(resp) {
		return nil, newHTTPError(opName, resp.StatusCode, result)
	}

	// 业务错误（HTTP 200 但 code/status 非零）-> 蓝盾 API 错误
	if code != 0 || status != 0 {
		return nil, newBusinessError(opName, code, status, result)
	}
	return result, nil
}

// newHTTPError 构造 HTTP 非 2xx 场景的错误
func newHTTPError(opName string, statusCode int, result map[string]any) error {
	return errors.Errorf(
		"call bkci api %s failed, status code: %d, message: %s",
		opName, statusCode, extractErrorMessage(result),
	)
}

// newBusinessError 构造 HTTP 200 但业务错误码非零的错误
func newBusinessError(opName string, code, status int, result map[string]any) error {
	return errors.Errorf(
		"call bkci api %s failed, code: %d, status: %d, message: %s",
		opName, code, status, extractErrorMessage(result),
	)
}

// handleStreamOperation 发起请求并返回原始响应流，适用于文件下载场景
func (c *ApiClient) handleStreamOperation(
	ctx context.Context,
	apiOperation define.Operation,
) (body io.ReadCloser, err error) {
	started := time.Now()
	opName := apiOperation.FullName()
	defer metrics.ReportClientRequestMetric("bkci", apiOperation.FullName(), started, &err)

	ctx, span := apm.StartClientSpan(ctx, "bkci", opName)
	resp, err := apiOperation.SetContext(ctx).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, err
	}

	// HTTP 状态码检查
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, ObjectNotFound
	}
	if !httpresp.IsSuccess(resp) {
		errMsg, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, errors.Errorf(
			"call bkci api %s failed, status code: %d, err: %s", opName, resp.StatusCode, errMsg,
		)
	}
	return resp.Body, nil
}

// handleOperationWithoutResult 发起请求，忽略返回数据但仍处理错误
// 本方法是 handleOperation 的简化版本，用于不需要返回数据的场景（如：Delete 操作）
func (c *ApiClient) handleOperationWithoutResult(ctx context.Context, apiOperation define.Operation) error {
	_, err := c.handleOperation(ctx, apiOperation)
	return err
}
