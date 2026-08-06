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

// WrapBuildLogUnavailable 包装为构建日志已过期或已清理错误
func WrapBuildLogUnavailable(err error, appID, buildID string) *Error {
	wrappedErr := Wrapf(err, ErrCodeNotFound, "build log unavailable, appID: %s, buildID: %s", appID, buildID)
	return wrappedErr.SetDetails(
		NewDetail(
			ErrDetailCodeBuildLogUnavailable,
			fmt.Sprintf("build log has expired or been cleaned, appID: %s, buildID: %s", appID, buildID),
			WithSystem(SystemName),
			WithModule("build-log"),
		),
	)
}
