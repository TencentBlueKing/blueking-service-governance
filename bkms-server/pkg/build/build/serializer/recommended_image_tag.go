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

package serializer

// GetRecommendedImageTagQueryInput is the query input for getting recommended image tag.
type GetRecommendedImageTagQueryInput struct {
	// 代码来源类型（branch / tag / commit），缺省为 branch，仅 custom 类型时使用
	RevisionType string `form:"revisionType" binding:"omitempty,oneof=branch tag commit"`
	// 代码修订：分支名、Tag 名或 CommitID，仅 custom 类型时使用；CommitID 只取前 7 位拼进版本号
	Revision string `form:"revision"`
	// 兼容字段：按分支检出，仅在 revision 为空且 revisionType 为空或 branch 时生效
	Branch string `form:"branch"`
}

// GetRecommendedImageTagOutput is the JSON response for getting recommended image tag.
type GetRecommendedImageTagOutput struct {
	// 推荐的镜像 Tag
	Data string `json:"data"`
}
