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

package bkci

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/gitremote"
)

// repoAliasMaxLen 蓝盾代码库别名的最大长度
const repoAliasMaxLen = 255

// repoAliasPrefixes Git 服务域名到代码库别名前缀的映射；未列出的域名以域名本身作为前缀
var repoAliasPrefixes = map[string]string{
	"git.tencent.com": "tgit",
	"github.com":      "github",
	"gitee.com":       "gitee",
}

var (
	// ErrRepoAliasRequired 未传代码库别名
	ErrRepoAliasRequired = errors.New("repository alias is required")
	// ErrRepoAliasInvalid 无法由仓库地址推导出合法的代码库别名
	ErrRepoAliasInvalid = errors.New("cannot derive repository alias from url")
)

// ResolveRepoAlias 确定代码库别名：关闭工蜂 OAuth 时由仓库地址推导（忽略传入的别名），否则使用传入的别名
func ResolveRepoAlias(repoURL, alias string) (string, error) {
	if config.G != nil && config.G.FeatureIntegrations.DisableBkCIGitOAuth {
		return DeriveRepoAlias(repoURL)
	}
	if alias == "" {
		return "", ErrRepoAliasRequired
	}
	return alias, nil
}

// DeriveRepoAlias 由仓库地址推导代码库别名：前缀 + ":" + 仓库路径，统一小写（蓝盾别名忽略大小写），
// 如 https://git.tencent.com/group/app.git -> tgit:group/app
func DeriveRepoAlias(repoURL string) (string, error) {
	if err := gitremote.ValidateRepoURL(repoURL); err != nil {
		return "", errors.Wrapf(ErrRepoAliasInvalid, "%v", err)
	}
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", errors.Wrapf(ErrRepoAliasInvalid, "parse %q: %v", repoURL, err)
	}

	path := strings.Trim(u.Path, "/")
	if strings.HasSuffix(strings.ToLower(path), ".git") {
		path = path[:len(path)-len(".git")]
	}
	if path = strings.Trim(path, "/"); path == "" {
		return "", errors.Wrapf(ErrRepoAliasInvalid, "repository path of %q is empty", repoURL)
	}

	host := strings.ToLower(u.Hostname())
	prefix, ok := repoAliasPrefixes[host]
	if !ok {
		prefix = host
	}
	alias := strings.ToLower(prefix + ":" + path)
	if len(alias) > repoAliasMaxLen {
		return "", errors.Wrapf(ErrRepoAliasInvalid, "alias of %q exceeds %d characters", repoURL, repoAliasMaxLen)
	}
	return alias, nil
}
