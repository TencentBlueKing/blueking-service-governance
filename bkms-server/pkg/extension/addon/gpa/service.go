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

package gpa

import (
	"context"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// 服务层错误定义
var (
	// ErrCRNotFound 集群中对应的 GPA CR 不存在
	ErrCRNotFound = errors.New("gpa CR not found in cluster")
	// ErrComponentNotInstalled 环境所在集群未安装 GPA 组件
	ErrComponentNotInstalled = errors.New("gpa component is not installed in cluster")
	// ErrFederationNotSupported 联邦环境本期不支持 GPA
	ErrFederationNotSupported = errors.New("gpa is not supported in federation environment")
)

// GPAService 负责将 GPA 配置下发为 GeneralPodAutoscaler CR 并管理其生命周期。
// 直接操作目标集群的 K8s ApiServer，状态完全以集群中的 CR 为准，不使用 DB 存储状态。
type GPAService struct {
	appModelStore appmodel.AppModelStore
}

// NewGPAService 创建 GPA 下发服务
func NewGPAService(appModelStore appmodel.AppModelStore) *GPAService {
	return &GPAService{
		appModelStore: appModelStore,
	}
}

// Apply 将 GPA 配置下发到目标集群（幂等 Upsert）。
func (s *GPAService) Apply(ctx context.Context, env *bkmsenv.Environment, config *GPAConfig) error {
	if env != nil && env.Cluster.IsFederation {
		return ErrFederationNotSupported
	}

	scaleTargetName, err := s.resolveScaleTargetName(ctx, config.AppID)
	if err != nil {
		return err
	}

	k8sClient, err := s.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		// discovery 阶段无法解析 GeneralPodAutoscaler 资源类型，说明集群未安装 GPA 组件
		if isComponentNotInstalledErr(err) {
			return errors.Wrapf(ErrComponentNotInstalled, "cluster %s", env.Cluster.ClusterID)
		}
		return errors.Wrap(err, "create k8s client for gpa")
	}

	manifest := buildGPAManifest(config, env.WorkspaceID, env.Name, scaleTargetName)
	if _, err = k8sClient.Upsert(ctx, env.Cluster.Namespace, manifest, metav1.PatchOptions{}); err != nil {
		// discovery 缓存可能过期：缓存中仍有 GPA 的 GVR 信息，但 CRD 实际已被卸载，
		// apiserver 会返回 404 "the server could not find the requested resource"。
		if isResourceTypeNotFoundErr(err) {
			return errors.Wrapf(ErrComponentNotInstalled, "cluster %s", env.Cluster.ClusterID)
		}
		return errors.Wrap(err, "upsert gpa CR to k8s")
	}

	return nil
}

// Get 回查集群中指定 GPA CR 的运行状态
func (s *GPAService) Get(ctx context.Context, env *bkmsenv.Environment, name string) (*GPAStatus, error) {
	clusterClient, err := s.NewClusterClient(env.Cluster.ClusterID)
	if err != nil {
		return nil, err
	}
	return clusterClient.GetStatus(ctx, env.Cluster.Namespace, name)
}

// ListByEnv 列出指定环境下所有 GPA CR 的运行状态
func (s *GPAService) ListByEnv(ctx context.Context, env *bkmsenv.Environment) ([]*GPAStatus, error) {
	k8sClient, err := s.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "create k8s client for gpa")
	}

	labelSelector := buildLabelSelector(env.WorkspaceID, env.Name)
	list, err := k8sClient.List(ctx, env.Cluster.Namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, errors.Wrap(err, "list gpa CRs from k8s")
	}

	return parseGPAStatusListFromUnstructured(list)
}

// Delete 删除集群中指定的 GPA CR
func (s *GPAService) Delete(ctx context.Context, env *bkmsenv.Environment, name string) error {
	k8sClient, err := s.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		return errors.Wrap(err, "create k8s client for gpa")
	}

	if err = k8sClient.Delete(ctx, env.Cluster.Namespace, name, metav1.DeleteOptions{}); err != nil {
		if errors.Is(err, k8sclient.ErrResourceNotFound) {
			return ErrCRNotFound
		}
		return errors.Wrap(err, "delete gpa CR from k8s")
	}

	return nil
}

// ClusterClient 单个集群的 GPA CR 客户端。
//
// 创建时解析一次 GVR（含 discovery 与 Redis 缓存查询），之后可在同集群多个 namespace 的
// 查询间复用，避免逐环境重复解析。
type ClusterClient struct {
	cli *k8sclient.Client
}

