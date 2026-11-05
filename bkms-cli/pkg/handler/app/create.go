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

// Package app 提供应用创建相关的处理逻辑
package app

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// CreateApp 创建应用，读取 YAML 文件并调用后端 API
// 返回创建成功的应用信息，由上层决定如何输出
func CreateApp(ctx context.Context, workspaceID, specFile string) (*client.AppMinimal, error) {
	if _, err := os.Stat(specFile); err != nil {
		return nil, errors.Wrapf(err, "app spec file %s not found", specFile)
	}
	file, err := os.ReadFile(specFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read app spec file %s failed", specFile)
	}
	// 已废弃的字段 YAML 解析时会被忽略，先显式拒绝，避免报出让人误解的缺字段错误
	if err = rejectRemovedFields(file); err != nil {
		return nil, err
	}
	spec := new(AppCreateSpec)
	if err = yaml.Unmarshal(file, spec); err != nil {
		return nil, errors.Wrapf(err, "parse app spec file %s failed, please check YAML syntax", specFile)
	}
	if err = spec.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate app spec")
	}

	// 应用 ID 固定由服务端返回的后缀与 name 拼接生成，不允许用户指定
	suffix, err := client.New().GetAppIDAutoSuffix(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get app id auto suffix")
	}
	appID, err := buildAppID(spec.Name, suffix)
	if err != nil {
		return nil, err
	}

	// 构建后端 API 请求体：平台托管应用走 bkmsapps，Helm / Agones 走旧的 apps 接口
	helmBody, bkmsBody, err := buildCreateAppRequest(appID, spec)
	if err != nil {
		return nil, errors.Wrap(err, "build create app request")
	}

	cli := client.New()
	if bkmsBody != nil {
		return cli.CreateBKMSApp(ctx, workspaceID, bkmsBody)
	}
	return cli.CreateApp(ctx, workspaceID, helmBody)
}

// rejectRemovedFields 拒绝已废弃的创建字段，并给出新写法指引。
// 这些字段已从 AppCreateSpec 移除，YAML 解析会静默忽略，必须在解析前单独检查。
func rejectRemovedFields(raw []byte) error {
	var doc struct {
		Type         string `yaml:"type"`
		AppModelSpec struct {
			TrpcSpec map[string]any `yaml:"trpcSpec"`
			TafSpec  map[string]any `yaml:"tafSpec"`
		} `yaml:"appModelSpec"`
	}
	// 只做字段检查：解析失败时 doc 为零值，下面都不命中，
	// YAML 语法错误交给后续的结构体解析统一报出
	_ = yaml.Unmarshal(raw, &doc)

	switch doc.Type {
	case constant.AppTypeTrpc, constant.AppTypeTaf:
		return errors.Errorf(
			"'type: %s' is no longer supported, use 'framework: %s' with 'language' and 'frameworkConfigFile' instead",
			doc.Type, doc.Type)
	case constant.AppTypeBKMSApp:
		return errors.New("'type: bkmsApp' should not be set, use 'framework' with 'language' instead")
	}
	if doc.AppModelSpec.TrpcSpec != nil || doc.AppModelSpec.TafSpec != nil {
		return errors.New(
			"'appModelSpec.trpcSpec' / 'appModelSpec.tafSpec' are no longer supported, " +
				"use the top-level 'frameworkConfigFile' with 'name' and 'mountDir' instead")
	}
	return nil
}
