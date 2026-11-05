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
	"log/slog"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// VersionRef 定位一个历史版本：按版本号，或按版本记录 ID。
type VersionRef struct {
	// Version 版本号。
	Version *int64
	// VersionID 版本记录 ID。
	VersionID string
}

// VersionOutput 是历史版本的结构化输出。
type VersionOutput struct {
	Version             int64   `json:"version" yaml:"version"`
	OperationType       string  `json:"operationType" yaml:"operationType"`
	RollbackFromVersion *int64  `json:"rollbackFromVersion,omitempty" yaml:"rollbackFromVersion,omitempty" table:"-"`
	Description         string  `json:"description" yaml:"description"`
	Creator             string  `json:"creator" yaml:"creator"`
	CreatedAt           string  `json:"createdAt" yaml:"createdAt"`
	IsCurrent           bool    `json:"isCurrent" yaml:"isCurrent"`
	Name                string  `json:"name" yaml:"name" table:"-"`
	EnvName             string  `json:"envName" yaml:"envName" table:"-"`
	Type                string  `json:"type" yaml:"type" table:"-"`
	ID                  string  `json:"id" yaml:"id"`
	AppConfigFileID     string  `json:"appConfigFileID" yaml:"appConfigFileID" table:"-"`
	Content             *string `json:"content,omitempty" yaml:"content,omitempty" table:"-"`
	OverlayContent      *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty" table:"-"`
}

// VersionsResult 是历史版本类命令的公共结果。
type VersionsResult struct {
	target *fileTarget
}

// SharesDefault 表示指定了环境，但该环境没有独立实例，版本记录属于默认文件。
func (r *VersionsResult) SharesDefault() bool {
	return r.target.SharesDefault()
}

// ListVersionsResult 是历史版本列表结果。
type ListVersionsResult struct {
	VersionsResult
	versions []client.AppConfigFileVersion
}

// Output 返回用于格式化输出的结构化数据。
func (r *ListVersionsResult) Output() []VersionOutput {
	return lo.Map(r.versions, func(v client.AppConfigFileVersion, _ int) VersionOutput {
		return toVersionOutput(&v, r.target.Detail.CurrentVersion)
	})
}

// ListVersions 列出配置文件在指定环境下实际生效文件的所有历史版本。
func ListVersions(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	envName string,
) (*ListVersionsResult, error) {
	target, err := loadVersionTarget(ctx, cli, appID, ref, envName, false)
	if err != nil {
		return nil, err
	}

	var versions []client.AppConfigFileVersion
	for page := 1; ; page++ {
		resp, err := cli.ListAppConfigFileVersions(ctx, appID, client.ListAppConfigFileVersionsOptions{
			AppConfigFileID: target.Detail.FileID,
			Page:            page,
			PageSize:        client.DefaultListAppConfigFileVersionsPageSize,
		})
		if err != nil {
			return nil, errors.Wrap(err, "list app config file versions")
		}
		slog.Debug(
			"fetched app config file versions page",
			"appID", appID, "fileID", target.Detail.FileID, "page", page,
			"resultsCount", len(resp.Results), "totalCount", resp.Count,
		)
		versions = append(versions, resp.Results...)
		if len(resp.Results) == 0 || len(versions) >= resp.Count {
			break
		}
	}

	return &ListVersionsResult{VersionsResult: VersionsResult{target: target}, versions: versions}, nil
}

// ViewVersionResult 是单个历史版本的查看结果。
type ViewVersionResult struct {
	VersionsResult
	version *client.AppConfigFileVersion
}

// Output 返回用于格式化输出的结构化数据。
func (r *ViewVersionResult) Output() VersionOutput {
	return toVersionOutput(r.version, r.target.Detail.CurrentVersion)
}

// ViewVersion 查看配置文件某个历史版本的内容。
func ViewVersion(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	envName string,
	versionRef VersionRef,
) (*ViewVersionResult, error) {
	target, err := loadVersionTarget(ctx, cli, appID, ref, envName, false)
	if err != nil {
		return nil, err
	}
	version, err := resolveVersion(ctx, cli, appID, target.Detail.FileID, versionRef)
	if err != nil {
		return nil, err
	}
	return &ViewVersionResult{VersionsResult: VersionsResult{target: target}, version: version}, nil
}

// RollbackVersionResult 是回滚结果。
type RollbackVersionResult struct {
	// Target 回滚的目标版本。
	Target *client.AppConfigFileVersion
	// Result 回滚后的文件信息。
	Result *client.RollbackAppConfigFileVersionResult
}

