// Package bscp 提供蓝鲸 bscp 服务的 API 调用封装
package bscp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// Client BSCP API 客户端接口
type Client interface {
	// ListUserBizs 获取用户有权限的业务（空间）列表
	ListUserBizs(ctx context.Context) ([]Biz, error)
	// GetBiz 获取指定业务信息
	GetBiz(ctx context.Context, bizID string) (*Biz, error)
	// CreateService 在业务下创建 BSCP 服务
	CreateService(ctx context.Context, req *CreateServiceReq) (*Service, error)
	// ListBizServices 获取业务下的服务列表
	ListBizServices(ctx context.Context, bizID string) ([]Service, error)
	// GetBizService 获取指定服务
	GetBizService(ctx context.Context, bizID, svcID string) (*Service, error)
	// ListServiceVersions 获取服务下版本列表
	ListServiceVersions(ctx context.Context, bizID, svcID string) (Versions, error)
	// ListServiceConfigs 获取服务下的配置项列表
	ListServiceConfigs(ctx context.Context, bizID, svcID, versionID string) ([]Config, error)
	// GetServiceConfig 获取指定的配置项
	GetServiceConfig(ctx context.Context, bizID, svcID, versionID, id string) (Config, error)
	// GetConfigContent 获取配置项的内容
	GetConfigContent(ctx context.Context, bizID, svcID, versionID, id string) (string, error)
	// GetOrCreateService 获取或创建 BSCP 服务
	GetOrCreateService(ctx context.Context, req *CreateServiceReq) (*Service, error)
	// CreateCredential 创建客户端密钥
	CreateCredential(ctx context.Context, req *CreateCredentialReq) (int64, error)
	// ListCredentials 获取业务下的客户端密钥列表
	ListCredentials(ctx context.Context, bizID string) ([]Credential, error)
	// UpdateCredential 更新客户端密钥
	UpdateCredential(ctx context.Context, req *UpdateCredentialReq) error
	// CheckCredentialName 检测客户端密钥名称是否已存在
	CheckCredentialName(ctx context.Context, bizID, name string) (bool, error)
	// UpdateCredentialScope 更新客户端密钥关联服务规则
	UpdateCredentialScope(ctx context.Context, req *UpdateCredentialScopeReq) error
	// ListCredentialScopes 获取客户端密钥关联服务列表
	ListCredentialScopes(ctx context.Context, bizID, credentialID string) ([]CredentialScope, error)
	// CreateHook 创建脚本
	CreateHook(ctx context.Context, req *CreateHookReq) (int64, error)
	// DeleteHook 删除脚本
	DeleteHook(ctx context.Context, req *DeleteHookReq) error
	// GetHook 获取脚本详情
	GetHook(ctx context.Context, bizID string, hookID int64) (*Hook, error)
	// GetReleaseHook 获取版本绑定的前后置脚本
	GetReleaseHook(ctx context.Context, bizID string, appID, releaseID int64) (*ReleaseHook, error)
	// ListHooks 获取脚本列表
	ListHooks(ctx context.Context, req *ListHooksReq) (*ListHooksResp, error)
	// UpdateConfigHook 更新服务绑定的前后置脚本
	UpdateConfigHook(ctx context.Context, req *UpdateConfigHookReq) error
	// UpdateHook 更新脚本信息（标签、描述）
	UpdateHook(ctx context.Context, req *UpdateHookReq) error
}

// ApiClient 蓝鲸 BSCP 组件 API Client
// 通过用户态调用 BSCP API，若无对应权限会报错
type ApiClient struct {
	define.BkApiClient
	user auth.User
}

// New 创建 BSCP API 客户端，根据配置返回真实客户端或 stub 客户端
func New(user auth.User) (Client, error) {
	// 测试时使用 stub 客户端
	if config.G.Development.UseStubBSCP {
		log.InfoNoContext("use stub bscp client according to config")
		return NewStub(user), nil
	}

	client, err := newApiClient(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial bscp client")
	}

	return &ApiClient{client, user}, nil
}

// newApiClient 创建 ApiClient
// 仅供包内部需要直接访问 ApiClient 私有方法的场景使用（如 File.Content）
func newApiClient(user auth.User) (*ApiClient, error) {
	authInfo, err := generateAuthInfo(user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate auth info")
	}

	client, err := bkapi.NewBkApiClient("bk-bscp", buildClientConfig(authInfo))
	if err != nil {
		return nil, err
	}

	return &ApiClient{client, user}, nil
}

// generateAuthInfo 生成鉴权信息
func generateAuthInfo(user auth.User) (string, error) {
	authorization, err := json.Marshal(map[string]string{
		"bk_app_code":       config.G.BkApp.Code,
		"bk_app_secret":     config.G.BkApp.Secret,
		user.Cred.CredKey(): user.Cred.CredValue(),
	})
	if err != nil {
		return "", err
	}

	return string(authorization), nil
}

// buildClientConfig 构建客户端配置
func buildClientConfig(authInfo string) bkapi.ClientConfig {
	return bkapi.ClientConfig{
		Stage:        config.G.BkApiStages.BSCP,
		BkApiUrlTmpl: config.G.BkPlatUrls.BkApiUrlTmpl,
		ClientOptions: []define.BkApiClientOption{
			bkapi.OptSetRequestHeader("x-bkapi-authorization", authInfo),
			bkapi.OptJsonResultProvider(),
			bkapi.OptJsonBodyProvider(),
			bkapi.OptTimeout(60 * time.Second),
		},
	}
}

// handleOperation 发起请求并检查结果，自动上报指标
func (c *ApiClient) handleOperation(
	ctx context.Context, op define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	defer metrics.ReportClientRequestMetric("bscp", op.FullName(), started, &err)

	ctx, span := apm.StartClientSpan(ctx, "bscp", op.FullName())
	chain := op.SetContext(ctx)
	resp, err := chain.SetResult(&result).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !httpresp.IsSuccess(resp) {
		errMsg, _ := io.ReadAll(resp.Body)
		return nil, wrapOperationHTTPError(op.FullName(), resp.StatusCode, errMsg)
	}

	return result, nil
}

// ErrNoPermission indicates BSCP rejected the request with an auth/perm status.
var ErrNoPermission = errors.New("bscp no permission")

func wrapOperationHTTPError(operationName string, statusCode int, errMsg []byte) error {
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return errors.Wrapf(
			ErrNoPermission,
			"call bscp api %s failed, http code: %d, err: %s",
			operationName,
			statusCode,
			errMsg,
		)
	}
	return errors.Errorf(
		"call bscp api %s failed, http code: %d, err: %s",
		operationName,
		statusCode,
		errMsg,
	)
}
