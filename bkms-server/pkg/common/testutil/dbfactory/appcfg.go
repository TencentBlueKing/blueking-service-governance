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

package dbfactory

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

// AppConfigFileOpts defines options for creating a test AppConfigFile.
type AppConfigFileOpts struct {
	AppID   string `validate:"required"`
	EnvName string
	Name    string
	Type    appcfg.AppConfigFileType
	Format  appcfg.FileFormat
	// ConfigKind 用于区分 framework 主配置和 plain 附加挂载文件。
	ConfigKind appcfg.ConfigKind
	// MountPath 是 plain 配置文件最终挂载到容器内的目标路径。
	MountPath string
	// DefaultAppConfigFileID 用于将环境级 plain 实例关联回默认逻辑文件。
	DefaultAppConfigFileID *bson.ObjectID
	// IsUnifiedConfig 表示当前逻辑文件是否为统一配置模式。
	IsUnifiedConfig bool
	// MountedEnvNames 表示挂载环境范围。
	MountedEnvNames     []string
	Content             *string
	OverlayContent      *string
	BaseAppConfigFileID *bson.ObjectID
}

// AppConfigFile creates and persists an AppConfigFile for tests.
func AppConfigFile(
	ctx context.Context,
	store appcfg.AppConfigFileStore,
	opts *AppConfigFileOpts,
) *appcfg.AppConfigFile {
	validateOpts(opts)

	if opts.Name == "" {
		opts.Name = "values-" + stringx.Random(4)
	}
	if opts.Type == "" {
		opts.Type = appcfg.AppConfigFileTypeNormal
	}
	if opts.Format == "" {
		opts.Format = appcfg.FileFormatYAML
	}
	if opts.ConfigKind == "" {
		opts.ConfigKind = appcfg.ConfigKindFramework
	}

	acf := &appcfg.AppConfigFile{
		AppConfigFileVersionedSpec: appcfg.AppConfigFileVersionedSpec{
			AppID:               opts.AppID,
			EnvName:             opts.EnvName,
			Name:                opts.Name,
			Type:                opts.Type,
			ConfigKind:          opts.ConfigKind,
			BaseAppConfigFileID: opts.BaseAppConfigFileID,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            opts.Format,
				Content:           opts.Content,
				OverlayContent:    opts.OverlayContent,
			},
		},
		PlainFileAttrs: appcfg.PlainFileAttrs{
			MountPath:              opts.MountPath,
			DefaultAppConfigFileID: opts.DefaultAppConfigFileID,
		},
		EnvConfigMode: appcfg.EnvConfigMode{
			IsUnifiedConfig: opts.IsUnifiedConfig,
			MountedEnvNames: opts.MountedEnvNames,
		},
	}
	id, err := store.Add(ctx, *acf)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	acf.ID = id
	return acf
}

// AppConfigFileVersionOpts 定义创建测试用 AppConfigFileVersion 时的可选参数
type AppConfigFileVersionOpts struct {
	AppID             string                                   `validate:"required"`
	AppConfigFileID   bson.ObjectID                            `validate:"required"`
	EnvName           string                                   `validate:"omitempty"`
	Name              string                                   `validate:"omitempty"`
	Version           int64                                    `validate:"required"`
	Description       string                                   `validate:"omitempty"`
	Type              appcfg.AppConfigFileType                 `validate:"omitempty"`
	ContentSourceType appcfg.ContentSourceType                 `validate:"omitempty"`
	Format            appcfg.FileFormat                        `validate:"omitempty"`
	Content           *string                                  `validate:"-"`
	OperationType     appcfg.AppConfigFileVersionOperationType `validate:"omitempty"`
	Creator           string                                   `validate:"omitempty"`
	CreatedAt         time.Time                                `validate:"-"`
}

// AppConfigFileVersion 创建一个未持久化的测试用 AppConfigFileVersion 对象
func AppConfigFileVersion(opts *AppConfigFileVersionOpts) appcfg.AppConfigFileVersion {
	if opts == nil {
		opts = &AppConfigFileVersionOpts{}
	}

	content := "foo: bar"
	if opts.Content != nil {
		content = *opts.Content
	}
	if opts.EnvName == "" {
		opts.EnvName = appcfg.EnvNameDefault
	}
	if opts.Name == "" {
		opts.Name = "values-" + stringx.Random(4)
	}
	if opts.Type == "" {
		opts.Type = appcfg.AppConfigFileTypeNormal
	}
	if opts.ContentSourceType == "" {
		opts.ContentSourceType = appcfg.ContentSourceTypeLocal
	}
	if opts.Format == "" {
		opts.Format = appcfg.FileFormatYAML
	}
	if opts.OperationType == "" {
		opts.OperationType = appcfg.AppConfigFileVersionOperationTypeUpdate
	}
	if opts.CreatedAt.IsZero() {
		opts.CreatedAt = time.Now()
	}

	return appcfg.AppConfigFileVersion{
		AppConfigFileID: opts.AppConfigFileID,
		AppConfigFileVersionedSpec: appcfg.AppConfigFileVersionedSpec{
			AppID:     opts.AppID,
			EnvName:   opts.EnvName,
			Name:      opts.Name,
			Type:      opts.Type,
			Creator:   opts.Creator,
			CreatedAt: opts.CreatedAt,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: opts.ContentSourceType,
				Format:            opts.Format,
				Content:           &content,
			},
		},
		Version:       opts.Version,
		Description:   opts.Description,
		OperationType: opts.OperationType,
	}
}