// NewClusterClient 创建单集群 GPA CR 客户端。
//
// Args:
//   - clusterID BCS / 本地集群 ID
//
// Returns:
//   - 集群 GPA CR 客户端
//   - error，集群未安装 GPA 组件时为 ErrComponentNotInstalled
func (s *GPAService) NewClusterClient(clusterID string) (*ClusterClient, error) {
	cli, err := s.newK8sClient(clusterID)
	if err != nil {
		// discovery 阶段无法解析 GeneralPodAutoscaler 资源类型，说明集群未安装 GPA 组件
		if isComponentNotInstalledErr(err) {
			return nil, errors.Wrapf(ErrComponentNotInstalled, "cluster %s", clusterID)
		}
		return nil, errors.Wrap(err, "create k8s client for gpa")
	}
	return &ClusterClient{cli: cli}, nil
}

// GetStatus 回查指定 namespace 下 GPA CR 的运行状态；CR 不存在时返回 ErrCRNotFound。
func (c *ClusterClient) GetStatus(ctx context.Context, namespace, name string) (*GPAStatus, error) {
	obj, err := c.cli.Get(ctx, namespace, name, metav1.GetOptions{})
	if err != nil {
		if errors.Is(err, k8sclient.ErrResourceNotFound) {
			return nil, ErrCRNotFound
		}
		return nil, errors.Wrap(err, "get gpa CR from k8s")
	}

	return parseGPAStatusFromUnstructured(obj)
}

// resolveScaleTargetName 解析 scaleTargetRef 指向的工作负载名（与应用工作负载名一致，见 workload builder）。
func (s *GPAService) resolveScaleTargetName(ctx context.Context, appID string) (string, error) {
	appModel, err := s.appModelStore.GetAppModel(ctx, appID)
	if err != nil {
		return "", errors.Wrapf(err, "get app model for app %s", appID)
	}
	if appModel.Workload.Name == "" {
		return "", errors.Errorf("app %s has empty workload name", appID)
	}
	return appModel.Workload.Name, nil
}

// newK8sClient 创建 GeneralPodAutoscaler CR 的 K8s 客户端
func (s *GPAService) newK8sClient(clusterID string) (*k8sclient.Client, error) {
	clusterCfg := cluster.NewConfig(clusterID)

	resGVR, err := discovery.GetGroupVersionResource(clusterCfg, gpaKind, gpaGroupVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "get GVR for GeneralPodAutoscaler in cluster %s", clusterID)
	}

	return k8sclient.NewWithGVR(clusterCfg, *resGVR), nil
}

// isComponentNotInstalledErr 判定 discovery 阶段的错误是否源于集群未安装 GPA 组件
// （GeneralPodAutoscaler CRD / 其所属 group 未注册）。仅用于 newK8sClient 的 discovery 结果。
//
// discovery 层已将"目标资源类型未注册"归一为 discovery.ErrKindNotFound（覆盖 group 整体未注册
// 与 group 内缺少目标 Kind 两种形态），此处用 errors.Is 精确判定，
// 避免把集群不通、鉴权失败、namespace 缺失等其他错误误判为组件未安装。
func isComponentNotInstalledErr(err error) bool {
	return errors.Is(err, discovery.ErrKindNotFound)
}

// isResourceTypeNotFoundErr 判定 K8s 操作返回的 NotFound 是否源于资源类型（CRD）不存在，
// 而非具体资源实例不存在。
//
// 当 discovery 缓存过期（Redis 缓存中仍保有 GVR 信息但 CRD 实际已卸载）时，
// newK8sClient 正常返回，但后续 Patch/Upsert 操作会收到 apiserver 的 404 响应
// （"the server could not find the requested resource"）。
//
// 区分方式：资源类型不存在时 StatusError.Details.Name 为空（整个 resource type 未知），
// 而具体实例不存在（如 namespace 缺失）时 Details.Name 不为空。
func isResourceTypeNotFoundErr(err error) bool {
	var statusErr *k8serrors.StatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	status := statusErr.ErrStatus
	if status.Reason != metav1.StatusReasonNotFound {
		return false
	}
	// Details.Name 为空表示 apiserver 无法识别资源类型本身（CRD 不存在）
	return status.Details == nil || status.Details.Name == ""
}

// buildLabelSelector 构建 label selector 字符串，按 workspaceID + envName 过滤
func buildLabelSelector(workspaceID, envName string) string {
	return LabelKeyWorkspaceID + "=" + workspaceID + "," + LabelKeyEnvName + "=" + envName
}
