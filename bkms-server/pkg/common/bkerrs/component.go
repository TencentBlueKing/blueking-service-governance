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

// WrapComponentNotInstalled 包装为"集群未安装所需组件"错误。
// 适用于任意依赖集群侧组件/CRD 的场景（如 GPA 自动扩缩容、APM 等），
// 前端可通过 detail code ErrDetailCodeComponentNotInstalled 识别该大类错误，
// 并通过 detail 的 module 字段区分具体是哪个组件（如 "gpa"）。
func WrapComponentNotInstalled(err error, component, clusterID string) error {
	wrappedErr := Wrapf(err, ErrCodeNotFound,
		"component %s not installed in cluster: %s", component, clusterID)
	return wrappedErr.SetDetails(
		NewDetail(
			ErrDetailCodeComponentNotInstalled,
			fmt.Sprintf("component %s is not installed in cluster %s, "+
				"please install it before using this feature", component, clusterID),
			WithSystem("bkms"),
			WithModule(component),
		),
	)
}
