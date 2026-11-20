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
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/gitremote"
)

// repoAliasMaxLen 蓝盾代码库别名的最大长度
const repoAliasMaxLen = 255

var (
	// ErrCodeRepoTypeUnsupported 当前环境不支持该代码库类型
	ErrCodeRepoTypeUnsupported = errors.New("code repository type is not supported")
	// ErrCodeRepoAuthUnsupported 该代码库类型不支持该鉴权方式
	ErrCodeRepoAuthUnsupported = errors.New("code repository auth type is not supported")
	// ErrRepoAliasRequired 未传代码库别名
	ErrRepoAliasRequired = errors.New("repository alias is required")
	// ErrRepoAliasInvalid 无法由仓库地址推导出合法的代码库别名
	ErrRepoAliasInvalid = errors.New("cannot derive repository alias from url")
)

// CodeRepoTypes 返回当前环境支持的代码库类型配置
func CodeRepoTypes() []config.CodeRepoTypeConfig {
	if config.G == nil {
		return config.DefaultCodeRepoTypes
	}
	return config.G.FeatureIntegrations.EffectiveCodeRepoTypes()
}

// codeRepoTypeOf 返回代码库类型的配置
func codeRepoTypeOf(repoType string) (config.CodeRepoTypeConfig, error) {
	for _, t := range CodeRepoTypes() {
		if t.Type == repoType {
			return t, nil
		}
	}
	return config.CodeRepoTypeConfig{}, errors.Wrapf(ErrCodeRepoTypeUnsupported, "type %s", repoType)
}

// ResolveAuthType 确定鉴权方式：未指定时取该代码库类型的默认鉴权方式，指定时校验是否支持
func ResolveAuthType(repoType, authType string) (string, error) {
	typeCfg, err := codeRepoTypeOf(repoType)
	if err != nil {
		return "", err
	}
	if authType == "" {
		return typeCfg.AuthTypes[0], nil
	}
	if !slices.Contains(typeCfg.AuthTypes, authType) {
		return "", errors.Wrapf(ErrCodeRepoAuthUnsupported, "type %s, auth type %s", repoType, authType)
	}
	return authType, nil
}

// ResolveCodeRepo 确定代码库的鉴权方式与别名。
// 工蜂 OAuth 使用传入的别名（工蜂项目全路径），其余由仓库地址推导（忽略传入的别名）
func ResolveCodeRepo(repoType, authType, repoURL, alias string) (resolvedAuth, resolvedAlias string, err error) {
	if resolvedAuth, err = ResolveAuthType(repoType, authType); err != nil {
		return "", "", err
	}
	if repoType == config.CodeRepoTypeTGit && resolvedAuth == config.CodeRepoAuthOAuth {
		if alias == "" {
			return "", "", ErrRepoAliasRequired
		}
		return resolvedAuth, alias, nil
	}
	if resolvedAlias, err = DeriveRepoAlias(repoType, repoURL); err != nil {
		return "", "", err
	}
	return resolvedAuth, resolvedAlias, nil
}

// DeriveRepoAlias 由仓库地址推导代码库别名：小写类型 + ":" + 仓库路径，统一小写（蓝盾别名忽略大小写），
// 如 TGit、https://git.tencent.com/group/app.git -> tgit:group/app
func DeriveRepoAlias(repoType, repoURL string) (string, error) {
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

	alias := strings.ToLower(repoType + ":" + path)
	if len(alias) > repoAliasMaxLen {
		return "", errors.Wrapf(ErrRepoAliasInvalid, "alias of %q exceeds %d characters", repoURL, repoAliasMaxLen)
	}
	return alias, nil
}
