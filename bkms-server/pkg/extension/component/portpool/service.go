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

package portpool

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
)

const (
	// portPoolKind PortPool CR 的 Kind
	portPoolKind = "PortPool"
	// portPoolGroupVersion PortPool CR 的 API GroupVersion
	portPoolGroupVersion = "networkextension.bkbcs.tencent.com/v1"

	// --- PortPool CR labels ---

	// LabelKeyWorkspaceID 标记所属工作空间 ID
	LabelKeyWorkspaceID = "io.tencent.bkms.workspace-id"
	// LabelKeyEnvName 标记所属环境名称
	LabelKeyEnvName = "io.tencent.bkms.env-name"

	portPoolStatusDeleting = "Deleting"
)

// 错误定义
var (
	// ErrConfigNotFound 端口池配置不存在
	ErrConfigNotFound = errors.New("portpool config not found")
	// ErrConfigNameExists 端口池配置名称已存在
	ErrConfigNameExists = errors.New("portpool config name already exists")
)

// PortPoolService 端口池服务，直接操作 K8s ApiServer
type PortPoolService struct{}

// NewPortPoolService 创建端口池服务
func NewPortPoolService() *PortPoolService {
	return &PortPoolService{}
}

// Create 创建端口池
func (s *PortPoolService) Create(ctx context.Context, config *PortPoolConfig, env *bkmsenv.Environment) error {
	fillEmptyItemNames(config.PoolItems)
	if err := validateItemNamesUnique(config.PoolItems); err != nil {
		return errors.Wrap(err, "validate portpool")
	}

	k8sClient, err := s.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		return errors.Wrap(err, "create k8s client for portpool")
	}

	manifest := buildPortPoolManifest(config)
	if _, err = k8sClient.Upsert(ctx, env.Cluster.Namespace, manifest, metav1.PatchOptions{}); err != nil {
		return errors.Wrap(err, "upsert portpool CR to k8s")
	}

	return nil
}

// Get 获取端口池配置
func (s *PortPoolService) Get(ctx context.Context, env *bkmsenv.Environment, name string) (*PortPoolConfig, error) {
	k8sClient, err := s.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "create k8s client for portpool")
	}

	obj, err := k8sClient.Get(ctx, env.Cluster.Namespace, name, metav1.GetOptions{})
	if err != nil {
		if errors.Is(err, k8sclient.ErrResourceNotFound) {
			return nil, ErrConfigNotFound
		}
		return nil, errors.Wrap(err, "get portpool CR from k8s")
	}

	return parsePortPoolFromUnstructured(obj)
}

// ListByEnv 获取指定环境下的端口池配置列表
func (s *PortPoolService) ListByEnv(ctx context.Context, env *bkmsenv.Environment) ([]*PortPoolConfig, error) {
	k8sClient, err := s.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "create k8s client for portpool")
	}

	// 使用 label selector 按 workspaceID + envName 过滤
	labelSelector := buildLabelSelector(env.WorkspaceID, env.Name)
	list, err := k8sClient.List(ctx, env.Cluster.Namespace, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, errors.Wrap(err, "list portpool CRs from k8s")
	}

	return parsePortPoolListFromUnstructured(list)
}

// UpdateResult 更新端口池的返回结果
type UpdateResult struct {
	// 更新前的端口池配置
	Before *PortPoolConfig
	// 更新后的端口池配置
	After *PortPoolConfig
}

// Update 更新端口池（全量替换 poolItems），返回更新前后的配置
func (s *PortPoolService) Update(
	ctx context.Context, env *bkmsenv.Environment, name string, newItems []PoolItem,
) (*UpdateResult, error) {
	// 获取现有配置
	existingConfig, err := s.Get(ctx, env, name)
	if err != nil {
		return nil, errors.Wrap(err, "get portpool config for update")
	}

	// 保存更新前的 poolItems 快照
	oldItems := make([]PoolItem, len(existingConfig.PoolItems))
	copy(oldItems, existingConfig.PoolItems)

	// 自动填充空 itemName，传入已有 items 避免复用被删除 item 的编号
	fillEmptyItemNames(newItems, existingConfig.PoolItems)

	// 校验 itemName 不重复
	if err = validateItemNamesUnique(newItems); err != nil {
		return nil, errors.Wrap(err, "validate poolItems")
	}

	// 校验 item 更新限制
	if err = validateItemUpdate(oldItems, newItems); err != nil {
		return nil, errors.Wrap(err, "validate poolItems")
	}

	// 替换 poolItems
	existingConfig.PoolItems = newItems

	// Upsert 更新后的配置到 K8s
	k8sClient, err := s.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "create k8s client for portpool")
	}

	manifest := buildPortPoolManifest(existingConfig)
	if _, err = k8sClient.Upsert(ctx, env.Cluster.Namespace, manifest, metav1.PatchOptions{}); err != nil {
		return nil, errors.Wrap(err, "upsert portpool CR to k8s")
	}

	log.Infof(
		ctx, "updated PortPool CR %s in cluster %s namespace %s",
		name, env.Cluster.ClusterID, env.Cluster.Namespace,
	)

	before := *existingConfig
	before.PoolItems = oldItems
	return &UpdateResult{Before: &before, After: existingConfig}, nil
}