// RollbackVersion 将配置文件回滚到某个历史版本，回滚会生成一个新版本。
func RollbackVersion(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	envName string,
	versionRef VersionRef,
	description string,
) (*RollbackVersionResult, error) {
	target, err := loadVersionTarget(ctx, cli, appID, ref, envName, true)
	if err != nil {
		return nil, err
	}
	version, err := resolveVersion(ctx, cli, appID, target.Detail.FileID, versionRef)
	if err != nil {
		return nil, err
	}
	result, err := cli.RollbackAppConfigFileVersion(ctx, appID, version.ID, client.RollbackAppConfigFileVersionOptions{
		CurrentVersion: lo.ToPtr(target.Detail.CurrentVersion),
		Description:    description,
	})
	if err != nil {
		return nil, errors.Wrap(err, "rollback app config file version")
	}
	return &RollbackVersionResult{Target: version, Result: result}, nil
}

// DeleteVersion 删除配置文件的某个历史版本，当前生效版本不可删除。
func DeleteVersion(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	envName string,
	versionRef VersionRef,
) (*client.AppConfigFileVersion, error) {
	target, err := loadVersionTarget(ctx, cli, appID, ref, envName, true)
	if err != nil {
		return nil, err
	}
	version, err := resolveVersion(ctx, cli, appID, target.Detail.FileID, versionRef)
	if err != nil {
		return nil, err
	}
	if version.Version == target.Detail.CurrentVersion {
		return nil, errors.Errorf("version %d is the current version and cannot be deleted", version.Version)
	}
	if err = cli.DeleteAppConfigFileVersion(ctx, appID, version.ID); err != nil {
		return nil, errors.Wrap(err, "delete app config file version")
	}
	return version, nil
}

func toVersionOutput(v *client.AppConfigFileVersion, currentVersion int64) VersionOutput {
	return VersionOutput{
		Version:             v.Version,
		OperationType:       v.OperationType,
		RollbackFromVersion: v.RollbackFromVersion,
		Description:         v.Description,
		Creator:             v.Creator,
		CreatedAt:           v.CreatedAt,
		IsCurrent:           v.Version == currentVersion,
		Name:                v.Name,
		EnvName:             formatEnvName(v.EnvName),
		Type:                v.Type,
		ID:                  v.ID,
		AppConfigFileID:     v.AppConfigFileID,
		Content:             v.Content,
		OverlayContent:      v.OverlayContent,
	}
}

// loadVersionTarget 定位版本类命令操作的文件；写操作要求指定环境必须有独立实例，避免误改默认文件。
func loadVersionTarget(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	envName string,
	write bool,
) (*fileTarget, error) {
	target, err := loadTarget(ctx, cli, appID, ref, envName)
	if err != nil {
		return nil, err
	}
	if write && target.SharesDefault() {
		return nil, errors.Errorf(
			"env %s of config file %s has no independent config, its versions belong to the default file; "+
				"omit --env to operate on the default file",
			envName, target.Def.Name,
		)
	}
	return target, nil
}

// resolveVersion 按 VersionRef 获取目标文件的某个历史版本，并校验版本属于该文件。
func resolveVersion(
	ctx context.Context,
	cli client.Client,
	appID, fileID string,
	ref VersionRef,
) (*client.AppConfigFileVersion, error) {
	if (ref.VersionID == "") == (ref.Version == nil) {
		return nil, errors.New("exactly one of version or version ID must be specified")
	}

	if ref.VersionID != "" {
		version, err := cli.GetAppConfigFileVersion(ctx, appID, ref.VersionID)
		if err != nil {
			return nil, errors.Wrap(err, "get app config file version")
		}
		if version.AppConfigFileID != fileID {
			return nil, errors.Errorf("version %s does not belong to the selected config file", ref.VersionID)
		}
		return version, nil
	}

	resp, err := cli.ListAppConfigFileVersions(ctx, appID, client.ListAppConfigFileVersionsOptions{
		AppConfigFileID: fileID,
		Version:         ref.Version,
	})
	if err != nil {
		return nil, errors.Wrap(err, "list app config file versions")
	}
	version, found := lo.Find(resp.Results, func(item client.AppConfigFileVersion) bool {
		return item.Version == *ref.Version
	})
	if !found {
		return nil, errors.Errorf("version %d not found", *ref.Version)
	}
	return &version, nil
}
