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

// Package client provides API call for bkms-cli
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
)

const (
	// DefaultListAppConfigFileVersionsPageSize 获取应用配置文件版本时的每页默认数量。
	DefaultListAppConfigFileVersionsPageSize = 100
	// DefaultListAppInstancesPageSize 获取应用实例列表时的每页默认数量。
	DefaultListAppInstancesPageSize = 100
)

// 接口兼容性断言
var _ Client = new(SvcBasedClient)

// ErrTokenExpiredOrInvalid Token 过期或无效
var ErrTokenExpiredOrInvalid = errors.New("access token expired or invalid")

// SvcBasedClient 调用 bkms 服务的客户端
type SvcBasedClient struct {
	cli     *resty.Client
	authCli *resty.Client
}

// New 新建 SvcBasedClient 实例
func New() Client {
	// 使用连接池
	transport := &http.Transport{
		MaxIdleConns:        5,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	bizCli := resty.New().
		SetTransport(transport).
		SetBaseURL(config.G.BkmsBaseURL).
		SetTimeout(30 * time.Second).
		// 配置重试策略
		SetRetryCount(2).
		SetRetryWaitTime(5 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second).
		AddRetryCondition(
			func(response *resty.Response, err error) bool {
				if response == nil {
					return false
				}
				// Retry on 5xx status codes
				// fixme err 读取
				return response.StatusCode() >= http.StatusInternalServerError
			},
		).
		SetHeaders(map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", config.G.AccessToken),
		})

	// 用于 ValidateAccessToken / ExchangeBkTicketForToken 等登录前的鉴权操作
	authCli := resty.New().
		SetTransport(transport).
		SetBaseURL(config.G.BkmsBaseURL).
		SetTimeout(10 * time.Second)

	return &SvcBasedClient{cli: bizCli, authCli: authCli}
}

// ValidateAccessToken 通过 AccessToken 获取用户名信息
func (c *SvcBasedClient) ValidateAccessToken(accessToken string) (string, error) {
	resp, err := c.authCli.R().
		SetQueryParam("access_token", accessToken).
		Get("/user_token/validate")
	if err != nil {
		return "", errors.Wrap(err, "failed to connect bkms server, please check your network")
	}
	if resp.StatusCode() != http.StatusOK {
		return "", errors.Errorf("validate access token failed [%d]: %s",
			resp.StatusCode(), truncateBody(resp.Body()))
	}
	result := map[string]any{}
	if err = json.Unmarshal(resp.Body(), &result); err != nil {
		return "", errors.Errorf("validate access token failed: unexpected response format: %s",
			truncateBody(resp.Body()))
	}

	username := mapx.GetStr(result, "username")
	if username == "" {
		return "", ErrTokenExpiredOrInvalid
	}
	return username, nil
}

// ExchangeBkTicketForToken 使用 bk_ticket 兑换 access_token
func (c *SvcBasedClient) ExchangeBkTicketForToken(username, bkTicket string) (string, error) {
	resp, err := c.authCli.R().
		SetCookie(&http.Cookie{Name: "bk_uid", Value: username}).
		SetCookie(&http.Cookie{Name: "bk_ticket", Value: bkTicket}).
		Get("/user_token/token")
	if err != nil {
		return "", errors.Wrap(err, "failed to connect bkms server, please check your network")
	}
	if resp.StatusCode() != http.StatusOK {
		return "", errors.Errorf("exchange bk_ticket failed [%d]: %s",
			resp.StatusCode(), truncateBody(resp.Body()))
	}
	result := map[string]any{}
	if err = json.Unmarshal(resp.Body(), &result); err != nil {
		return "", errors.Errorf("exchange bk_ticket failed: unexpected response format: %s",
			truncateBody(resp.Body()))
	}

	token := mapx.GetStr(result, "access_token")
	if token == "" {
		return "", errors.New("access_token not found in response")
	}
	return token, nil
}

