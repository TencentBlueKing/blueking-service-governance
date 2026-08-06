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
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// portPoolSpec 对应 PortPool CR 的 spec 段，用于 mapstructure 解码
type portPoolSpec struct {
	PoolItems []PoolItem `mapstructure:"poolItems"`
}

// portPoolStatus 对应 PortPool CR 的 status 段，用于 mapstructure 解码
type portPoolStatus struct {
	Status    string               `mapstructure:"status"`
	PoolItems []poolItemStatusItem `mapstructure:"poolItems"`
}

// poolItemStatusItem 对应 status.poolItems 中的单个 item
type poolItemStatusItem struct {
	ItemName string `mapstructure:"itemName"`
	Status   string `mapstructure:"status"`
	Message  string `mapstructure:"message"`
}

// parsePortPoolFromUnstructured 将 K8s Unstructured 转为 PortPoolConfig
func parsePortPoolFromUnstructured(obj *unstructured.Unstructured) (*PortPoolConfig, error) {
	config := &PortPoolConfig{
		Name:        obj.GetName(),
		WorkspaceID: obj.GetLabels()[LabelKeyWorkspaceID],
		EnvName:     obj.GetLabels()[LabelKeyEnvName],
	}

	spec, ok := obj.Object["spec"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid PortPool CR %s: spec is not a map", config.Name)
	}

	var result portPoolSpec
	if err := mapstructure.Decode(spec, &result); err != nil {
		return nil, fmt.Errorf("invalid PortPool CR %s: %w", config.Name, err)
	}

	config.PoolItems = result.PoolItems
	if config.PoolItems == nil {
		config.PoolItems = make([]PoolItem, 0)
	}

	// 解析 status 段
	if statusMap, ok := obj.Object["status"].(map[string]any); ok {
		var statusResult portPoolStatus
		if err := mapstructure.Decode(statusMap, &statusResult); err == nil {
			config.Status = statusResult.Status
			// 如果 PortPool CR 正在删除，则状态为 Deleting
			if obj.GetDeletionTimestamp() != nil {
				config.Status = portPoolStatusDeleting
			}
			// 将 status 中的 item 状态合并到 PoolItems
			statusItemMap := lo.SliceToMap(
				statusResult.PoolItems,
				func(item poolItemStatusItem) (string, poolItemStatusItem) {
					return item.ItemName, item
				},
			)
			for i := range config.PoolItems {
				if si, found := statusItemMap[config.PoolItems[i].ItemName]; found {
					config.PoolItems[i].Status = PoolItemStatus{Status: si.Status, Message: si.Message}
				}
			}
		}
	}

	return config, nil
}

// parsePortPoolListFromUnstructured 批量转换 UnstructuredList 为 PortPoolConfig 列表
func parsePortPoolListFromUnstructured(list *unstructured.UnstructuredList) ([]*PortPoolConfig, error) {
	configs := make([]*PortPoolConfig, 0, len(list.Items))
	for i := range list.Items {
		config, err := parsePortPoolFromUnstructured(&list.Items[i])
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// buildPortPoolManifest 将 PortPoolConfig 转换为 k8s Unstructured manifest
func buildPortPoolManifest(config *PortPoolConfig) map[string]any {
	poolItems := make([]any, 0, len(config.PoolItems))
	for _, item := range config.PoolItems {
		poolItem := map[string]any{
			"itemName":      item.ItemName,
			"startPort":     item.StartPort,
			"endPort":       item.EndPort,
			"segmentLength": item.SegmentLength,
		}
		if len(item.LoadBalancerIDs) > 0 {
			poolItem["loadBalancerIDs"] = item.LoadBalancerIDs
		}
		if item.Protocol != "" {
			poolItem["protocol"] = item.Protocol
		}
		if item.External != "" {
			poolItem["external"] = item.External
		}
		poolItems = append(poolItems, poolItem)
	}

	return map[string]any{
		"apiVersion": portPoolGroupVersion,
		"kind":       portPoolKind,
		"metadata": map[string]any{
			"name": config.Name,
			"labels": map[string]any{
				LabelKeyWorkspaceID: config.WorkspaceID,
				LabelKeyEnvName:     config.EnvName,
			},
		},
		"spec": map[string]any{
			"poolItems": poolItems,
		},
	}
}
