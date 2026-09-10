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

// CreateCredential 在项目下创建客户端密钥
func (c *ConfigApiClient) CreateCredential(ctx context.Context, req *CreateCredentialReq) (int64, error) {
	if err := req.Validate(); err != nil {
		return 0, errors.Wrap(err, "validate create credential req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_CreateCredentials2",
			Method: "POST",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/credentials",
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
			err, "call bscp create credential api, bizID: %s, projectID: %d, name: %s",
			req.BizID, req.ProjectID, req.Name,
		)
	}

	id := cast.ToInt64(mapx.Get(result, "data.id", 0))
	if id == 0 {
		return 0, errors.Errorf(
			"create credential returned empty id, bizID: %s, projectID: %d, name: %s",
			req.BizID, req.ProjectID, req.Name,
		)
	}

	return id, nil
}

// ListCredentials 获取项目下的客户端密钥列表
func (c *ConfigApiClient) ListCredentials(ctx context.Context, bizID string, projectID int64) ([]Credential, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListCredentials2",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/credentials",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":     bizID,
			"project_id": cast.ToString(projectID),
		}),
		bkapi.OptSetRequestQueryParam("all", "true"),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(err, "call bscp list credentials api, bizID: %s, projectID: %d", bizID, projectID)
	}

	var credentials []Credential
	for _, item := range mapx.GetList(result, "data.details") {
		if v, ok := item.(map[string]any); ok {
			cred := Credential{
				ID:             cast.ToInt64(mapx.Get(v, "id", 0)),
				Name:           mapx.GetStr(v, "spec.name"),
				Memo:           mapx.GetStr(v, "spec.memo"),
				Enable:         mapx.GetBool(v, "spec.enable"),
				CredentialType: mapx.GetStr(v, "spec.credential_type"),
				EncCredential:  mapx.GetStr(v, "spec.enc_credential"),
			}
			credentials = append(credentials, cred)
		}
	}
	return credentials, nil
}

// UpdateCredentialScope 更新项目下密钥关联服务规则
func (c *ConfigApiClient) UpdateCredentialScope(ctx context.Context, req *UpdateCredentialScopeReq) error {
	if err := req.Validate(); err != nil {
		return errors.Wrap(err, "validate update credential scope req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_UpdateCredentialScope2",
			Method: "PUT",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/credential/{credential_id}/scope",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":        req.BizID,
			"project_id":    cast.ToString(req.ProjectID),
			"credential_id": cast.ToString(req.CredentialID),
		}),
		bkapi.OptSetRequestBody(req),
	)

	_, err := c.handleOperation(ctx, op)
	if err != nil {
		return errors.Wrapf(
			err, "call bscp update credential scope api, bizID: %s, projectID: %d, credentialID: %d",
			req.BizID, req.ProjectID, req.CredentialID,
		)
	}

	return nil
}

// ListCredentialScopes 获取项目下密钥关联服务列表
func (c *ConfigApiClient) ListCredentialScopes(
	ctx context.Context, bizID string, projectID, credentialID int64,
) ([]CredentialScope, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListCredentialScopes2",
			Method: "GET",
			Path:   "/api/v1/config/biz/{biz_id}/projects/{project_id}/credential/{credential_id}/scopes",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":        bizID,
			"project_id":    cast.ToString(projectID),
			"credential_id": cast.ToString(credentialID),
		}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(
			err, "call bscp list credential scopes api, bizID: %s, projectID: %d, credentialID: %d",
			bizID, projectID, credentialID,
		)
	}

	var scopes []CredentialScope
	for _, item := range mapx.GetList(result, "data.details") {
		if v, ok := item.(map[string]any); ok {
			scopes = append(scopes, CredentialScope{
				ID:      cast.ToInt64(mapx.Get(v, "id", 0)),
				App:     mapx.GetStr(v, "spec.app"),
				Scope:   mapx.GetStr(v, "spec.scope"),
				EnvID:   cast.ToInt64(mapx.Get(v, "spec.env_id", 0)),
				EnvName: mapx.GetStr(v, "spec.env_name"),
				EnvType: mapx.GetStr(v, "spec.env_type"),
			})
		}
	}
	return scopes, nil
}