// DevModePublishPreflight 开发模式 Publish 预检，获取 publish 所需的全部上下文信息
func (c *SvcBasedClient) DevModePublishPreflight(
	ctx context.Context,
	appID, envName string,
	instanceIDs []string,
	publishAll bool,
) (*DevModePreflightData, error) {
	var respData DevModePreflightRespData
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/devmode/%s/envs/%s/preflight",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	body := map[string]any{
		"instanceIDs": instanceIDs,
		"publishAll":  publishAll,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(path)
	if err != nil {
		return nil, errors.Wrap(err, "publish preflight request failed")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf(
			"publish preflight failed: [%d] -> %s",
			resp.StatusCode(),
			truncateBody(resp.Body()),
		)
	}
	if respData.Data == nil {
		return nil, errors.New("devmode publish preflight returned empty data")
	}

	return respData.Data, nil
}

// ListWorkspaces 获取工作空间列表
func (c *SvcBasedClient) ListWorkspaces(ctx context.Context, keyword string) ([]Workspace, error) {
	var respData ListWorkspacesRespData
	path := "/bkms/v1/bkms-server/workspaces"

	resp, err := c.cli.R().SetContext(ctx).SetQueryParam("keyword", keyword).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list workspaces failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data, nil
}

// GetWorkspace 获取工作空间详情
func (c *SvcBasedClient) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	var respData GetWorkspaceRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s", url.PathEscape(id))

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("get workspace failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &respData.Data, nil
}

// ListEnvs 获取环境列表
func (c *SvcBasedClient) ListEnvs(ctx context.Context, workspaceID string) ([]Env, error) {
	var respData ListEnvsRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/envs", url.PathEscape(workspaceID))

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list envs failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data, nil
}

// ListAppEnvs 获取应用可用环境，包含标准环境和该应用专属的特性环境。
func (c *SvcBasedClient) ListAppEnvs(ctx context.Context, appID string) ([]Env, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs", appID)

	var respData ListEnvsRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list app envs failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data, nil
}

