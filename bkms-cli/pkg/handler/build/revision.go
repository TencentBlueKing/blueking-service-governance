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

// Package build 提供应用构建命令的处理逻辑
package build

import (
	"regexp"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
)

// commitIDPattern CommitID 允许 7–40 位十六进制，与构建服务的校验一致
var commitIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

// ResolveCodeRevision 从三个互斥参数里取出唯一的代码修订。
// 都不给或给了多个时直接失败，不发起构建请求。
func ResolveCodeRevision(branch, tag, commit string) (revisionType, revision string, err error) {
	provided := 0
	if branch != "" {
		provided++
		revisionType, revision = "branch", branch
	}
	if tag != "" {
		provided++
		revisionType, revision = "tag", tag
	}
	if commit != "" {
		provided++
		revisionType, revision = "commit", commit
	}

	if provided != 1 {
		return "", "", clierr.Usagef("exactly one of --branch, --tag, or --commit is required")
	}

	if revisionType == "commit" && !commitIDPattern.MatchString(revision) {
		return "", "", clierr.Usagef("commit id must be 7-40 hexadecimal characters")
	}

	return revisionType, revision, nil
}
