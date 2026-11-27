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

package coderepo

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/repoutil/git"
)

// aliasMaxLen 别名最大长度，与蓝盾一致
const aliasMaxLen = 255

// DeriveAlias 由仓库地址生成别名，如 Git、https://git.example.com/group/app.git -> git:group/app
func DeriveAlias(sourceTypeName, repoURL string) (string, error) {
	if err := git.ValidateRepoURL(repoURL); err != nil {
		return "", errors.Wrapf(ErrAliasInvalid, "%v", err)
	}
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", errors.Wrapf(ErrAliasInvalid, "parse %q: %v", repoURL, err)
	}

	path := strings.Trim(u.Path, "/")
	if strings.HasSuffix(strings.ToLower(path), ".git") {
		path = path[:len(path)-len(".git")]
	}
	if path = strings.Trim(path, "/"); path == "" {
		return "", errors.Wrapf(ErrAliasInvalid, "repository path of %q is empty", repoURL)
	}

	alias := strings.ToLower(sourceTypeName + ":" + path)
	if len(alias) > aliasMaxLen {
		return "", errors.Wrapf(ErrAliasInvalid, "alias of %q exceeds %d characters", repoURL, aliasMaxLen)
	}
	return alias, nil
}