// GetEnv 获取环境详情
func (c *SvcBasedClient) GetEnv(ctx context.Context, envID string) (*Env, error) {
	var respData GetEnvRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/envs/%s", url.PathEscape(envID))

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("get env failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &respData.Data, nil
}

// CreateEnv 创建环境
func (c *SvcBasedClient) CreateEnv(ctx context.Context, workspaceID string, body CreateEnvBody) (string, error) {
	var respData CreateEnvRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/envs", url.PathEscape(workspaceID))

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(path)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return "", errors.Errorf("create env failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.ID, nil
}

// UpdateEnvBasicInfo 更新环境基本信息
func (c *SvcBasedClient) UpdateEnvBasicInfo(ctx context.Context, envID string, body UpdateEnvBasicInfoBody) error {
	path := fmt.Sprintf("/bkms/v1/bkms-server/envs/%s/basic-info", url.PathEscape(envID))

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("update env basic info failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// DeleteEnv 删除环境
func (c *SvcBasedClient) DeleteEnv(ctx context.Context, envID string) error {
	path := fmt.Sprintf("/bkms/v1/bkms-server/envs/%s", url.PathEscape(envID))

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("delete env failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// CreateFeatureEnv 创建特性环境，返回服务端生成的环境名称和独立命名空间。
func (c *SvcBasedClient) CreateFeatureEnv(ctx context.Context, appID string, body CreateFeatureEnvBody) (*Env, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/feat-envs", appID)

	var respData GetEnvRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, errors.Errorf("create feature env failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &respData.Data, nil
}

// ResolveApp 通过 ID 或 Name 解析应用，返回 appID
func (c *SvcBasedClient) ResolveApp(ctx context.Context, workspaceID, input string) (string, error) {
	var respData struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/workspaces/%s/apps/resolve/%s",
		url.PathEscape(workspaceID),
		url.PathEscape(input),
	)

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return "", errors.Wrap(err, "resolve app")
	}
	if resp.StatusCode() == http.StatusNotFound {
		return "", errors.Errorf("app %s not found in workspace %s", input, workspaceID)
	}
	if resp.StatusCode() != http.StatusOK {
		return "", errors.Errorf("resolve app failed: [%d] -> %s", resp.StatusCode(), truncateBody(resp.Body()))
	}

	return respData.Data.ID, nil
}

// ListApps 获取应用列表
func (c *SvcBasedClient) ListApps(ctx context.Context, workspaceID string) ([]AppMinimal, error) {
	var respData ListAppsRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/apps", url.PathEscape(workspaceID))

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list apps failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data, nil
}

// GetAppMinimal 获取应用，过滤 ListApps 结果。
// 假如：直接从 GetApp 接口获取，resp body 有太多字段，导致 cli 场景使用不便，也用不到这些字段。
func (c *SvcBasedClient) GetAppMinimal(ctx context.Context, workspaceID, appID string) (*AppMinimal, error) {
	apps, err := c.ListApps(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	for i, app := range apps {
		if app.ID == appID {
			return &apps[i], nil
		}
	}

	return nil, errors.Errorf("app %s not found", appID)
}

// GetApp 获取应用完整定义（含 BuildConfig / AppModelSpec）
func (c *SvcBasedClient) GetApp(ctx context.Context, appID string) (*AppFull, error) {
	var respData GetAppFullRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s", url.PathEscape(appID))

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("get app failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &respData.Data, nil
}

// DeleteApp 删除应用
func (c *SvcBasedClient) DeleteApp(ctx context.Context, appID string) error {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s", url.PathEscape(appID))

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("delete app failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// UpdateAppDisplayName 更新应用显示名
func (c *SvcBasedClient) UpdateAppDisplayName(ctx context.Context, appID, displayName string) error {
	body := UpdateAppDisplayNameBody{DisplayName: displayName}
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/display-name", url.PathEscape(appID))

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("update app display name failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// UpdateAppBuildConfig 更新应用构建配置
func (c *SvcBasedClient) UpdateAppBuildConfig(ctx context.Context, appID string, body AppBuildConfigUpdateBody) error {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/build-configs", url.PathEscape(appID))

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("update app build config failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// GetAppIDAutoSuffix 获取应用 ID 自动后缀（后端生成）
func (c *SvcBasedClient) GetAppIDAutoSuffix(ctx context.Context) (string, error) {
	var respData GetAppIDAutoSuffixRespData
	path := "/bkms/v1/bkms-server/apps/auto-id-suffix"

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", errors.Errorf("get app id auto suffix failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Suffix, nil
}

// CreateApp 创建应用
func (c *SvcBasedClient) CreateApp(ctx context.Context, workspaceID string, body any) (*AppMinimal, error) {
	path := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/apps", url.PathEscape(workspaceID))

	var respData CreateAppRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, errors.Errorf("create app failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &respData.Data, nil
}

// CreateAppBuild 执行应用构建
func (c *SvcBasedClient) CreateAppBuild(ctx context.Context, appID string, opts BuildOptions) error {
	body := map[string]any{"branch": opts.Branch, "imageTag": opts.ImageTag}
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/builds", url.PathEscape(appID))

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("create app build failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// ListAppImages 获取应用镜像列表（因 cli 场景，默认只提供前 10 条）
func (c *SvcBasedClient) ListAppImages(ctx context.Context, appID, keyword string) ([]Image, error) {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/images", url.PathEscape(appID))
	queryParams := map[string]string{
		"keyword":  keyword,
		"page":     "1",
		"pageSize": "10",
	}

	var respData ListAppImagesResp
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list app images failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// ListAppConfigFiles 获取应用配置文件列表
func (c *SvcBasedClient) ListAppConfigFiles(ctx context.Context, appID, envName string) ([]AppConfigFile, error) {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-files", url.PathEscape(appID))

	var respData ListAppConfigFilesRespData
	req := c.cli.R().SetContext(ctx).SetResult(&respData)
	if envName != "" {
		req.SetQueryParam("envName", envName)
	}

	resp, err := req.Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list app config files failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Items, nil
}

// GetAppConfigFileDetails 获取应用配置文件详情
func (c *SvcBasedClient) GetAppConfigFileDetails(
	ctx context.Context,
	appID, fileID string,
) (*AppConfigFileDetails, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/app-config-files/%s/details",
		url.PathEscape(appID),
		url.PathEscape(fileID),
	)

	var details AppConfigFileDetails
	resp, err := c.cli.R().SetContext(ctx).SetResult(&details).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("get app config file details failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &details, nil
}

// ListAppConfigFileVersions 获取应用配置文件历史版本列表
func (c *SvcBasedClient) ListAppConfigFileVersions(
	ctx context.Context,
	appID string,
	opts ListAppConfigFileVersionsOptions,
) (*PaginatedAppConfigFileVersions, error) {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-file/versions", url.PathEscape(appID))

	page := opts.Page
	if page <= 0 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = DefaultListAppConfigFileVersionsPageSize
	}

	var respData ListAppConfigFileVersionsRespData
	req := c.cli.R().SetContext(ctx).SetResult(&respData).SetQueryParams(map[string]string{
		"page":     strconv.Itoa(page),
		"pageSize": strconv.Itoa(pageSize),
	})
	if opts.AppConfigFileID != "" {
		req.SetQueryParam("appConfigFileID", opts.AppConfigFileID)
	}
	if opts.EnvName != "" {
		req.SetQueryParam("envName", opts.EnvName)
	}
	if opts.Name != "" {
		req.SetQueryParam("name", opts.Name)
	}
	if opts.Version != nil {
		req.SetQueryParam("version", strconv.FormatInt(*opts.Version, 10))
	}
	if opts.Creator != "" {
		req.SetQueryParam("creator", opts.Creator)
	}
	if opts.Description != "" {
		req.SetQueryParam("description", opts.Description)
	}

	resp, err := req.Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf(
			"list app config file versions failed: [%d] -> %s",
			resp.StatusCode(),
			resp.Body(),
		)
	}

	return &respData.Data, nil
}

// GetAppConfigFileVersion 获取应用配置文件某个历史版本详情
func (c *SvcBasedClient) GetAppConfigFileVersion(
	ctx context.Context,
	appID, versionID string,
) (*AppConfigFileVersion, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/app-config-file/versions/%s",
		url.PathEscape(appID),
		url.PathEscape(versionID),
	)

	var respData GetAppConfigFileVersionRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf(
			"get app config file version failed: [%d] -> %s",
			resp.StatusCode(),
			resp.Body(),
		)
	}

	return &respData.Data, nil
}

// DeleteAppConfigFileVersion 删除应用配置文件某个历史版本
func (c *SvcBasedClient) DeleteAppConfigFileVersion(
	ctx context.Context,
	appID, versionID string,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/app-config-file/versions/%s",
		url.PathEscape(appID),
		url.PathEscape(versionID),
	)

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf(
			"delete app config file version failed: [%d] -> %s",
			resp.StatusCode(),
			resp.Body(),
		)
	}

	return nil
}

// RollbackAppConfigFileVersion 回滚到应用配置文件某个历史版本
func (c *SvcBasedClient) RollbackAppConfigFileVersion(
	ctx context.Context,
	appID, versionID string,
	opts RollbackAppConfigFileVersionOptions,
) (*AppConfigFile, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/app-config-file/versions/%s/rollback",
		url.PathEscape(appID),
		url.PathEscape(versionID),
	)
	body := map[string]any{
		"currentVersion": opts.CurrentVersion,
		"description":    opts.Description,
	}

	var respData RollbackAppConfigFileVersionRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf(
			"rollback app config file version failed: [%d] -> %s",
			resp.StatusCode(),
			resp.Body(),
		)
	}

	return &respData.Data, nil
}

// UpdateAppConfigFileContent 更新应用配置文件 Content
func (c *SvcBasedClient) UpdateAppConfigFileContent(
	ctx context.Context,
	appID, fileID string,
	opts AppConfigFileContentOptions,
) (*AppConfigFileContentUpdateResult, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/app-config-files/%s/content",
		url.PathEscape(appID),
		url.PathEscape(fileID),
	)
	body := map[string]any{
		"content":        opts.Content,
		"description":    opts.Description,
		"currentVersion": opts.CurrentVersion,
	}

	var result AppConfigFileContentUpdateResult
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&result).Put(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("update app config file content failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &result, nil
}

// UpdateAppConfigFileOverlayContent 更新应用配置文件 OverlayContent
func (c *SvcBasedClient) UpdateAppConfigFileOverlayContent(
	ctx context.Context,
	appID, fileID string,
	opts AppConfigFileContentOptions,
) (*AppConfigFileContentUpdateResult, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/app-config-files/%s/overlay-content",
		url.PathEscape(appID),
		url.PathEscape(fileID),
	)
	body := map[string]any{
		"overlayContent": opts.Content,
		"description":    opts.Description,
		"currentVersion": opts.CurrentVersion,
	}

	var result AppConfigFileContentUpdateResult
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&result).Put(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf(
			"update app config file overlay content failed: [%d] -> %s",
			resp.StatusCode(),
			resp.Body(),
		)
	}

	return &result, nil
}

// ListBuildRecords 获取应用构建记录（因 cli 场景，默认只提供前 10 条）
func (c *SvcBasedClient) ListBuildRecords(ctx context.Context, appID, keyword string) ([]BuildRecord, error) {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/builds", url.PathEscape(appID))
	queryParams := map[string]string{
		"keyword":  keyword,
		"page":     "1",
		"pageSize": "10",
	}

	var respData ListBuildRecordsRespData
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list build records failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// CreateAppHelmDeploy 执行 Helm 应用部署
func (c *SvcBasedClient) CreateAppHelmDeploy(
	ctx context.Context,
	appID, envName string,
	opts HelmDeployOptions,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/helm-deploys",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	body := map[string]any{
		"imageTag":        opts.ImageTag,
		"chartVersion":    opts.ChartVersion,
		"valuesFileID":    opts.ValuesFileID,
		"trafficLaneName": opts.TrafficLaneName,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("create app helm deploy failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// ListHelmDeployRecords 获取应用 Helm 部署记录（因 cli 场景，默认只提供前 10 条）
func (c *SvcBasedClient) ListHelmDeployRecords(
	ctx context.Context, appID, envName, trafficLaneName, keyword string,
) ([]HelmDeployRecord, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/helm-deploys",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	queryParams := map[string]string{
		"trafficLaneName": trafficLaneName,
		"keyword":         keyword,
		"page":            "1",
		"pageSize":        "10",
	}

	var respData ListHelmDeployRecordsRespData
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list helm deploy records failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// CreateAppTrpcDeploy 执行 Trpc 应用部署
func (c *SvcBasedClient) CreateAppTrpcDeploy(
	ctx context.Context,
	appID, envName string,
	opts AppModelDeployOptions,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/trpc-deploys",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	body := map[string]any{
		"imageTag":        opts.ImageTag,
		"replicas":        opts.Replicas,
		"trafficLaneName": opts.TrafficLaneName,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("create app trpc deploy failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// ListTrpcDeployRecords 获取最近的应用 Trpc 部署记录
func (c *SvcBasedClient) ListTrpcDeployRecords(
	ctx context.Context, appID, envName, keyword, trafficLaneName string,
) ([]AppModelDeployRecord, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/trpc-deploys",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	queryParams := map[string]string{
		"keyword":         keyword,
		"trafficLaneName": trafficLaneName,
		"page":            "1",
		"pageSize":        "10",
	}

	var respData AppModelDeployRecordsResp
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list trpc deploy records failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// DeleteHelmDeploy 删除 Helm 部署
func (c *SvcBasedClient) DeleteHelmDeploy(ctx context.Context, appID, envName, deployID string) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/helm-deploys/%s",
		url.PathEscape(appID),
		url.PathEscape(envName),
		url.PathEscape(deployID),
	)

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("delete helm deploy failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// GrayscaleUpdateInstance 灰度更新 Trpc 实例
func (c *SvcBasedClient) GrayscaleUpdateInstance(
	ctx context.Context, appID, envName, imageTag string, instanceIDs []string, updateStrategy string,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/instances",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	body := map[string]any{
		// fixme 泳道支持
		"imageTag":       imageTag,
		"instanceIDs":    instanceIDs,
		"updateStrategy": updateStrategy,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("update trpc instance failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// BatchUpdateInstance 全量更新 Trpc 实例
func (c *SvcBasedClient) BatchUpdateInstance(
	ctx context.Context, appID, envName, imageTag, updateStrategy string,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/instances",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	body := map[string]any{
		"imageTag":       imageTag,
		"updateStrategy": updateStrategy,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("update trpc instance failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// ListAppInstances 获取应用实例列表
func (c *SvcBasedClient) ListAppInstances(
	ctx context.Context,
	appID, envName string,
	opts ListAppInstancesOptions,
) (*PaginatedInstances, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/instances",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)

	page := opts.Page
	if page <= 0 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = DefaultListAppInstancesPageSize
	}

	var respData ListAppInstancesRespData
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(map[string]string{
		"page":     strconv.Itoa(page),
		"pageSize": strconv.Itoa(pageSize),
	}).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list app instances failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &respData.Data, nil
}

// UpdateInstancePolaris 更新实例北极星权重/隔离状态
func (c *SvcBasedClient) UpdateInstancePolaris(
	ctx context.Context, appID, envName string, opts UpdateInstancePolarisOptions,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/instances/operations/polaris",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).Put(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("update instance polaris failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// BatchDeleteInstances 批量删除实例
func (c *SvcBasedClient) BatchDeleteInstances(
	ctx context.Context, appID, envName string, opts BatchDeleteInstancesOptions,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/instances/operations/batch_delete",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).Post(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("batch delete instances failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// ListTrpcAdminCmds 查询 Trpc 管理命令列表
func (c *SvcBasedClient) ListTrpcAdminCmds(
	ctx context.Context, appID, envName string, instanceIDs []string,
) ([]string, error) {
	var respData ListTrpcAdminCmdsRespData
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/instances/admin-cmds",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)

	req := c.cli.R().SetContext(ctx).SetResult(&respData)
	for _, id := range instanceIDs {
		req.QueryParam.Add("instanceIDs", id)
	}
	resp, err := req.Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list trpc admin cmds failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// ExecuteTrpcAdminCmd 执行 Trpc 管理命令
func (c *SvcBasedClient) ExecuteTrpcAdminCmd(
	ctx context.Context, appID, envName string, opts ExecuteTrpcAdminCmdOptions,
) ([]AdminCmdResult, error) {
	var respData ExecuteAdminCmdRespData
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/instances/admin-cmds",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).SetResult(&respData).Post(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("execute trpc admin cmd failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// ExecuteTafAdminCmd 执行 Taf 管理命令
func (c *SvcBasedClient) ExecuteTafAdminCmd(
	ctx context.Context, appID, envName string, opts ExecuteTafAdminCmdOptions,
) ([]AdminCmdResult, error) {
	var respData ExecuteAdminCmdRespData
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/instances/taf-admin-cmds",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).SetResult(&respData).Post(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("execute taf admin cmd failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// CreateAppTafDeploy 执行 TAF 应用部署
func (c *SvcBasedClient) CreateAppTafDeploy(
	ctx context.Context, appID, envName string, opts AppModelDeployOptions,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/taf-deploys",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	body := map[string]any{
		"imageTag":        opts.ImageTag,
		"replicas":        opts.Replicas,
		"trafficLaneName": opts.TrafficLaneName,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("create app taf deploy failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// ListTafDeployRecords 获取最近的应用 TAF 部署记录（因 cli 场景，默认只提供前 10 条）
func (c *SvcBasedClient) ListTafDeployRecords(
	ctx context.Context, appID, envName, keyword, trafficLaneName string,
) ([]AppModelDeployRecord, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/taf-deploys",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	queryParams := map[string]string{
		"keyword":         keyword,
		"trafficLaneName": trafficLaneName,
		"page":            "1",
		"pageSize":        "10",
	}

	var respData AppModelDeployRecordsResp
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list taf deploy records failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// DeleteTrpcDeploy 删除 Trpc 部署
func (c *SvcBasedClient) DeleteTrpcDeploy(ctx context.Context, appID, envName string) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/trpc-deploys",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("delete trpc deploy failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// PreCheckTrpcDeploy 部署前检查 Trpc 应用
func (c *SvcBasedClient) PreCheckTrpcDeploy(
	ctx context.Context,
	appID, envName string,
) (*DeployPrecheckResult, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/trpc-deploys/precheck",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	return c.preCheckDeploy(ctx, path, "precheck trpc deploy")
}

// PreCheckTafDeploy 部署前检查 TAF 应用
func (c *SvcBasedClient) PreCheckTafDeploy(
	ctx context.Context,
	appID, envName string,
) (*DeployPrecheckResult, error) {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/taf-deploys/precheck",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)
	return c.preCheckDeploy(ctx, path, "precheck taf deploy")
}

func (c *SvcBasedClient) preCheckDeploy(ctx context.Context, path, op string) (*DeployPrecheckResult, error) {
	var respData DeployPrecheckResult
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("%s failed: [%d] -> %s", op, resp.StatusCode(), resp.Body())
	}
	respData.Normalize()
	return &respData, nil
}

// DeleteTafDeploy 删除 TAF 部署
func (c *SvcBasedClient) DeleteTafDeploy(ctx context.Context, appID, envName string) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/taf-deploys",
		url.PathEscape(appID),
		url.PathEscape(envName),
	)

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("delete taf deploy failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// --- AppSpec method implementations ---

// GetAppDetail queries the full app detail to retrieve type and start command info.
func (c *SvcBasedClient) GetAppDetail(ctx context.Context, appID string) (*AppDetail, error) {
	var respData GetAppDetailRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s", url.PathEscape(appID))

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, errors.Wrapf(err, "get app detail %s", appID)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("get app detail %s failed: [%d] -> %s", appID, resp.StatusCode(), resp.Body())
	}
	if respData.Data == nil {
		return nil, errors.Errorf("app %s not found", appID)
	}

	return respData.Data, nil
}

// GetAppSpecDefaultSection 获取应用默认 section 配置。
func (c *SvcBasedClient) GetAppSpecDefaultSection(
	ctx context.Context,
	appID string,
	section AppSpecSectionName,
	result any,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/app-spec/default-%s",
		url.PathEscape(appID),
		url.PathEscape(string(section)),
	)
	return c.getAppSpecSection(ctx, path, fmt.Sprintf("get default %s", section), result)
}

// GetAppSpecEnvEffectiveSection 获取应用环境生效 section 配置。
func (c *SvcBasedClient) GetAppSpecEnvEffectiveSection(
	ctx context.Context,
	appID, envName string,
	section AppSpecSectionName,
	result any,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/app-spec/%s/effective",
		url.PathEscape(appID),
		url.PathEscape(envName),
		url.PathEscape(string(section)),
	)
	return c.getAppSpecSection(ctx, path, fmt.Sprintf("get env effective %s", section), result)
}

func (c *SvcBasedClient) getAppSpecSection(ctx context.Context, path, desc string, result any) error {
	type respWrapper struct {
		Data any `json:"data"`
	}
	wrapper := &respWrapper{Data: result}
	resp, err := c.cli.R().SetContext(ctx).SetResult(wrapper).Get(path)
	if err != nil {
		return errors.Wrap(err, desc)
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("%s failed: [%d] -> %s", desc, resp.StatusCode(), resp.Body())
	}
	return nil
}

// SetAppSpecDefaultSection sets the default section config for an application.
func (c *SvcBasedClient) SetAppSpecDefaultSection(
	ctx context.Context,
	appID string,
	section AppSpecSectionName,
	body any,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/app-spec/default-%s",
		url.PathEscape(appID),
		url.PathEscape(string(section)),
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return errors.Wrapf(err, "set default %s config", section)
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("set default %s config failed: [%d] -> %s", section, resp.StatusCode(), resp.Body())
	}

	return nil
}

// SetAppSpecEnvSection sets the environment-specific section override.
func (c *SvcBasedClient) SetAppSpecEnvSection(
	ctx context.Context,
	appID, envName string,
	section AppSpecSectionName,
	body any,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/app-spec/%s",
		url.PathEscape(appID),
		url.PathEscape(envName),
		url.PathEscape(string(section)),
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return errors.Wrapf(err, "set %s config for env %s", section, envName)
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf(
			"set %s config for env %s failed: [%d] -> %s",
			section,
			envName,
			resp.StatusCode(),
			resp.Body(),
		)
	}

	return nil
}

// DeleteAppSpecEnvSection deletes the environment-specific section override (resets to default).
func (c *SvcBasedClient) DeleteAppSpecEnvSection(
	ctx context.Context,
	appID, envName string,
	section AppSpecSectionName,
) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/envs/%s/app-spec/%s",
		url.PathEscape(appID),
		url.PathEscape(envName),
		url.PathEscape(string(section)),
	)

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return errors.Wrapf(err, "delete %s config for env %s", section, envName)
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf(
			"delete %s config for env %s failed: [%d] -> %s",
			section, envName, resp.StatusCode(), resp.Body(),
		)
	}

	return nil
}

// UpdateAppStartCommand updates the app start command via the appropriate API based on app type.
func (c *SvcBasedClient) UpdateAppStartCommand(ctx context.Context, appID, appType string, body any) error {
	var specPath string
	switch appType {
	case "trpc":
		specPath = "trpc-spec"
	case "taf":
		specPath = "taf-spec"
	default:
		return errors.Errorf("app type %q does not support start command configuration", appType)
	}

	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/%s", url.PathEscape(appID), url.PathEscape(specPath))

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return errors.Wrapf(err, "update start command for app %s", appID)
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf(
			"update start command for app %s failed: [%d] -> %s",
			appID,
			resp.StatusCode(),
			resp.Body(),
		)
	}

	return nil
}

// ListAppPolarisConfigs 获取应用的北极星配置列表
func (c *SvcBasedClient) ListAppPolarisConfigs(ctx context.Context, appID string) ([]PolarisConfig, error) {
	var respData ListPolarisConfigsRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/deps/polaris-configs", url.PathEscape(appID))

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list app polaris configs failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data, nil
}

// CreateAppPolarisConfig 创建应用的北极星配置，返回配置名称
func (c *SvcBasedClient) CreateAppPolarisConfig(ctx context.Context, appID string, body any) (string, error) {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/deps/polaris-configs", url.PathEscape(appID))

	var respData CreatePolarisConfigRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(path)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return "", errors.Errorf("create app polaris config failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Name, nil
}

// DeleteAppPolarisConfig 删除应用的北极星配置
func (c *SvcBasedClient) DeleteAppPolarisConfig(ctx context.Context, appID, configName string) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/deps/polaris-configs/%s",
		url.PathEscape(appID),
		url.PathEscape(configName),
	)

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("delete app polaris config failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// ListWorkspaceComponents 获取工作空间组件列表
func (c *SvcBasedClient) ListWorkspaceComponents(
	ctx context.Context,
	workspaceID string,
) ([]WorkspaceComponent, error) {
	var respData ListWorkspaceComponentsRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/components", url.PathEscape(workspaceID))

	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list workspace components failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data, nil
}

// CreateAppComponent 添加应用组件，返回组件名称
func (c *SvcBasedClient) CreateAppComponent(ctx context.Context, appID string, body any) (string, error) {
	var respData CreateAppComponentRespData
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/components", url.PathEscape(appID))

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(path)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return "", errors.Errorf("create app component failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Name, nil
}

// DeleteAppComponent 删除应用组件
func (c *SvcBasedClient) DeleteAppComponent(ctx context.Context, appID, compName string) error {
	path := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/components/%s", url.PathEscape(appID), url.PathEscape(compName))

	resp, err := c.cli.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("delete app component failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// PatchAppPolarisConfig 更新应用的北极星配置（部分更新）
func (c *SvcBasedClient) PatchAppPolarisConfig(ctx context.Context, appID, configName string, body any) error {
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/deps/polaris-configs/%s",
		url.PathEscape(appID),
		url.PathEscape(configName),
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Patch(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("patch app polaris config failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// UpdatePolarisConfigEnvWeight 更新北极星配置在指定环境下的全局默认权重（对该环境所有实例生效）
func (c *SvcBasedClient) UpdatePolarisConfigEnvWeight(
	ctx context.Context,
	appID, configName, envName string,
	weight int32,
) error {
	body := UpdatePolarisConfigEnvWeightBody{Weight: weight}
	path := fmt.Sprintf(
		"/bkms/v1/bkms-server/apps/%s/deps/polaris-configs/%s/envs/%s/weight",
		appID, configName, envName,
	)

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(path)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("update polaris config env weight failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// truncateBody 将响应体截断到 500 字符，超出部分用 "..." 替代，便于错误信息展示
func truncateBody(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) <= 500 {
		return s
	}
	return s[:500] + "..."
}
