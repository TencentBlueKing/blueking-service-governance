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
	"encoding/json"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// CRApplier 负责构建并向单个目标环境下发或更新 Polaris 资源。
//
// on_deploy 模式下仅用于 patch 后的动态下发，其他情况通过正常部署流程下发及删除；
// immediate 模式下还负责下发配套的 Service，以及配置离域或删除时清理集群资源，
// 因为这类配置不依赖部署流程，无法借助部署时的资源差异清理。
type CRApplier struct{}

// NewCRApplier 创建 PolarisConfig CR 下发器。
func NewCRApplier() *CRApplier {
	return &CRApplier{}
}

// Apply 向指定环境下发 Polaris 资源。
// on_deploy 只 upsert PolarisConfig CR（配套 Service 由部署流程管理）；
// immediate 同时 upsert 配套 Service，因为该模式不走部署流程。
func (a *CRApplier) Apply(
	ctx context.Context,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	config *PolarisConfig,
	envVars map[string]string,
) error {
	resources, err := buildExtraResources(app, env, config, envVars, nil)
	if err != nil {
		return errors.Wrap(err, "build polaris resources")
	}

	wanted := applyKinds(config)
	clusterCfg := cluster.NewConfig(env.Cluster.ClusterID)
	for idx := range resources {
		obj := resources[idx]
		if _, ok := wanted[obj.GetKind()]; !ok {
			continue
		}
		k8sClient, clientErr := newK8sClientForObject(clusterCfg, obj.GetAPIVersion(), obj.GetKind())
		if clientErr != nil {
			return clientErr
		}
		if _, err = k8sClient.Upsert(
			ctx, env.Cluster.Namespace, obj.Object, metav1.PatchOptions{},
		); err != nil {
			return errors.Wrapf(err, "apply polaris %s %s in env %s", obj.GetKind(), obj.GetName(), env.Name)
		}
	}
	return nil
}

// applyKinds 返回本次应 upsert 的资源类型。
//
//   - on_deploy：只下发 PolarisConfig CR， 对应的 Service 会在部署流程中下发
//   - immediate：同时下发 PolarisConfig CR 与配套 Service，因为该模式不走部署流程。
func applyKinds(config *PolarisConfig) map[string]struct{} {
	wanted := map[string]struct{}{polarisConfigCRKind: {}}
	if config.IsImmediateRegister() {
		wanted[k8skind.SVC] = struct{}{}
	}
	return wanted
}

// DeleteResources 从指定环境删除该配置的 PolarisConfig CR 与配套 Service。
// 资源不存在时视为成功，因此重复删除是安全的。
func (a *CRApplier) DeleteResources(
	ctx context.Context,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	config *PolarisConfig,
) error {
	crName, serviceName := PolarisResourceNames(app.Name, config.Name)
	targets := []struct {
		apiVersion string
		kind       string
		name       string
	}{
		{polarisConfigCRAPIVersion, polarisConfigCRKind, crName},
		{corev1.SchemeGroupVersion.String(), k8skind.SVC, serviceName},
	}

	clusterCfg := cluster.NewConfig(env.Cluster.ClusterID)
	for _, target := range targets {
		k8sClient, err := newK8sClientForObject(clusterCfg, target.apiVersion, target.kind)
		if err != nil {
			return err
		}
		if err = k8sClient.Delete(
			ctx, env.Cluster.Namespace, target.name, metav1.DeleteOptions{},
		); err != nil {
			return errors.Wrapf(err, "delete polaris %s %s in env %s", target.kind, target.name, env.Name)
		}
	}
	return nil
}

type jsonPatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

func buildWeightPatch(serviceName string, weight int32) ([]byte, error) {
	return json.Marshal([]jsonPatchOperation{
		{Op: "test", Path: "/spec/services/0/name", Value: serviceName},
		{Op: "add", Path: "/spec/services/0/weight", Value: int64(weight)},
	})
}

// PatchWeight 仅更新现有 PolarisConfig CR 的服务权重，不修改其他配置字段。
func (a *CRApplier) PatchWeight(
	ctx context.Context,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	config *PolarisConfig,
	weight int32,
) error {
	crName, serviceName := PolarisResourceNames(app.Name, config.Name)
	patch, err := buildWeightPatch(serviceName, weight)
	if err != nil {
		return errors.Wrap(err, "build polaris CR weight patch")
	}

	k8sClient, err := newK8sClientForObject(
		cluster.NewConfig(env.Cluster.ClusterID), polarisConfigCRAPIVersion, polarisConfigCRKind,
	)
	if err != nil {
		return errors.Wrap(err, "create k8s client for polaris CR")
	}
	if _, err = k8sClient.Patch(
		ctx,
		env.Cluster.Namespace,
		crName,
		types.JSONPatchType,
		patch,
		metav1.PatchOptions{},
	); err != nil {
		return errors.Wrap(err, "patch polaris CR weight")
	}
	return nil
}

func newK8sClientForObject(clusterCfg *cluster.Config, apiVersion, kind string) (*k8sclient.Client, error) {
	if apiVersion == "" || kind == "" {
		return nil, errors.Errorf("invalid polaris resource: apiVersion %q kind %q", apiVersion, kind)
	}
	resGVR, err := discovery.GetGroupVersionResource(clusterCfg, kind, apiVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "get GVR for %s in cluster %s", kind, clusterCfg.ClusterID)
	}
	return k8sclient.NewWithGVR(clusterCfg, *resGVR), nil
}
