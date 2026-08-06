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

// WrapAppConfigFileVersionConflict 包装为应用配置文件版本冲突错误
func WrapAppConfigFileVersionConflict(err error, appID, configFileID string) error {
	wrappedErr := Wrapf(
		err,
		ErrCodeAborted,
		"app config file version conflict, appID: %s, configFileID: %s",
		appID,
		configFileID,
	)
	return wrappedErr.SetDetails(
		NewDetail(
			ErrDetailCodeAppConfigFileVersionConflict,
			fmt.Sprintf(
				"the config file has been modified by another user, please refresh and try again (appID: %s, configFileID: %s)",
				appID,
				configFileID,
			),
			WithSystem("bkms"),
			WithModule("app-config-file"),
		),
	)
}
