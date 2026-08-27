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

package bkerrs

import "fmt"

// WrapEnvClusterNamespaceOccupied 包装为"集群命名空间已被其他环境占用"错误。
func WrapEnvClusterNamespaceOccupied(clusterID, namespace, occupiedByEnvName, occupiedByWorkspaceID string) error {
	msg := fmt.Sprintf(
		"cluster %s namespace %s is already bound to environment %q in workspace %s",
		clusterID, namespace, occupiedByEnvName, occupiedByWorkspaceID,
	)
	return New(ErrCodeAlreadyExists, msg).SetDetails(
		NewDetail(
			ErrDetailCodeEnvClusterNamespaceOccupied,
			msg,
			WithSystem(SystemName),
			WithModule("env"),
			WithExtras(map[string]string{
				"clusterID":             clusterID,
				"namespace":             namespace,
				"occupiedByEnvName":     occupiedByEnvName,
				"occupiedByWorkspaceID": occupiedByWorkspaceID,
			}),
		),
	)
}
