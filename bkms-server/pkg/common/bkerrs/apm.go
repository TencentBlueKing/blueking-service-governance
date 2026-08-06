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

// WrapAPMConfigMissing 包装为 APM 配置缺失错误
func WrapAPMConfigMissing(err error, appID, envName string) error {
	wrappedErr := Wrapf(err, ErrCodeNotFound, "apm config missing, appID: %s, envName: %s", appID, envName)
	return wrappedErr.SetDetails(
		NewDetail(
			ErrDetailCodeAPMConfigMissing,
			fmt.Sprintf("should enable apm config in config file for appID: %s, envName: %s", appID, envName),
			WithSystem("bkms"),
			WithModule("apm"),
		),
	)
}
