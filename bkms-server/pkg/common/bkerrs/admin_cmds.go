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

// WrapTrpcAdminPrecheckFailed 包装为 trpc admin 配置预检查失败错误
func WrapTrpcAdminPrecheckFailed(err error, appID, envName string) error {
	wrappedErr := Wrapf(
		err,
		ErrCodeNotFound,
		"trpc admin configuration is incorrect, appID: %s, env: %s",
		appID,
		envName,
	)

	return wrappedErr.SetDetails(
		NewDetail(
			ErrDetailCodeTrpcAdminPrecheckFailed,
			fmt.Sprintf("should ensure trpc admin ip is configured as 0.0.0.0 or 127.0.0.1"+
				" and port is valid for appID: %s, env: %s",
				appID,
				envName,
			),
			WithSystem("bkms"),
			WithModule("trpc.admin"),
		),
	)
}
