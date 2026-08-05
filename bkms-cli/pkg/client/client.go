// Package client provides API call for bkms-cli
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

// ListWorkspaces 获取工作空间列表
func (c *SvcBasedClient) ListWorkspaces(ctx context.Context, keyword string) ([]Workspace, error) {
	url := "/bkms/v1/bkms-server/workspaces"

	var respData ListWorkspacesRespData
	resp, err := c.cli.R().SetContext(ctx).SetQueryParam("keyword", keyword).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s", id)

	var respData GetWorkspaceRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/envs", workspaceID)

	var respData ListEnvsRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list envs failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data, nil
}

// ListApps 获取应用列表
func (c *SvcBasedClient) ListApps(ctx context.Context, workspaceID string) ([]AppMinimal, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/apps", workspaceID)

	var respData ListAppsRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
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

// GetAppIDAutoSuffix 获取应用 ID 自动后缀（后端生成）
func (c *SvcBasedClient) GetAppIDAutoSuffix(ctx context.Context) (string, error) {
	url := "/bkms/v1/bkms-server/apps/auto-id-suffix"

	var respData GetAppIDAutoSuffixRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/workspaces/%s/apps", workspaceID)

	var respData CreateAppRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/builds", appID)
	body := map[string]any{"branch": opts.Branch, "imageTag": opts.ImageTag}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/images", appID)
	queryParams := map[string]string{
		"keyword":  keyword,
		"page":     "1",
		"pageSize": "10",
	}

	var respData ListAppImagesResp
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-files", appID)

	var respData ListAppConfigFilesRespData
	req := c.cli.R().SetContext(ctx).SetResult(&respData)
	if envName != "" {
		req.SetQueryParam("envName", envName)
	}

	resp, err := req.Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-files/%s/details", appID, fileID)

	var details AppConfigFileDetails
	resp, err := c.cli.R().SetContext(ctx).SetResult(&details).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-file/versions", appID)

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

	resp, err := req.Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-file/versions/%s", appID, versionID)

	var respData GetAppConfigFileVersionRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-file/versions/%s", appID, versionID)

	resp, err := c.cli.R().SetContext(ctx).Delete(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-file/versions/%s/rollback", appID, versionID)
	body := map[string]any{
		"currentVersion": opts.CurrentVersion,
		"description":    opts.Description,
	}

	var respData RollbackAppConfigFileVersionRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-files/%s/content", appID, fileID)
	body := map[string]any{
		"content":        opts.Content,
		"description":    opts.Description,
		"currentVersion": opts.CurrentVersion,
	}

	var result AppConfigFileContentUpdateResult
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&result).Put(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-config-files/%s/overlay-content", appID, fileID)
	body := map[string]any{
		"overlayContent": opts.Content,
		"description":    opts.Description,
		"currentVersion": opts.CurrentVersion,
	}

	var result AppConfigFileContentUpdateResult
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&result).Put(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/builds", appID)
	queryParams := map[string]string{
		"keyword":  keyword,
		"page":     "1",
		"pageSize": "10",
	}

	var respData ListBuildRecordsRespData
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/helm-deploys", appID, envName)
	body := map[string]any{
		"imageTag":        opts.ImageTag,
		"chartVersion":    opts.ChartVersion,
		"valuesFileID":    opts.ValuesFileID,
		"trafficLaneName": opts.TrafficLaneName,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/helm-deploys", appID, envName)
	queryParams := map[string]string{
		"trafficLaneName": trafficLaneName,
		"keyword":         keyword,
		"page":            "1",
		"pageSize":        "10",
	}

	var respData ListHelmDeployRecordsRespData
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/trpc-deploys", appID, envName)
	body := map[string]any{
		"imageTag":        opts.ImageTag,
		"replicas":        opts.Replicas,
		"trafficLaneName": opts.TrafficLaneName,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/trpc-deploys", appID, envName)
	queryParams := map[string]string{
		"keyword":         keyword,
		"trafficLaneName": trafficLaneName,
		"page":            "1",
		"pageSize":        "10",
	}

	var respData AppModelDeployRecordsResp
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list trpc deploy records failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// GrayscaleUpdateInstance 灰度更新 Trpc 实例
func (c *SvcBasedClient) GrayscaleUpdateInstance(
	ctx context.Context, appID, envName, imageTag string, instanceIDs []string, updateStrategy string,
) error {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/instances", appID, envName)
	body := map[string]any{
		// fixme 泳道支持
		"imageTag":       imageTag,
		"instanceIDs":    instanceIDs,
		"updateStrategy": updateStrategy,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/instances", appID, envName)
	body := map[string]any{
		"imageTag":       imageTag,
		"updateStrategy": updateStrategy,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(url)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("update trpc instance failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// GetEnvEffectiveDevMode 获取应用在某个环境下实际生效的开发模式配置
func (c *SvcBasedClient) GetEnvEffectiveDevMode(ctx context.Context, appID, envName string) (*DevModeConfig, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/app-spec/dev-mode/effective", appID, envName)

	var respData GetEnvEffectiveDevModeRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("get env effective dev mode failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data, nil
}

// ListAppInstances 获取应用实例列表
func (c *SvcBasedClient) ListAppInstances(
	ctx context.Context,
	appID, envName string,
	opts ListAppInstancesOptions,
) (*PaginatedInstances, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/instances", appID, envName)

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
	}).SetResult(&respData).Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list app instances failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return &respData.Data, nil
}

// ListTrpcAdminCmds 查询 Trpc 管理命令列表
func (c *SvcBasedClient) ListTrpcAdminCmds(
	ctx context.Context, appID, envName string, instanceIDs []string,
) ([]string, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/instances/admin-cmds", appID, envName)

	var respData ListTrpcAdminCmdsRespData
	req := c.cli.R().SetContext(ctx).SetResult(&respData)
	for _, id := range instanceIDs {
		req.QueryParam.Add("instanceIDs", id)
	}
	resp, err := req.Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/instances/admin-cmds", appID, envName)

	var respData ExecuteAdminCmdRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).SetResult(&respData).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/instances/taf-admin-cmds", appID, envName)

	var respData ExecuteAdminCmdRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(opts).SetResult(&respData).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/taf-deploys", appID, envName)
	body := map[string]any{
		"imageTag":        opts.ImageTag,
		"replicas":        opts.Replicas,
		"trafficLaneName": opts.TrafficLaneName,
	}

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/taf-deploys", appID, envName)
	queryParams := map[string]string{
		"keyword":         keyword,
		"trafficLaneName": trafficLaneName,
		"page":            "1",
		"pageSize":        "10",
	}

	var respData AppModelDeployRecordsResp
	resp, err := c.cli.R().SetContext(ctx).SetQueryParams(queryParams).SetResult(&respData).Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("list taf deploy records failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return respData.Data.Results, nil
}

// --- AppSpec method implementations ---

// GetAppDetail queries the full app detail to retrieve type and start command info.
func (c *SvcBasedClient) GetAppDetail(ctx context.Context, appID string) (*AppDetail, error) {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s", appID)

	var respData GetAppDetailRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-spec/default-%s", appID, section)
	return c.getAppSpecSection(ctx, url, fmt.Sprintf("get default %s", section), result)
}

// GetAppSpecEnvEffectiveSection 获取应用环境生效 section 配置。
func (c *SvcBasedClient) GetAppSpecEnvEffectiveSection(
	ctx context.Context,
	appID, envName string,
	section AppSpecSectionName,
	result any,
) error {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/app-spec/%s/effective", appID, envName, section)
	return c.getAppSpecSection(ctx, url, fmt.Sprintf("get env effective %s", section), result)
}

func (c *SvcBasedClient) getAppSpecSection(ctx context.Context, url, desc string, result any) error {
	type respWrapper struct {
		Data any `json:"data"`
	}
	wrapper := &respWrapper{Data: result}
	resp, err := c.cli.R().SetContext(ctx).SetResult(wrapper).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/app-spec/default-%s", appID, section)

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/app-spec/%s", appID, envName, section)

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/envs/%s/app-spec/%s", appID, envName, section)

	resp, err := c.cli.R().SetContext(ctx).Delete(url)
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

	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/%s", appID, specPath)

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Put(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/deps/polaris-configs", appID)

	var respData ListPolarisConfigsRespData
	resp, err := c.cli.R().SetContext(ctx).SetResult(&respData).Get(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/deps/polaris-configs", appID)

	var respData CreatePolarisConfigRespData
	resp, err := c.cli.R().SetContext(ctx).SetBody(body).SetResult(&respData).Post(url)
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
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/deps/polaris-configs/%s", appID, configName)

	resp, err := c.cli.R().SetContext(ctx).Delete(url)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return errors.Errorf("delete app polaris config failed: [%d] -> %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// PatchAppPolarisConfig 更新应用的北极星配置（部分更新）
func (c *SvcBasedClient) PatchAppPolarisConfig(ctx context.Context, appID, configName string, body any) error {
	url := fmt.Sprintf("/bkms/v1/bkms-server/apps/%s/deps/polaris-configs/%s", appID, configName)

	resp, err := c.cli.R().SetContext(ctx).SetBody(body).Patch(url)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("patch app polaris config failed: [%d] -> %s", resp.StatusCode(), resp.Body())
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
