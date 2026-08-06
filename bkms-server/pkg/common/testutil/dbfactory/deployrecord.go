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

package dbfactory

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/onsi/gomega"
	helmrelease "helm.sh/helm/v3/pkg/release"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
)

// AppModelDeployRecordOpts 定义创建 AppModel 部署记录时的可覆盖字段
type AppModelDeployRecordOpts struct {
	TrafficLaneName string
	Status          appmodeldeploy.Status
	ImageTag        string
	ClusterID       string
	Namespace       string
	LabelSelector   map[string]string
}

// AppModelDeployRecord 创建并持久化一条 AppModel 部署记录，返回部署 ID
func AppModelDeployRecord(
	ctx context.Context,
	store appmodeldeploy.RecordStore,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	opts *AppModelDeployRecordOpts,
) string {
	if opts == nil {
		opts = &AppModelDeployRecordOpts{}
	}
	status := opts.Status
	if status == "" {
		status = appmodeldeploy.StatusDeployed
	}
	imageTag := opts.ImageTag
	if imageTag == "" {
		imageTag = "v1"
	}
	clusterID := opts.ClusterID
	if clusterID == "" {
		clusterID = "c-" + stringx.Random(6)
	}
	namespace := opts.Namespace
	if namespace == "" {
		namespace = "ns-" + stringx.Random(6)
	}
	deployID, err := store.Create(ctx, &appmodeldeploy.Record{
		WorkspaceID:     app.WorkspaceID,
		AppID:           app.ID,
		EnvName:         env.Name,
		TrafficLaneName: opts.TrafficLaneName,
		Status:          status,
		ClusterID:       clusterID,
		Namespace:       namespace,
		LabelSelector:   opts.LabelSelector,
		ImageTag:        imageTag,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return deployID
}

// BuildAutoDeployRecordOpts 定义创建构建自动部署记录时的可覆盖字段
type BuildAutoDeployRecordOpts struct {
	TrafficLaneName string
	BuildID         string
	DeployID        string
	Stage           autodeploy.Stage
	Status          string
}

// BuildAutoDeployRecord 创建并持久化一条构建自动部署记录，用于测试
func BuildAutoDeployRecord(
	ctx context.Context,
	store autodeploy.RecordStore,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	opts *BuildAutoDeployRecordOpts,
) {
	if opts == nil {
		opts = &BuildAutoDeployRecordOpts{}
	}
	buildID := opts.BuildID
	if buildID == "" {
		buildID = "build-" + stringx.Random(6)
	}
	stage := opts.Stage
	if stage == "" {
		stage = autodeploy.StageDeploy
	}
	status := opts.Status
	if status == "" {
		status = string(appmodeldeploy.StatusDeploying)
	}
	err := store.Create(ctx, &autodeploy.Record{
		WorkspaceID:     app.WorkspaceID,
		AppID:           app.ID,
		AppType:         app.Type,
		EnvName:         env.Name,
		TrafficLaneName: opts.TrafficLaneName,
		BuildID:         buildID,
		DeployID:        opts.DeployID,
		Stage:           stage,
		Status:          status,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

// HelmDeployRecordOpts 定义创建 Helm 部署记录时的可覆盖字段
type HelmDeployRecordOpts struct {
	TrafficLaneName string
	Status          helmrelease.Status
	ImageTag        string
}

// HelmDeployRecord 创建并持久化一条 Helm 部署记录，用于测试
func HelmDeployRecord(
	ctx context.Context,
	store helmdeploy.RecordStore,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	opts *HelmDeployRecordOpts,
) {
	if opts == nil {
		opts = &HelmDeployRecordOpts{}
	}
	status := opts.Status
	if status == "" {
		status = helmrelease.StatusDeployed
	}
	imageTag := opts.ImageTag
	if imageTag == "" {
		imageTag = "helm-v1"
	}
	_, err := store.Create(ctx, &helmdeploy.Record{
		WorkspaceID:     app.WorkspaceID,
		AppID:           app.ID,
		EnvName:         env.Name,
		TrafficLaneName: opts.TrafficLaneName,
		Status:          status,
		Revision:        "1",
		ProjectCode:     "p-" + stringx.Random(6),
		ClusterID:       "c-" + stringx.Random(6),
		Namespace:       "ns-" + stringx.Random(6),
		ReleaseName:     app.Name,
		ChartName:       app.Name,
		Operator:        "test",
		ImageTag:        imageTag,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}
