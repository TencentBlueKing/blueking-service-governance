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

// Package component 处理应用组件实例相关逻辑。
package component

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/apptype"
)

// CreateAppComponentRefInput 是创建应用组件引用的请求体。
type CreateAppComponentRefInput struct {
	// CompName 应用内组件实例名称，为空时由服务端生成
	CompName string `json:"compName,omitempty"`
	// RefWorkspaceCompName 引用的工作空间组件实例名称
	RefWorkspaceCompName string `json:"refWorkspaceCompName"`
}

// ListAppComponents 列出应用组件实例。kind 为空返回全部，取值 ref 或 inst。
func ListAppComponents(
	ctx context.Context,
	cli client.Client,
	appID, kind string,
) ([]client.AppComponent, error) {
	if kind != "" && kind != client.AppComponentKindRef && kind != client.AppComponentKindInst {
		return nil, errors.Errorf("unsupported kind %q, want %s or %s",
			kind, client.AppComponentKindRef, client.AppComponentKindInst)
	}

	app, err := cli.GetAppDetail(ctx, appID)
	if err != nil {
		return nil, err
	}
	if err = ensureAppModelApp(app.Type); err != nil {
		return nil, err
	}

	comps := []client.AppComponent{}
	if app.AppModelSpec != nil {
		comps = append(comps, app.AppModelSpec.Components...)
	}
	out := make([]client.AppComponent, 0, len(comps))
	for _, comp := range comps {
		comp.Kind = componentKind(comp)
		if kind == "" || comp.Kind == kind {
			out = append(out, comp)
		}
	}
	return out, nil
}

// CreateAppComponentRef 为应用引用一个工作空间组件实例。
func CreateAppComponentRef(
	ctx context.Context,
	cli client.Client,
	appID, refName, compName string,
) (string, error) {
	app, err := cli.GetAppDetail(ctx, appID)
	if err != nil {
		return "", err
	}
	if err = ensureAppModelApp(app.Type); err != nil {
		return "", err
	}
	return cli.CreateAppComponent(ctx, appID, CreateAppComponentRefInput{
		CompName:             compName,
		RefWorkspaceCompName: refName,
	})
}

// DeleteAppComponent 按名称删除应用上的组件实例。
func DeleteAppComponent(ctx context.Context, cli client.Client, appID, compName string) error {
	app, err := cli.GetAppDetail(ctx, appID)
	if err != nil {
		return err
	}
	if err = ensureAppModelApp(app.Type); err != nil {
		return err
	}
	return cli.DeleteAppComponent(ctx, appID, compName)
}

func ensureAppModelApp(appType string) error {
	if apptype.IsAppModel(appType) {
		return nil
	}
	return errors.Errorf("app type %q does not support components (only appmodel apps are supported)", appType)
}

func componentKind(comp client.AppComponent) string {
	if comp.RefWorkspaceCompName != "" {
		return client.AppComponentKindRef
	}
	return client.AppComponentKindInst
}
