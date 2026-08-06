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

// Package bscp 提供蓝鲸 bscp 服务的 API 调用封装 - 凭证（Credential）管理
package bscp

import (
	"context"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// CreateCredential 创建客户端密钥
func (c *ApiClient) CreateCredential(ctx context.Context, req *CreateCredentialReq) (int64, error) {
	if err := req.Validate(); err != nil {
		return 0, errors.Wrap(err, "validate create credential req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_CreateCredentials",
			Method: "POST",
			Path:   "/api/v1/config/biz_id/{biz_id}/credentials",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": req.BizID}),
		bkapi.OptSetRequestBody(req),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return 0, errors.Wrapf(err, "call bscp create credential api, bizID: %s, name: %s", req.BizID, req.Name)
	}

	id := cast.ToInt64(mapx.Get(result, "data.id", 0))
	if id == 0 {
		return 0, errors.Errorf("create credential returned empty id, bizID: %s, name: %s", req.BizID, req.Name)
	}

	return id, nil
}

// ListCredentials 获取业务下的客户端密钥列表
func (c *ApiClient) ListCredentials(ctx context.Context, bizID string) ([]Credential, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListCredentials",
			Method: "GET",
			Path:   "/api/v1/config/biz_id/{biz_id}/credentials",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": bizID}),
		bkapi.OptSetRequestQueryParam("all", "true"),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(err, "call bscp list credentials api, bizID: %s", bizID)
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
			// 解析关联规则
			if scopes, sOk := v["credentialScopes"].([]any); sOk {
				for _, s := range scopes {
					if str, strOk := s.(string); strOk {
						cred.CredentialScopes = append(cred.CredentialScopes, str)
					}
				}
			}
			credentials = append(credentials, cred)
		}
	}

	return credentials, nil
}

// UpdateCredential 更新客户端密钥
func (c *ApiClient) UpdateCredential(ctx context.Context, req *UpdateCredentialReq) error {
	if err := req.Validate(); err != nil {
		return errors.Wrap(err, "validate update credential req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_UpdateCredential",
			Method: "PUT",
			Path:   "/api/v1/config/biz_id/{biz_id}/credential",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"biz_id": req.BizID}),
		bkapi.OptSetRequestBody(req),
	)

	_, err := c.handleOperation(ctx, op)
	if err != nil {
		return errors.Wrapf(err, "call bscp update credential api, bizID: %s, id: %d", req.BizID, req.ID)
	}

	return nil
}

// CheckCredentialName 检测客户端密钥名称是否已存在
func (c *ApiClient) CheckCredentialName(ctx context.Context, bizID, name string) (bool, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_CheckCredentialName",
			Method: "GET",
			Path:   "/api/v1/config/biz_id/{biz_id}/credential/{credential_name}/check",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":          bizID,
			"credential_name": name,
		}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return false, errors.Wrapf(err, "call bscp check credential name api, bizID: %s, name: %s", bizID, name)
	}

	return mapx.GetBool(result, "data.exist"), nil
}

// UpdateCredentialScope 更新客户端密钥关联服务规则
func (c *ApiClient) UpdateCredentialScope(ctx context.Context, req *UpdateCredentialScopeReq) error {
	if err := req.Validate(); err != nil {
		return errors.Wrap(err, "validate update credential scope req")
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_UpdateCredentialScope",
			Method: "PUT",
			Path:   "/api/v1/config/biz_id/{biz_id}/credential/{credential_id}/scope",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":        req.BizID,
			"credential_id": req.CredentialID,
		}),
		bkapi.OptSetRequestBody(req),
	)

	_, err := c.handleOperation(ctx, op)
	if err != nil {
		return errors.Wrapf(
			err, "call bscp update credential scope api, bizID: %s, credentialID: %s",
			req.BizID, req.CredentialID,
		)
	}

	return nil
}

// ListCredentialScopes 获取客户端密钥关联服务列表
func (c *ApiClient) ListCredentialScopes(ctx context.Context, bizID, credentialID string) ([]CredentialScope, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "Config_ListCredentialScopes",
			Method: "GET",
			Path:   "/api/v1/config/biz_id/{biz_id}/credential/{credential_id}/scopes",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"biz_id":        bizID,
			"credential_id": credentialID,
		}),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, errors.Wrapf(
			err, "call bscp list credential scopes api, bizID: %s, credentialID: %s",
			bizID, credentialID,
		)
	}

	var scopes []CredentialScope
	for _, item := range mapx.GetList(result, "data.details") {
		if v, ok := item.(map[string]any); ok {
			scopes = append(scopes, CredentialScope{
				ID:    cast.ToInt64(mapx.Get(v, "id", 0)),
				App:   mapx.GetStr(v, "spec.app"),
				Scope: mapx.GetStr(v, "spec.scope"),
			})
		}
	}

	return scopes, nil
}
