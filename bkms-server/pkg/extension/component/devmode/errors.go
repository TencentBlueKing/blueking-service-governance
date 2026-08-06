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

package devmode

import "github.com/pkg/errors"

// sentinel errors，外部可通过 errors.Is 判断具体错误类型
var (
	// ErrNotAllowed 当前环境不允许使用开发模式
	ErrNotAllowed = errors.New("dev mode is not allowed in current environment")

	// ErrUnsupportedAppType 不支持的应用类型
	ErrUnsupportedAppType = errors.New("dev mode unsupported app type")

	// ErrAppNameRequired 应用名称未指定
	ErrAppNameRequired = errors.New("dev mode requires app name to be specified")

	// ErrStartupCommandRequired 启动命令未指定
	ErrStartupCommandRequired = errors.New("dev mode requires startup command to be specified")

	// ErrTrpcBinaryPathRequired trpc 二进制路径未指定
	ErrTrpcBinaryPathRequired = errors.New("dev mode requires trpc binary path to be specified")
)
