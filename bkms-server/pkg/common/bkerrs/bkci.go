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

// WrapBkCIRepositoryNotBound 包装为「代码仓库未关联到蓝盾」错误
func WrapBkCIRepositoryNotBound(err error, workspaceID, alias string) *Error {
	wrappedErr := Wrapf(
		err, ErrCodeNotFound, "bkci repository not bound, workspaceID: %s, alias: %s", workspaceID, alias,
	)
	return wrappedErr.SetDetails(
		NewDetail(
			ErrDetailCodeBkCIRepositoryNotBound,
			fmt.Sprintf("repository %s is not bound to bkci project", alias),
			WithSystem(SystemName),
			WithModule("bkci"),
		),
	)
}

// WrapBkCIRepositoryPermissionDenied 包装为「蓝盾拒绝关联或访问代码仓库」错误
func WrapBkCIRepositoryPermissionDenied(err error, msg string) *Error {
	return Wrap(err, ErrCodeNoPermission, msg).SetDetails(
		NewDetail(
			ErrDetailCodeBkCIRepositoryPermissionDenied,
			"should authorize git oauth to bkci and have permission to the repository",
			WithSystem(SystemName),
			WithModule("bkci"),
		),
	)
}
