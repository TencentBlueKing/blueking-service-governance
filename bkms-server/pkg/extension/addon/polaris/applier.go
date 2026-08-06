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

// PolarisConfig CR 下发器负责根据应用、环境、配置和环境变量构建 CR，并写入目标集群。

package polaris

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
)

// polarisCRApplier 负责构建并向单个目标环境下发 PolarisConfig CR。
// 目前仅用于 patch 后的动态下发，其他情况通过正常部署流程下发及删除
type polarisCRApplier struct{}

func newPolarisCRApplier() *polarisCRApplier {
	return &polarisCRApplier{}
}

// apply 向指定环境下发 CR，仅返回资源构建或集群操作错误。
func (a *polarisCRApplier) apply(
	ctx context.Context,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	config *PolarisConfig,
	envVars map[string]string,
) error {
	manifest, err := a.buildCRManifest(app, env, config, envVars)
	if err != nil {
		return err
	}
	if err = a.upsertCR(ctx, env, manifest); err != nil {
		return errors.Wrapf(err, "apply polaris CR in env %s", env.Name)
	}
	return nil
}

// buildCRManifest 复用 workload 构建逻辑并提取 PolarisConfig CR。
func (a *polarisCRApplier) buildCRManifest(
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	config *PolarisConfig,
	envVars map[string]string,
) (map[string]any, error) {
	resources, err := buildExtraResources(app, env, config, envVars, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build polaris resources")
	}
	for i := range resources {
		if resources[i].GetKind() == polarisConfigCRKind {
			return resources[i].Object, nil
		}
	}
	return nil, errors.Errorf(
		"PolarisConfig CR not found in built resources for app %s config %s",
		app.ID,
		config.Name,
	)
}

func (a *polarisCRApplier) upsertCR(
	ctx context.Context,
	env *bkmsenv.Environment,
	manifest map[string]any,
) error {
	k8sClient, err := a.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		return errors.Wrap(err, "create k8s client for polaris CR")
	}
	if _, err = k8sClient.Upsert(ctx, env.Cluster.Namespace, manifest, metav1.PatchOptions{}); err != nil {
		return errors.Wrap(err, "upsert polaris CR to k8s")
	}
	return nil
}

func (a *polarisCRApplier) newK8sClient(clusterID string) (*k8sclient.Client, error) {
	clusterCfg := cluster.NewConfig(clusterID)
	resGVR, err := discovery.GetGroupVersionResource(clusterCfg, polarisConfigCRKind, polarisConfigCRAPIVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "get GVR for PolarisConfig in cluster %s", clusterID)
	}
	return k8sclient.NewWithGVR(clusterCfg, *resGVR), nil
}
