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

package batchrelease

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

var (
	// ErrRolloutNotInstalled 集群未安装 bcs-kourse-rollout（不存在 Rollout CRD），错误信息中携带集群返回的原因
	ErrRolloutNotInstalled = errors.New("rollout CRD is not installed in cluster")
	// ErrRolloutNotFound 集群中不存在指定的 Rollout CR
	ErrRolloutNotFound = errors.New("rollout not found")
)

// StatusPatch Rollout status 子资源的修改内容，nil 字段不修改
type StatusPatch struct {
	// Paused 对应 status.paused，取值见 PausedTrue / PausedFalse 等
	Paused *string
	// CurrentStepIndex 对应 status.stepStatus.currentStepIndex：
	// 当前值 - 1 为回退上一批，批次总数为跳过剩余批次，-1 为完全回滚
	CurrentStepIndex *int64
}

// RolloutClient 读写集群中的 Rollout CR（仅支持非联邦集群）
type RolloutClient struct {
	clusterID string
	cli       *k8sclient.Client
}

// NewRolloutClient 新建 Rollout CR 客户端
func NewRolloutClient(cfg *cluster.Config) *RolloutClient {
	return &RolloutClient{clusterID: cfg.ClusterID, cli: k8sclient.NewWithGVR(cfg, GVR)}
}

// Get 获取 Rollout CR，不存在时返回 ErrRolloutNotFound，集群未安装 CRD 时返回 ErrRolloutNotInstalled
func (c *RolloutClient) Get(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	obj, err := c.cli.Get(ctx, namespace, name, metav1.GetOptions{})
	if errors.Is(err, k8sclient.ErrResourceNotFound) {
		if installErr := c.checkInstalled(ctx, namespace); installErr != nil {
			return nil, installErr
		}
		return nil, c.notFound(namespace, name)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get rollout %s/%s in cluster %s", namespace, name, c.clusterID)
	}
	return obj, nil
}

// GetStatus 获取 Rollout CR 并解析其 status
func (c *RolloutClient) GetStatus(ctx context.Context, namespace, name string) (Status, error) {
	obj, err := c.Get(ctx, namespace, name)
	if err != nil {
		return Status{}, err
	}
	status, err := ParseStatus(obj.Object)
	if err != nil {
		return Status{}, errors.Wrapf(err, "rollout %s/%s in cluster %s", namespace, name, c.clusterID)
	}
	return status, nil
}

// Apply 创建或更新 Rollout CR（Server-Side Apply），返回更新后的 Rollout；manifest 通常由 BuildManifest 生成
func (c *RolloutClient) Apply(
	ctx context.Context, namespace string, manifest map[string]any,
) (*unstructured.Unstructured, error) {
	name, _, _ := unstructured.NestedString(manifest, "metadata", "name")
	obj, err := c.cli.Upsert(ctx, namespace, manifest, metav1.PatchOptions{})
	if err != nil {
		// SSA 在对象不存在时会直接创建，NotFound 只可能来自资源类型或命名空间不存在
		if k8serrors.IsNotFound(err) {
			if installErr := c.checkInstalled(ctx, namespace); installErr != nil {
				return nil, installErr
			}
		}
		return nil, errors.Wrapf(err, "apply rollout %s/%s in cluster %s", namespace, name, c.clusterID)
	}
	return obj, nil
}

// Delete 删除 Rollout CR；CR 不存在或集群未安装 CRD 时视为删除成功（本就没有遗留的 Rollout）
func (c *RolloutClient) Delete(ctx context.Context, namespace, name string) error {
	if err := c.cli.Delete(ctx, namespace, name, metav1.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "delete rollout %s/%s in cluster %s", namespace, name, c.clusterID)
	}
	return nil
}

// UpdateStatus 修改 Rollout 的 status 子资源，用于继续、暂停、回退、跳过与回滚等手动操作
func (c *RolloutClient) UpdateStatus(ctx context.Context, namespace, name string, patch StatusPatch) error {
	status := map[string]any{}
	if patch.Paused != nil {
		status["paused"] = *patch.Paused
	}
	if patch.CurrentStepIndex != nil {
		// Kourse 以 pauseStartTime != nil 判断当前批次是否已暂停过；Merge Patch 修改批次序号时需一并清空，
		// 否则从已暂停批次（含 pauseStartTime）回退上一批并置 paused=false 后，会被当成本批已解除暂停而直接跳回下一批
		status["stepStatus"] = map[string]any{
			"currentStepIndex": *patch.CurrentStepIndex,
			"pauseStartTime":   nil,
		}
	}
	if len(status) == 0 {
		return errors.New("empty rollout status patch")
	}
	// Merge Patch 在 stepStatus 等父节点缺失时也能写入，JSON Patch 的 replace 则会失败
	data, err := json.Marshal(map[string]any{"status": status})
	if err != nil {
		return errors.Wrap(err, "marshal rollout status patch")
	}

	_, err = c.cli.Patch(ctx, namespace, name, types.MergePatchType, data, metav1.PatchOptions{}, "status")
	if k8serrors.IsNotFound(err) {
		if installErr := c.checkInstalled(ctx, namespace); installErr != nil {
			return installErr
		}
		return c.notFound(namespace, name)
	}
	if err != nil {
		return errors.Wrapf(err, "patch rollout %s/%s status in cluster %s", namespace, name, c.clusterID)
	}
	return nil
}

// checkInstalled 在请求返回 NotFound 后确认集群是否安装了 Rollout CRD：apiserver 对不存在的资源类型
// 与不存在的对象都返回 404，因此 List 同命名空间的 Rollout 来区分（命名空间不存在时 List 仍返回空列表）。
// CRD 存在时返回 nil。
func (c *RolloutClient) checkInstalled(ctx context.Context, namespace string) error {
	_, err := c.cli.PaginateList(ctx, namespace, 1, 1, metav1.ListOptions{})
	if k8serrors.IsNotFound(err) {
		return errors.Wrapf(ErrRolloutNotInstalled, "cluster %s: %v", c.clusterID, err)
	}
	return errors.Wrapf(err, "check rollout CRD in cluster %s", c.clusterID)
}

func (c *RolloutClient) notFound(namespace, name string) error {
	return errors.Wrapf(ErrRolloutNotFound, "rollout %s/%s in cluster %s", namespace, name, c.clusterID)
}
