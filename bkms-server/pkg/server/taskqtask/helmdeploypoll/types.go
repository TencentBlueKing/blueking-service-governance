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

package helmdeploypoll

import (
	"fmt"

	"github.com/samber/lo"
)

// Args Helm 部署 / 回滚状态轮询的业务参数，不含用户身份
type Args struct {
	WorkspaceID        string `json:"workspaceID"`
	AppID              string `json:"appID"`
	EnvName            string `json:"envName"`
	TrafficLaneName    string `json:"trafficLaneName"`
	DeployID           string `json:"deployID"`
	FailureRetryRemain int    `json:"failureRetryRemain,omitempty"`
	// TopologyRefreshed 标记本轮部署已触发过拓扑资源范围刷新，由 enqueueNext 置真后随下一 tick 透传
	// 刷新是重操作（集群资源全量扫描 + 快照乐观锁更新），整轮部署只在首个 tick 触发一次
	TopologyRefreshed bool `json:"topologyRefreshed,omitempty"`
}

// String 输出轮询身份与剩余失败次数，便于日志对齐同一部署的连续 tick
func (args Args) String() string {
	trafficLaneName := lo.Ternary(args.TrafficLaneName == "", "default", args.TrafficLaneName)
	return fmt.Sprintf(
		"<workspace: %s, appID: %s, envName: %s, trafficLaneName: %s, id: %s, remain: %d>",
		args.WorkspaceID, args.AppID, args.EnvName, trafficLaneName, args.DeployID, args.FailureRetryRemain,
	)
}
