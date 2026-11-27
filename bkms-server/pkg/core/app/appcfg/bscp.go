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

package appcfg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
)

var (
	// ErrBSCPConfigRequired 内容来源为 BSCP 时未提供 BSCP 配置引用
	ErrBSCPConfigRequired = errors.New("bscp config is required when content source type is bscp")
	// ErrBSCPNoFullyReleasedVersion BSCP 服务下没有全量发布的版本
	ErrBSCPNoFullyReleasedVersion = errors.New("no fully released bscp version")
	// ErrBSCPContentInvalidYAML BSCP 配置内容不是合法的 YAML
	ErrBSCPContentInvalidYAML = errors.New("bscp config content is not valid YAML")
)

// ResolveBSCPConfig 校验内容来源为 BSCP 的配置引用，返回记录了最新全量发布版本、可持久化的 BSCPConfig。
// 非 BSCP 来源返回 nil；ref 只需填写 BizID、ServiceID、ConfigID。
func ResolveBSCPConfig(
	ctx context.Context, sourceType ContentSourceType, ref *BSCPConfig, format FileFormat,
) (*BSCPConfig, error) {
	if sourceType != ContentSourceTypeBSCP {
		return nil, nil
	}
	if ref == nil {
		return nil, errors.WithStack(ErrBSCPConfigRequired)
	}

	versionID, content, err := ref.fetchLatest(ctx)
	if err != nil {
		return nil, err
	}
	if content != "" && format == FileFormatYAML {
		var out any
		if err = yaml.Unmarshal([]byte(content), &out); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrBSCPContentInvalidYAML, err)
		}
	}

	return &BSCPConfig{
		BizID:     ref.BizID,
		ServiceID: ref.ServiceID,
		VersionID: versionID,
		ConfigID:  ref.ConfigID,
	}, nil
}

// FetchContent 获取 BSCP 配置对应的配置内容。
func (c *BSCPConfig) FetchContent(ctx context.Context) (string, error) {
	if c == nil {
		return "", errors.New("bscp config cannot be nil")
	}
	_, content, err := c.fetchLatest(ctx)
	return content, err
}

// fetchLatest 拉取最新全量发布版本中该配置项的内容，返回版本 ID 与内容
func (c *BSCPConfig) fetchLatest(ctx context.Context) (versionID, content string, err error) {
	client, err := bscp.New(auth.MustGetUser(ctx))
	if err != nil {
		return "", "", errors.Wrap(err, "initial bscp client")
	}

	versions, err := client.ListServiceVersions(ctx, c.BizID, c.ServiceID)
	if err != nil {
		return "", "", errors.Wrap(err, "list bscp service versions")
	}
	ver := versions.LatestFullyReleased()
	if ver == nil {
		return "", "", errors.WithStack(ErrBSCPNoFullyReleasedVersion)
	}

	cfg, err := client.GetServiceConfig(ctx, c.BizID, c.ServiceID, ver.ID, c.ConfigID)
	if err != nil {
		return "", "", errors.Wrap(err, "get bscp service config")
	}
	content, err = cfg.Content(ctx)
	if err != nil {
		return "", "", errors.Wrap(err, "get bscp config content")
	}
	return ver.ID, content, nil
}
