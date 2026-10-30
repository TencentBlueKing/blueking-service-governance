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

package cfgfile

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// DiffVersionsResult 是两个历史版本的对比结果。
type DiffVersionsResult struct {
	VersionsResult
	comparison *client.AppConfigFileVersionComparison
}

// DiffVersionsOutput 是版本对比的结构化输出。
type DiffVersionsOutput struct {
	Previous VersionOutput `json:"previous" yaml:"previous"`
	Current  VersionOutput `json:"current" yaml:"current"`
}

// Output 返回用于格式化输出的结构化数据。
func (r *DiffVersionsResult) Output() DiffVersionsOutput {
	currentVersion := r.target.Detail.CurrentVersion
	return DiffVersionsOutput{
		Previous: toVersionOutput(r.comparison.Previous, currentVersion),
		Current:  toVersionOutput(r.comparison.Current, currentVersion),
	}
}

// UnifiedDiff 按内容字段分别生成 unified diff，两个版本内容一致时返回空字符串。
func (r *DiffVersionsResult) UnifiedDiff() (string, error) {
	prev, cur := r.comparison.Previous, r.comparison.Current
	fields := []struct {
		name      string
		prev, cur *string
	}{
		{"content", prev.Content, cur.Content},
		{"overlayContent", prev.OverlayContent, cur.OverlayContent},
	}

	var sb strings.Builder
	for _, field := range fields {
		if field.prev == nil && field.cur == nil {
			continue
		}
		diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(lo.FromPtr(field.prev)),
			B:        difflib.SplitLines(lo.FromPtr(field.cur)),
			FromFile: fmt.Sprintf("v%d/%s", prev.Version, field.name),
			ToFile:   fmt.Sprintf("v%d/%s", cur.Version, field.name),
			Context:  3,
		})
		if err != nil {
			return "", errors.Wrapf(err, "diff %s", field.name)
		}
		sb.WriteString(diff)
	}
	return sb.String(), nil
}

// DiffVersions 对比配置文件的两个历史版本，to 为 nil 时与当前版本对比。
func DiffVersions(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	envName string,
	from int64,
	to *int64,
) (*DiffVersionsResult, error) {
	target, err := loadVersionTarget(ctx, cli, appID, ref, envName, false)
	if err != nil {
		return nil, err
	}
	fileID := target.Detail.FileID
	if to == nil {
		to = lo.ToPtr(target.Detail.CurrentVersion)
	}

	previous, err := resolveVersion(ctx, cli, appID, fileID, VersionRef{Version: &from})
	if err != nil {
		return nil, err
	}
	current, err := resolveVersion(ctx, cli, appID, fileID, VersionRef{Version: to})
	if err != nil {
		return nil, err
	}

	comparison, err := cli.CompareAppConfigFileVersions(ctx, appID, previous.ID, current.ID)
	if err != nil {
		return nil, errors.Wrap(err, "compare app config file versions")
	}
	return &DiffVersionsResult{VersionsResult: VersionsResult{target: target}, comparison: comparison}, nil
}
