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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AppConfigFileService 向后兼容的门面，内嵌 AppCfgFileDefService。
// 仅覆盖签名不同的方法（如 Create），其余方法自动提升。
//
// TODO: 所有调用方迁移到 AppCfgFileDefService 后移除此门面。
type AppConfigFileService struct {
	*AppCfgFileDefService
}

// 编译期校验：AppCfgFileDefService 必须实现 MountableFileProvider。
var _ MountableFileProvider = (*AppCfgFileDefService)(nil)

// NewAppConfigFileService 内部构造分层服务。
func NewAppConfigFileService(db *mongo.Database) *AppConfigFileService {
	base := NewBaseAppCfgFileService(db)
	inner := NewAppCfgFileDefService(base)
	return &AppConfigFileService{AppCfgFileDefService: inner}
}

// Create 创建配置文件（含 def）和初始版本，返回 AppConfigFile 以保持兼容。
// 覆盖内嵌的 AppCfgFileDefService.Create（返回 *AppConfigFileWithDef）。
func (s *AppConfigFileService) Create(
	ctx context.Context,
	params CreateCfgFileParams,
) (*AppConfigFile, error) {
	wm, err := s.AppCfgFileDefService.Create(ctx, params)
	if err != nil {
		return nil, err
	}
	return &wm.AppConfigFile, nil
}

// Rollback 回滚到指定历史版本。
// 这里回滚的是文件记录的完整存储形态（包括 type 以及对应的内容字段），
// 因此当前文件类型与目标版本类型不一致时直接拒绝，避免跨类型回滚把当前策略下的实例形态改回去。
func (s *AppConfigFileService) Rollback(
	ctx context.Context,
	appID string,
	versionID bson.ObjectID,
	operator string,
	description string,
	expectedCurrentVersion *int64,
) (*AppConfigFile, *AppConfigFileVersion, error) {
	targetVersion, err := s.GetVersionByAppAndID(ctx, appID, versionID)
	if err != nil {
		return nil, nil, err
	}
	currentFile, err := s.FileStore.GetByID(ctx, targetVersion.AppConfigFileID)
	if err != nil {
		return nil, nil, err
	}
	if currentFile.Type != targetVersion.Type {
		return nil, nil, errors.Wrapf(
			ErrInvalidConfigSpec,
			"current file type %s does not match target version type %s",
			currentFile.Type,
			targetVersion.Type,
		)
	}
	return s.BaseAppCfgFileService.Rollback(
		ctx, appID, versionID, operator, description, expectedCurrentVersion,
	)
}
