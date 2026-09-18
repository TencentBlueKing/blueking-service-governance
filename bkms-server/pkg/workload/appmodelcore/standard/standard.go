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

// Package standard converts standard application inputs into application models and specifications.
//
// standard 是语言无关的通用应用：复用 AppModel + AppSpec + GameDeployment 部署链路，但不绑定任何框架。
// 语言/框架作为 Application 顶层字段由 handler 写入，本服务只负责 AppModel/AppSpec/Application 的落库。
package standard

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// CreateParams 创建 standard 应用的参数（不依赖 API 层）
type CreateParams struct {
	// Command 容器启动命令
	Command []string
	// Args 容器启动参数
	Args []string
	// EnvVars 容器环境变量
	EnvVars []appmodel.Variable
}

// Service standard 应用服务
type Service struct {
	appModelStore       appmodel.AppModelStore
	appSpecStore        appspec.AppSpecStore
	appDefaultRuleStore appdefaults.RuleStore
	envStore            envmodel.EnvironmentStore
	appStore            bkmsapp.ApplicationStore
}

// NewService 创建服务实例
func NewService(
	appModelStore appmodel.AppModelStore,
	appSpecStore appspec.AppSpecStore,
	appDefaultRuleStore appdefaults.RuleStore,
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
) *Service {
	return &Service{
		appModelStore:       appModelStore,
		appSpecStore:        appSpecStore,
		appDefaultRuleStore: appDefaultRuleStore,
		envStore:            envStore,
		appStore:            appStore,
	}
}

// Create 创建 standard 应用资源（AppModel + AppSpec + Application）。
// standard 无框架配置文件，故不写 app_config_files（与 trpc/taf 不同）。
func (s *Service) Create(ctx context.Context, app *bkmsapp.Application, params *CreateParams) error {
	// 在写入任何应用数据前读取初始化规则，避免读取失败时留下部分应用数据。
	resolved, err := appdefaults.Resolve(
		ctx,
		s.appDefaultRuleStore,
		s.envStore,
		app.WorkspaceID,
		app.ID,
	)
	if err != nil {
		return errors.Wrap(err, "resolve application defaults")
	}

	// 初始化 AppModel
	appModel := &appmodel.AppModel{
		AppID: app.ID,
		Workload: appmodel.Workload{
			Name:    app.Name,
			Type:    appmodel.WorkloadTypeStandard,
			Command: params.Command,
			Args:    params.Args,
			EnvVars: params.EnvVars,
		},
	}

	// 将平台默认 AppSpec 应用到 AppModel。
	appspec.ApplyToAppModel(&resolved.Default, appModel)

	// 创建 AppModel
	if err = s.appModelStore.CreateAppModel(ctx, appModel); err != nil {
		return errors.Wrapf(err, "create app(%s) model", app.Name)
	}

	// 插入 appspec 初始配置
	if err = s.appSpecStore.Upsert(ctx, &resolved.Default); err != nil {
		return errors.Wrap(err, "create default app spec")
	}
	for _, spec := range resolved.Environments {
		if err = s.appSpecStore.Upsert(ctx, spec); err != nil {
			return errors.Wrapf(err, "create app spec for environment %q", spec.EnvName)
		}
	}

	// 创建应用基础数据
	if err := s.appStore.CreateApp(ctx, app); err != nil {
		return errors.Wrap(err, "create app")
	}

	return nil
}
