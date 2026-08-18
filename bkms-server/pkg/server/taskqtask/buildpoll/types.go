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

package buildpoll

import "fmt"

// AutoDeployArgs 构建成功后自动部署参数
type AutoDeployArgs struct {
	EnvName         string `json:"envName"`
	TrafficLaneName string `json:"trafficLaneName"`
	Replicas        int32  `json:"replicas"`
}

// Args 镜像构建状态轮询的业务参数，不含用户身份
type Args struct {
	WorkspaceID        string          `json:"workspaceID"`
	PipelineType       string          `json:"pipelineType"`
	AppID              string          `json:"appID"`
	BuildID            string          `json:"buildID"`
	AutoDeploy         *AutoDeployArgs `json:"autoDeploy,omitempty"`
	FailureRetryRemain int             `json:"failureRetryRemain,omitempty"`
}

// String 输出轮询身份、自动部署目标与剩余失败次数，便于日志对齐同一构建的连续 tick
func (args Args) String() string {
	autoDeploy := "off"
	if args.AutoDeploy != nil {
		autoDeploy = fmt.Sprintf("%s/%s", args.AutoDeploy.EnvName, args.AutoDeploy.TrafficLaneName)
	}
	return fmt.Sprintf(
		"<workspace: %s, pipelineType: %s, appID: %s, buildID: %s, autoDeploy: %s, remain: %d>",
		args.WorkspaceID, args.PipelineType, args.AppID, args.BuildID, autoDeploy, args.FailureRetryRemain,
	)
}
