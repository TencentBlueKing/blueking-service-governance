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
	"net/url"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
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
		createParams.Metadata,
	)
	if err != nil {
		metrics.DepservicePolarisFailed("create")
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

// UpdateInstance 更新 Polaris 服务实例，同步完成。
func (p *Provider) UpdateInstance(
	ctx context.Context,
	_ string,
	_ *types.ServicePlanConfig,
	instConfig map[string]any,
	params types.ProvisionParams,
) error {
	updateParams, ok := params.(*UpdateParams)
	if !ok {
		return errors.New("invalid params type, expected *polaris.UpdateParams")
	}
	if err := updateParams.Validate(); err != nil {
		return errors.Wrap(err, "validate polaris update params")
	}

	instCfg, err := ParseInstConfig(instConfig)
	if err != nil {
		return errors.Wrap(err, "parse polaris inst config")
	}
	if err = instCfg.Validate(); err != nil {
		return errors.Wrap(err, "validate polaris inst config")
	}

	updateMetadata := len(updateParams.Metadata) > 0 || len(updateParams.MetadataKeysToDelete) > 0
	var metadata map[string]string
	if updateMetadata {
		existing, getErr := p.getServiceMetadata(ctx, instCfg.PolarisName, instCfg.PolarisNamespace)
		if getErr != nil {
			metrics.DepservicePolarisFailed("update")
			return errors.Wrap(getErr, "get polaris service metadata")
		}
		metadata = mergeServiceMetadata(existing, updateParams.Metadata, updateParams.MetadataKeysToDelete)
	}

	if err = p.updateService(
		ctx,
		instCfg.PolarisName,
		instCfg.PolarisNamespace,
		instCfg.Token,
		updateParams.Owners,
		metadata,
		updateMetadata,
	); err != nil {
		metrics.DepservicePolarisFailed("update")
		return errors.Wrap(err, "update polaris service")
	}
	return nil
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
		metrics.DepservicePolarisFailed("delete")
		return nil, err
	}
	return &types.DeleteInstanceResult{}, nil
}

// createService calls Polaris API to create a service
func (p *Provider) createService(
	ctx context.Context,
	name, namespace, owners string,
	metadata map[string]string,
) (string, error) {
	item := map[string]any{
		"name":      name,
		"namespace": namespace,
		"owners":    owners,
	}
	if len(metadata) > 0 {
		item["metadata"] = metadata
	}

	respBody, err := p.doRequest(ctx, http.MethodPost, "/naming/v1/services", []map[string]any{item})
	if err != nil {
		return "", err
	}

	tokenField := gjson.GetBytes(respBody, "responses.0.service.token")
	if !tokenField.Exists() {
		return "", errors.New("token not found in response")
	}

	return tokenField.String(), nil
}

// updateService calls Polaris API to update a service.
// includeMetadata 为 true 时带上 metadata（空 map 也会下发，以便清空）。
func (p *Provider) updateService(
	ctx context.Context,
	name, namespace, token, owners string,
	metadata map[string]string,
	updateMetadata bool,
) error {
	item := map[string]any{
		"name":      name,
		"namespace": namespace,
		"token":     token,
	}
	if owners != "" {
		item["owners"] = owners
	}
	if updateMetadata {
		if metadata == nil {
			metadata = map[string]string{}
		}
		item["metadata"] = metadata
	}

	_, err := p.doRequest(ctx, http.MethodPut, "/naming/v1/services", []map[string]any{item})
	return err
}

// getServiceMetadata 查询北极星服务当前 metadata，供更新时合并。
func (p *Provider) getServiceMetadata(ctx context.Context, name, namespace string) (map[string]string, error) {
	query := url.Values{}
	query.Set("name", name)
	query.Set("namespace", namespace)
	query.Set("offset", "0")
	query.Set("limit", "10")

	respBody, err := p.doRequest(ctx, http.MethodGet, "/naming/v1/services?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	return parseServiceMetadata(respBody, name, namespace)
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

	reqURL := p.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
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

func parseServiceMetadata(respBody []byte, name, namespace string) (map[string]string, error) {
	services := gjson.GetBytes(respBody, "services")
	if !services.IsArray() {
		return nil, errors.New("invalid polaris services response")
	}
	for _, svc := range services.Array() {
		if svc.Get("name").String() != name || svc.Get("namespace").String() != namespace {
			continue
		}
		return gjsonObjectToStringMap(svc.Get("metadata")), nil
	}
	return nil, errors.New("polaris service not found")
}

func gjsonObjectToStringMap(obj gjson.Result) map[string]string {
	result := make(map[string]string)
	if !obj.IsObject() {
		return result
	}
	obj.ForEach(func(key, value gjson.Result) bool {
		result[key.String()] = value.String()
		return true
	})
	return result
}

// mergeServiceMetadata 以 existing 为底，先写入 overlay，再删除指定键。
func mergeServiceMetadata(existing, overlay map[string]string, deleteKeys []string) map[string]string {
	merged := make(map[string]string, len(existing)+len(overlay))
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range overlay {
		merged[k] = v
	}
	for _, k := range deleteKeys {
		delete(merged, k)
	}
	return merged
}
