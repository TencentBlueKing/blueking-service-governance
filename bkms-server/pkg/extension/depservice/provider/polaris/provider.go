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

package polaris

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

// Provider implements ServiceProvider for Polaris
type Provider struct {
	httpCli *http.Client
	config  *Config
}

// Config represents the Polaris provider configuration
type Config struct {
	BaseURL string `mapstructure:"baseUrl"`
}

// parseConfig parses the plan config into Polaris Config
func parseConfig(planConfig map[string]any) (*Config, error) {
	cfg := new(Config)
	if err := mapstructure.Decode(planConfig, cfg); err != nil {
		return nil, errors.Wrap(err, "decode polaris config")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("baseUrl is required")
	}
	return cfg, nil
}

// NewProvider creates a new Polaris provider
func NewProvider(planConfig map[string]any) (*Provider, error) {
	cfg, err := parseConfig(planConfig)
	if err != nil {
		return nil, err
	}

	return &Provider{
		httpCli: &http.Client{
			Timeout:   30 * time.Second,
			Transport: httpcli.NewTransport(),
		},
		config: cfg,
	}, nil
}

// CreateInstance creates a Polaris service instance
func (p *Provider) CreateInstance(
	ctx context.Context,
	_ string,
	_ *types.ServicePlanConfig,
	params types.ProvisionParams,
) (*types.CreateInstanceResult, error) {
	createParams, ok := params.(*CreateParams)
	if !ok {
		return nil, errors.New("invalid params type, expected *polaris.CreateParams")
	}
	if err := createParams.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate polaris create params")
	}

	token, err := p.createService(
		ctx,
		createParams.PolarisName,
		createParams.PolarisNamespace,
		createParams.Owners,
	)
	if err != nil {
		return nil, errors.Wrap(err, "create polaris service")
	}

	instConfigMap, err := types.ToMap(&InstConfig{
		PolarisName:      createParams.PolarisName,
		PolarisNamespace: createParams.PolarisNamespace,
		Token:            token,
	})
	if err != nil {
		return nil, errors.Wrap(err, "marshal inst config")
	}

	credentialsMap, err := types.ToMap(&Credentials{Token: token})
	if err != nil {
		return nil, errors.Wrap(err, "marshal credentials")
	}

	return &types.CreateInstanceResult{
		InstConfig:  instConfigMap,
		Credentials: credentialsMap,
	}, nil
}

// DeleteInstance 删除 Polaris 服务实例，同步完成。
//
// 若实例配置不完整（例如仍处于 provisioning、尚未写出 polaris 资源标识），
// 视为外部资源不存在，直接返回成功，由 Manager 删除本地记录。
func (p *Provider) DeleteInstance(
	ctx context.Context,
	_ string,
	_ *types.ServicePlanConfig,
	instConfig map[string]any,
) (*types.DeleteInstanceResult, error) {
	instCfg, err := ParseInstConfig(instConfig)
	if err != nil {
		return nil, errors.Wrap(err, "parse polaris inst config")
	}
	// 配置不完整视为外部资源不存在，跳过 Polaris 调用（幂等删除）
	if err = instCfg.Validate(); err != nil {
		//nolint:nilerr // intentional: incomplete config means nothing to delete remotely
		return &types.DeleteInstanceResult{}, nil
	}
	if err = p.deleteService(ctx, instCfg.PolarisName, instCfg.PolarisNamespace, instCfg.Token); err != nil {
		return nil, err
	}
	return &types.DeleteInstanceResult{}, nil
}

// createService calls Polaris API to create a service
func (p *Provider) createService(ctx context.Context, name, namespace, owners string) (string, error) {
	reqBody := []map[string]any{
		{
			"name":      name,
			"namespace": namespace,
			"owners":    owners,
		},
	}

	respBody, err := p.doRequest(ctx, http.MethodPost, "/naming/v1/services", reqBody)
	if err != nil {
		return "", err
	}

	tokenField := gjson.GetBytes(respBody, "responses.0.service.token")
	if !tokenField.Exists() {
		return "", errors.New("token not found in response")
	}

	return tokenField.String(), nil
}

// deleteService calls Polaris API to delete a service
func (p *Provider) deleteService(ctx context.Context, name, namespace, token string) error {
	reqBody := []map[string]any{
		{
			"name":      name,
			"namespace": namespace,
			"token":     token,
		},
	}

	_, err := p.doRequest(ctx, http.MethodPost, "/naming/v1/services/delete", reqBody)
	return err
}

// doRequest performs HTTP request to Polaris API
func (p *Provider) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "marshal request body")
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := p.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpCli.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "do request")
	}
	defer resp.Body.Close() // nolint

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body")
	}

	// 非 200 状态码，尝试从 info 字段获取错误信息
	if resp.StatusCode != http.StatusOK {
		if info := gjson.GetBytes(respBody, "info"); info.Exists() && info.String() != "" {
			return nil, errors.Errorf("polaris api error: %s", info.String())
		}
		return nil, errors.Errorf("polaris api error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
