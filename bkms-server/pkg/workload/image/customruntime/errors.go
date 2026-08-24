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

package customruntime

import "github.com/pkg/errors"

var (
	// ErrImageNameNotFound 生效镜像源中不存在该镜像名
	ErrImageNameNotFound = errors.New("custom runtime image name not found in registry")
	// ErrImageTagNotFound 生效镜像源中存在镜像名但不存在该 tag
	ErrImageTagNotFound = errors.New("custom runtime image tag not found in registry")
	// ErrRegistryAccessDenied 访问生效镜像源鉴权失败
	ErrRegistryAccessDenied = errors.New("custom runtime image registry auth required")
	// ErrRegistryAccessFailed 访问生效镜像源超时或网络失败
	ErrRegistryAccessFailed = errors.New("custom runtime image registry access failed")
)
