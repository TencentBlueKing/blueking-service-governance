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

package deploydiagnosis

import (
	"sort"
	"time"

	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
)

// diagnose 只根据快照跑规则集，不访问存储或集群
func (d *Diagnoser) diagnose(snap *rules.Snapshot) *Diagnosis {
	abnormal := snap.AbnormalInstances()
	out := &Diagnosis{
		GeneratedAt:    time.Now().UTC(),
		PartialReasons: append([]string{}, snap.PartialReasons...),
		Deploy:         snap.Deploy,
		TotalCount:     len(snap.Instances),
		AbnormalCount:  len(abnormal),
		Groups:         []*rules.Group{},
	}

	if snap.Deploy == nil {
		out.EmptyReason = EmptyReasonNoDeploy
		return out
	}
	// 滚动更新中的 Pending / ContainerCreating 实例会被判成异常，进行中一律不跑规则
	if snap.Deploy.Phase == deploystatus.DeployPhaseInProgress {
		out.EmptyReason = EmptyReasonDeploying
		return out
	}
	if len(abnormal) > 0 {
		out.Groups = groupAbnormalInstances(snap, abnormal)
		return out
	}

	// 没有异常实例时先跑阶段规则，都未命中才给空态
	if group := rules.StageRules.Diagnose(snap); group != nil {
		out.Groups = []*rules.Group{group}
		return out
	}
	out.EmptyReason = resolveEmptyReason(snap)
	return out
}

// resolveEmptyReason 没有任何结论时，区分「从未部署」「列不出实例」「没有实例」「都健康」
func resolveEmptyReason(snap *rules.Snapshot) string {
	// 一键构建停在构建阶段且之前没部署过时，没有可定位的实例
	if snap.Deploy.DeployID == "" {
		return EmptyReasonNoDeploy
	}
	// 有部署但没列 Pod（如 Helm），不能说成没有实例
	if !snap.InstancesListed {
		return EmptyReasonInstancesUnavailable
	}
	if len(snap.Instances) == 0 {
		return EmptyReasonNoInstance
	}
	return EmptyReasonAllHealthy
}

// groupAbnormalInstances 同一规则命中的实例并成一组，组按规则集顺序输出，组内按重启次数降序
func groupAbnormalInstances(snap *rules.Snapshot, abnormal []*rules.Instance) []*rules.Group {
	byRule := make(map[string]*rules.Group, len(rules.InstanceRules))
	for _, inst := range abnormal {
		hit := rules.InstanceRules.Diagnose(snap, inst)
		if group, ok := byRule[hit.RuleID]; ok {
			group.Instances = append(group.Instances, hit.Instances...)
			continue
		}
		byRule[hit.RuleID] = hit
	}

	// 按规则集顺序输出，避免 map 遍历顺序让 unknown 排到具体规则前面
	groups := make([]*rules.Group, 0, len(byRule))
	for _, rule := range rules.InstanceRules {
		group, ok := byRule[rule.ID()]
		if !ok {
			continue
		}
		// 前端默认展开第一项，重启次数高的放前面
		sort.SliceStable(group.Instances, func(i, j int) bool {
			return group.Instances[i].RestartCount > group.Instances[j].RestartCount
		})
		groups = append(groups, group)
	}
	return groups
}