// Delete 删除端口池
func (s *PortPoolService) Delete(ctx context.Context, env *bkmsenv.Environment, name string) error {
	k8sClient, err := s.newK8sClient(env.Cluster.ClusterID)
	if err != nil {
		return errors.Wrap(err, "create k8s client for portpool")
	}

	if err = k8sClient.Delete(ctx, env.Cluster.Namespace, name, metav1.DeleteOptions{}); err != nil {
		if errors.Is(err, k8sclient.ErrResourceNotFound) {
			return ErrConfigNotFound
		}
		return errors.Wrap(err, "delete portpool CR from k8s")
	}

	return nil
}

// newK8sClient 创建 PortPool CR 的 K8s 客户端
func (s *PortPoolService) newK8sClient(clusterID string) (*k8sclient.Client, error) {
	clusterCfg := cluster.NewConfig(clusterID)

	resGVR, err := discovery.GetGroupVersionResource(clusterCfg, portPoolKind, portPoolGroupVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "get GVR for PortPool in cluster %s", clusterID)
	}

	return k8sclient.NewWithGVR(clusterCfg, *resGVR), nil
}

// validateItemNamesUnique 校验 poolItems 中 itemName 是否唯一
func validateItemNamesUnique(items []PoolItem) error {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, exists := seen[item.ItemName]; exists {
			return errors.Errorf("poolItem itemName %s already exists", item.ItemName)
		}
		seen[item.ItemName] = struct{}{}
	}
	return nil
}

// validateItemUpdate 校验新 poolItems 中与旧 item 同名的 endPort 不缩小，startPort 不能改变
func validateItemUpdate(oldItems, newItems []PoolItem) error {
	oldMap := lo.SliceToMap(oldItems, func(item PoolItem) (string, PoolItem) {
		return item.ItemName, item
	})
	for _, newItem := range newItems {
		if oldItem, found := oldMap[newItem.ItemName]; found {
			if newItem.EndPort < oldItem.EndPort {
				return errors.Errorf("endPort %d of poolItem %s must not be less than current endPort %d",
					newItem.EndPort, newItem.ItemName, oldItem.EndPort)
			}

			if newItem.StartPort != oldItem.StartPort {
				return errors.Errorf("startPort %d of poolItem %s must not be changed",
					newItem.StartPort, newItem.ItemName)
			}
		}
	}
	return nil
}

// buildLabelSelector 构建 label selector 字符串，按 workspaceID + envName 过滤
func buildLabelSelector(workspaceID, envName string) string {
	return fmt.Sprintf("%s=%s,%s=%s",
		LabelKeyWorkspaceID, workspaceID,
		LabelKeyEnvName, envName,
	)
}

const itemNamePrefix = "item-"

// fillEmptyItemNames 为 itemName 为空的 PoolItem 自动填充名称
// 先扫描 items 和 existingItems 中所有已有 itemName 的最大编号，再从 max+1 开始递增分配。
// existingItems 为可选参数，用于在 Update 场景下传入已有的 items，避免复用被删除 item 的编号。
// NOTE: PoolItem 正常情况下不会超过 10 个
func fillEmptyItemNames(items []PoolItem, existingItems ...[]PoolItem) {
	maxIdx := maxItemIndex(items)
	for _, existing := range existingItems {
		if idx := maxItemIndex(existing); idx > maxIdx {
			maxIdx = idx
		}
	}
	for i := range items {
		if items[i].ItemName == "" {
			maxIdx++
			items[i].ItemName = fmt.Sprintf("%s%d", itemNamePrefix, maxIdx)
		}
	}
}

// maxItemIndex 返回 items 中符合 item-%d 格式的最大编号，若无则返回 -1
func maxItemIndex(items []PoolItem) int {
	maxIdx := -1
	for _, item := range items {
		if strings.HasPrefix(item.ItemName, itemNamePrefix) {
			idxStr := strings.TrimPrefix(item.ItemName, itemNamePrefix)
			if idx, err := strconv.Atoi(idxStr); err == nil && idx > maxIdx {
				maxIdx = idx
			}
		}
	}
	return maxIdx
}
