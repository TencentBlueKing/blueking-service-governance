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

// Package secret manages image pull secrets used when deploying applications.
package secret

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/credentials"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// ImagePullSecretSyncer 是镜像仓库拉取凭证的 Secret 同步器，用于管理集群内的 ImagePullSecret 资源
type ImagePullSecretSyncer struct {
	workspaceID string
	clusterID   string
	namespace   string
	appID       string
	buildCfg    *build.Config
}

// NewImagePullSecretSyncer 创建 ImagePullSecretSyncer 实例
func NewImagePullSecretSyncer(env *bkmsenv.Environment, appID string, buildCfg *build.Config) *ImagePullSecretSyncer {
	return &ImagePullSecretSyncer{
		workspaceID: env.WorkspaceID,
		clusterID:   env.Cluster.ClusterID,
		namespace:   env.Cluster.Namespace,
		appID:       appID,
		buildCfg:    buildCfg,
	}
}

// genImagePullSecretName 生成 bkms 下发的镜像仓库的 imagePullSecret 名称（工作空间级）
func genImagePullSecretName(workspaceID string) string {
	return fmt.Sprintf("bkms-image-pull-secret-%s", workspaceID)
}

// genAppImagePullSecretName 生成应用级的 imagePullSecret 名称
func genAppImagePullSecretName(workspaceID, appID string) string {
	return fmt.Sprintf("%s-%s", genImagePullSecretName(workspaceID), appID)
}

// hasCustomImageCredential 判断构建配置中是否包含应用自定义镜像凭证
func hasCustomImageCredential(buildCfg *build.Config) bool {
	if buildCfg == nil || buildCfg.Image == nil {
		return false
	}
	// 必须是用户名 & 密码都是有效值才会使用应用级的 ImagePullSecret
	return credentials.HasUserPass(buildCfg.Image.Username, buildCfg.Image.Password)
}

// ResolveImagePullSecretName 根据构建配置解析部署时应该引用的 imagePullSecret 名称
func ResolveImagePullSecretName(workspaceID, appID string, buildCfg *build.Config) string {
	if appID != "" && hasCustomImageCredential(buildCfg) {
		return genAppImagePullSecretName(workspaceID, appID)
	}
	return genImagePullSecretName(workspaceID)
}

// Sync 确保集群中经存在镜像拉取的 Secret，且其 Auth 信息需与数据库中保持一致
func (m *ImagePullSecretSyncer) Sync(ctx context.Context) error {
	// 1. 先确保对应的命名空间已经存在，如果不存在则需要报错
	if err := m.validateNamespace(ctx); err != nil {
		return err
	}

	client := k8sclient.NewWithGVR(cluster.NewConfig(m.clusterID), gvr.Secret)

	// 2. 获取镜像仓库凭证信息
	auths, err := m.genAuthsData(ctx)
	if err != nil {
		return errors.Wrapf(err, "get workspace %s image registry auths", m.workspaceID)
	}

	// 3. 生成镜像拉取 Secret 的 manifest
	manifest, err := m.genSecretManifest(auths)
	if err != nil {
		return errors.Wrapf(err, "generate workspace %s image pull secret manifest", m.workspaceID)
	}

	// 4. 下发 image-pull-secret 到集群内（更新 / 创建）
	if _, err = client.Upsert(ctx, m.namespace, manifest, metav1.PatchOptions{}); err != nil {
		return errors.Wrapf(err, "upsert workspace %s image pull secret", m.workspaceID)
	}
	return nil
}

// validateNamespace 检查命名空间是否已经存在
func (m *ImagePullSecretSyncer) validateNamespace(ctx context.Context) error {
	client := k8sclient.NewWithGVR(cluster.NewConfig(m.clusterID), gvr.NS)
	// 检查 k8s 命名空间是否存在
	if _, err := client.Get(ctx, "", m.namespace, metav1.GetOptions{}); err != nil {
		return errors.Wrapf(err, "get namespace: %s", m.namespace)
	}
	return nil
}

// genAuthsData 生成镜像仓库的凭证信息
func (m *ImagePullSecretSyncer) genAuthsData(ctx context.Context) (map[string]any, error) {
	store, err := registry.NewImageRegistryStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "get image registry store")
	}

	imageRegistries, err := store.List(ctx, m.workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "list image registry for workspace: %s", m.workspaceID)
	}

	auths := map[string]any{}
	for _, reg := range imageRegistries {
		auths[reg.Registry] = map[string]any{
			"username": reg.Username,
			"password": reg.Password,
		}
	}

	addImageConfigAuth(auths, m.buildCfg)
	return auths, nil
}

func addImageConfigAuth(auths map[string]any, buildCfg *build.Config) {
	if !hasCustomImageCredential(buildCfg) {
		return
	}
	auths[buildCfg.Image.Name] = map[string]any{
		"username": buildCfg.Image.Username,
		"password": buildCfg.Image.Password,
	}
}

// genSecretManifest 生成镜像拉取 Secret 的 manifest
func (m *ImagePullSecretSyncer) genSecretManifest(auths map[string]any) (map[string]any, error) {
	// 按指定格式生成 docker config json
	dockerCfgJson, err := json.Marshal(map[string]any{"auths": auths})
	if err != nil {
		return nil, errors.Wrapf(err, "marshal docker config json")
	}

	// 组装 secret manifest
	return map[string]any{
		"apiVersion": "v1",
		"kind":       k8skind.Secret,
		"metadata": map[string]any{
			"name": ResolveImagePullSecretName(m.workspaceID, m.appID, m.buildCfg),
		},
		"type": string(v1.SecretTypeDockerConfigJson),
		"data": map[string]any{
			// 需要将 docker config json 进行 base64 编码
			v1.DockerConfigJsonKey: base64.StdEncoding.EncodeToString(dockerCfgJson),
		},
	}, nil
}
