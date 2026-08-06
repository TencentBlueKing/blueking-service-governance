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

package credential

// HelmRepoCredential Helm 仓库凭证
type HelmRepoCredential struct {
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// CredentialID 蓝盾凭证 ID（固定值 bkms_helm_repo_credential）
	CredentialID string `bson:"credentialID"`
	// Username bkrepo 用户名
	Username string `bson:"username"`
	// Password bkrepo 密码（对称加密存储）
	Password string `bson:"password"`
}
