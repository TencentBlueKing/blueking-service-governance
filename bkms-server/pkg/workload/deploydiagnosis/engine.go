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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
)

const (
	// previousLogHint 取到重启前日志时的日志区说明
	previousLogHint = "以下为进程退出前的最近一次重启日志。"
	// currentLogHint 取到当前容器日志时的日志区说明
	currentLogHint = "以下为当前容器日志。"
)

// diagnose 只根据快照跑规则集，不访问 registry
func (d *Diagnoser) diagnose(snap *rules.Snapshot) *serializer.DeployDiagnosisOutputObj {
	out := emptyDiagnosis(snap)
	abnormal := snap.AbnormalInstances()
	// 没有异常实例时先跑阶段规则；都未命中才给空态
	if len(abnormal) == 0 {
		if group, ok := rules.StageRules.Run(snap); ok {
			out.Groups = []*serializer.DiagnosisGroupObj{group}
			return out
		}
		if len(snap.Instances) == 0 {
			out.EmptyReason = serializer.EmptyReasonNoInstance
			return out
		}
		out.EmptyReason = serializer.EmptyReasonAllHealthy
		return out
	}

	// 同一规则并成一组，再按规则集顺序输出，避免先遍历到的 unknown 排到前面
	groups := make(map[string]*serializer.DiagnosisGroupObj, len(rules.InstanceRules))
	for _, inst := range abnormal {
		hit := rules.InstanceRules.Run(snap, inst)
		group, ok := groups[hit.RuleID]
		if !ok {
			groups[hit.RuleID] = hit
			continue
		}
		group.Instances = append(group.Instances, hit.Instances...)
	}
	for _, rule := range rules.InstanceRules {
		group, ok := groups[rule.ID()]
		if !ok {
			continue
		}
		// 前端默认展示第一项，重启次数高的放前面
		sort.SliceStable(group.Instances, func(i, j int) bool {
			return group.Instances[i].RestartCount > group.Instances[j].RestartCount
		})
		out.Groups = append(out.Groups, group)
	}
	return out
}

func emptyDiagnosis(snap *rules.Snapshot) *serializer.DeployDiagnosisOutputObj {
	return &serializer.DeployDiagnosisOutputObj{
		GeneratedAt:    time.Now().UTC(),
		CacheHit:       false,
		Partial:        false,
		PartialReasons: []string{},
		EmptyReason:    "",
		Deploy:         snap.Deploy,
		Summary: serializer.DiagnosisSummaryObj{
			TotalCount:    len(snap.Instances),
			AbnormalCount: len(snap.AbnormalInstances()),
		},
		Groups: []*serializer.DiagnosisGroupObj{},
	}
}
