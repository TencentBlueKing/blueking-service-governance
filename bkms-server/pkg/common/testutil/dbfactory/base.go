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

// Package dbfactory 中包含创建测试用的数据库对象的各类辅助函数，
// 开发者应该优先考虑将测试中的对象创建逻辑迁移至此包，以便复用和维护。
package dbfactory

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/go-playground/validator/v10"
	"github.com/onsi/gomega"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var validate = validator.New()

// Workspace 创建一个已持久化的测试用 Workspace，其名称和 ID 使用随机后缀避免冲突
func Workspace(ctx context.Context, store bkmsworkspace.WorkspaceStore) *bkmsworkspace.Workspace {
	workspaceName := "test-workspace-" + stringx.Random(6)
	ws := &bkmsworkspace.Workspace{
		ID:          workspaceName + stringx.Random(6),
		DisplayName: workspaceName,
		State:       bkmsworkspace.StateReady,
	}
	err := store.Create(ctx, ws)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return ws
}

// ApplicationOpts 定义创建测试 Application 时的可选参数。
type ApplicationOpts struct {
	// WorkspaceID 应用所属工作空间 ID；为空时使用随机 test-ws-*
	WorkspaceID string
	// ID 应用 ID；为空时使用随机值
	ID string
	// Name 应用名称；为空时使用随机值
	Name string
	// Type 应用类型；为空时保持默认空值
	Type string
}

// Application 创建一个已持久化的测试用 Application 对象。
func Application(ctx context.Context, store bkmsapp.ApplicationStore) *bkmsapp.Application {
	return ApplicationWithOpts(ctx, store, nil)
}

// ApplicationWithOpts 创建一个已持久化的测试用 Application 对象，并允许覆写关键字段。
func ApplicationWithOpts(
	ctx context.Context,
	store bkmsapp.ApplicationStore,
	opts *ApplicationOpts,
) *bkmsapp.Application {
	if opts == nil {
		opts = &ApplicationOpts{}
	}

	appName := "test-app-" + stringx.Random(6)
	app := &bkmsapp.Application{
		ID:          appName + stringx.Random(6),
		Name:        appName,
		WorkspaceID: "test-ws-" + stringx.Random(6),
	}
	if opts.ID != "" {
		app.ID = opts.ID
	}
	if opts.Name != "" {
		app.Name = opts.Name
	}
	if opts.WorkspaceID != "" {
		app.WorkspaceID = opts.WorkspaceID
	}
	if opts.Type != "" {
		app.Type = opts.Type
	}
	err := store.CreateApp(ctx, app)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return app
}

// Env 创建一个已持久化的测试用 Environment 对象，需要指定 Workspace ID
func Env(ctx context.Context, svc *env.EnvService, workspaceID string) *envmodel.Environment {
	return EnvWithOpts(ctx, svc, &EnvOpts{
		WorkspaceID: workspaceID,
		Type:        stringx.Random(10),
		Cluster: &envmodel.BizCluster{
			ProjectCode: stringx.Random(10),
			ClusterID:   stringx.Random(10),
			ClusterType: stringx.Random(10),
			Namespace:   stringx.Random(10),
		},
		Description: stringx.Random(10),
	})
}

// EnvOpts 定义创建 Environment 时的可选参数
type EnvOpts struct {
	// WorkspaceID 工作空间 ID（必填）
	WorkspaceID string
	// AppIDs 关联的应用 ID 列表
	AppIDs []string
	// Type 环境类型，默认为 "test"
	Type string
	// Cluster 集群信息；未传时保持为空
	Cluster *envmodel.BizCluster
	// Description 环境描述；未传时保持为空
	Description string
}

// EnvWithOpts 创建一个已持久化的测试用 Environment
func EnvWithOpts(
	ctx context.Context,
	svc *env.EnvService,
	opts *EnvOpts,
) *envmodel.Environment {
	if opts == nil {
		opts = &EnvOpts{}
	}
	envType := opts.Type
	if envType == "" {
		envType = "test"
	}

	envName := "test-env-" + stringx.Random(6)
	envData := &envmodel.Environment{
		Name:        envName,
		DisplayName: envName,
		Type:        envType,
		WorkspaceID: opts.WorkspaceID,
		AppIDs:      opts.AppIDs,
	}
	if opts.Cluster != nil {
		envData.Cluster = *opts.Cluster
	}
	if opts.Description != "" {
		envData.Description = opts.Description
	}

	envID, err := svc.Create(ctx, envData)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	envObj, err := svc.Get(ctx, envID)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return envObj
}

// FeatEnv 创建一个已持久化的测试用特性环境。
func FeatEnv(
	ctx context.Context,
	svc *env.EnvService,
	app *bkmsapp.Application,
	sourceEnv *envmodel.Environment,
) *envmodel.Environment {
	name := "test-feat-env-" + stringx.Random(6)
	envData := &envmodel.Environment{
		Name:        name,
		DisplayName: name,
		Type:        sourceEnv.Type,
		WorkspaceID: app.WorkspaceID,
		Kind:        envmodel.EnvironmentKindFeature,
		OwnerAppID:  app.ID,
		SourceEnvID: sourceEnv.ID,
		Cluster: envmodel.BizCluster{
			ProjectCode:  sourceEnv.Cluster.ProjectCode,
			ClusterID:    sourceEnv.Cluster.ClusterID,
			ClusterType:  sourceEnv.Cluster.ClusterType,
			Namespace:    name,
			IsFederation: sourceEnv.Cluster.IsFederation,
		},
		Creator: "dbfactory",
	}

	envID, err := svc.Create(ctx, envData)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	featEnv, err := svc.Get(ctx, envID)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return featEnv
}

// ComponentDefOpts 定义创建 ComponentDef 时的可选参数
type ComponentDefOpts struct {
	Properties []component.Property `validate:"-"`
	Patchers   []string             `validate:"-"`
	Specs      []string             `validate:"-"`
}

// CompDef 创建一个已持久化的测试用 ComponentDef 对象
// Args:
//   - opts: 可选参数, 用于定制 ComponentDef 的字段
func CompDef(
	ctx context.Context,
	store component.ComponentDefStore,
	opts *ComponentDefOpts,
) *component.ComponentDef {
	validateOpts(opts)

	patchers := opts.Patchers
	if patchers == nil {
		patchers = []string{"foo: var\n"}
	}

	// The default component for testing
	compDef := &component.ComponentDef{
		Name:        "TestComp-" + stringx.Random(6),
		Version:     "v1.0.0",
		Description: "A test component",
		Properties:  opts.Properties,
		Patchers:    patchers,
		Specs:       opts.Specs,
		Creator:     "admin",
		Updater:     "admin",
	}
	err := store.Create(ctx, compDef)
	gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	return compDef
}

func validateOpts(opts any) {
	err := validate.Struct(opts)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

// TrpcApplicationStores 创建 tRPC 应用所需的 store 集合
type TrpcApplicationStores struct {
	AppStore                  bkmsapp.ApplicationStore
	AppModelStore             appmodel.AppModelStore
	AppConfigFileStore        appcfg.AppConfigFileStore
	AppConfigFileVersionStore appcfg.AppConfigFileVersionStore
	BuildConfigStore          build.ConfigStore
}
