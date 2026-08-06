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

package pipelinevar

const (
	// BuildNum 构建号
	BuildNum = "BK_CI_BUILD_NUM"
	// BuildStartTime 构建开始时间（毫秒时间戳）
	BuildStartTime = "BK_CI_BUILD_START_TIME"
	// BuildEndTime 构建结束时间（毫秒时间戳）
	BuildEndTime = "BK_CI_BUILD_END_TIME"

	// GitRepoURL 代码库地址
	GitRepoURL = "BK_CI_GIT_REPO_URL"
	// GitRepoHeadCommitID 代码库 HEAD Commit ID
	GitRepoHeadCommitID = "BK_CI_GIT_REPO_HEAD_COMMIT_ID"
	// GitRepoHeadCommitAuthor 代码库 HEAD Commit 作者
	GitRepoHeadCommitAuthor = "BK_CI_GIT_REPO_HEAD_COMMIT_COMMITTER"
	// GitRepoHeadCommitMessage 代码库 HEAD Commit 消息
	GitRepoHeadCommitMessage = "BK_CI_GIT_REPO_HEAD_COMMIT_COMMENT"
)

// RequiredVariables 必须参数
var RequiredVariables = []string{
	BuildNum,
	BuildStartTime,
	BuildEndTime,
	GitRepoURL,
	GitRepoHeadCommitID,
	GitRepoHeadCommitAuthor,
	GitRepoHeadCommitMessage,
}
