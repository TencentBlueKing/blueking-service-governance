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

// ErrDetailCode 错误详情码，用于标记某类具体的错误，如删除工作空间失败、创建环境失败等，推荐开发者按需定义
// 重要：在业务逻辑中所有 bkerrs.NewDetail() 调用，都应使用该处枚举值之一，以便后续统一管理，单元测试则不做限制！
type ErrDetailCode string

const (
	// ErrDetailCodeIAMNoPermission 无 IAM 权限
	ErrDetailCodeIAMNoPermission ErrDetailCode = "IAM_NO_PERMISSION"

	// ErrDetailCodeAPMConfigMissing APM 配置缺失
	ErrDetailCodeAPMConfigMissing ErrDetailCode = "APM_CONFIG_MISSING"

	// ErrDetailCodeTrpcAdminPrecheckFailed trpc admin 配置预检查失败
	ErrDetailCodeTrpcAdminPrecheckFailed ErrDetailCode = "TRPC_ADMIN_PRECHECK_FAILED"

	// ErrDetailCodeAppConfigFileVersionConflict 应用配置文件版本冲突
	ErrDetailCodeAppConfigFileVersionConflict ErrDetailCode = "APP_CONFIG_FILE_VERSION_CONFLICT"

	// ErrDetailCodeNotFullyReleased BSCP 服务未全量发布
	ErrDetailCodeNotFullyReleased ErrDetailCode = "BSCP_NOT_FULLY_RELEASED"

	// ErrDetailCodeImageRepositoryAuthRequired 删除镜像时镜像仓库鉴权缺失或失败
	ErrDetailCodeImageRepositoryAuthRequired ErrDetailCode = "IMAGE_REPOSITORY_AUTH_REQUIRED"

	// ErrDetailCodeBSCPNoPermission BSCP 服务无权限
	ErrDetailCodeBSCPNoPermission ErrDetailCode = "BSCP_NO_PERMISSION"

	// ErrDetailCodeComponentNotInstalled 环境所在集群未安装所需的组件（通过 module 区分具体组件）
	ErrDetailCodeComponentNotInstalled ErrDetailCode = "COMPONENT_NOT_INSTALLED"

	// ErrDetailCodeBuildLogUnavailable 构建日志已过期或已清理
	ErrDetailCodeBuildLogUnavailable ErrDetailCode = "BUILD_LOG_UNAVAILABLE"

	// ErrDetailCodeEnvClusterNamespaceOccupied 环境绑定的集群+命名空间已被其他环境占用
	ErrDetailCodeEnvClusterNamespaceOccupied ErrDetailCode = "ENV_CLUSTER_NAMESPACE_OCCUPIED"
)
