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
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// BaseAppCfgFileService 是三表模型（def / file / version）的底层 CRUD 服务。
//
// 历史：配置文件原先落在单表 app_config_files 上，逻辑身份与环境实例内容混在一起，
// AppConfigFileService 同时承担 CRUD 和 ConfigKind 校验。拆表后，逻辑身份进
// app_config_file_defs，环境实例留在 app_config_files，版本在 app_config_file_versions。
//
// 定位：只做跨表读写（hydrate、创建文件+版本、删除时级联 sibling/def），不感知 ConfigKind。
// ConfigKind 相关规则由上层 AppCfgFileDefService 通过 ConfigKindPolicy 处理。
// AppConfigFileService 仍作为兼容门面内嵌场景层，CLI 切完后会撤掉。
type BaseAppCfgFileService struct {
	DefStore     AppConfigFileDefStore
	FileStore    AppConfigFileStore
	VersionStore AppConfigFileVersionStore
}

// NewBaseAppCfgFileService 创建底层服务实例。
func NewBaseAppCfgFileService(
	defStore AppConfigFileDefStore,
	fileStore AppConfigFileStore,
	versionStore AppConfigFileVersionStore,
) *BaseAppCfgFileService {
	return &BaseAppCfgFileService{
		DefStore:     defStore,
		FileStore:    fileStore,
		VersionStore: versionStore,
	}
}

// --- 查询方法 ---

// GetByID 加载文件记录及其 def，返回组合视图。
func (s *BaseAppCfgFileService) GetByID(ctx context.Context, id bson.ObjectID) (*AppConfigFileWithDef, error) {
	acf, err := s.FileStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, acf)
}

// --- 变更方法 ---

// UpdateFile 持久化文件变更并追加版本记录，name 从 def 传入用于版本快照。
func (s *BaseAppCfgFileService) UpdateFile(
	ctx context.Context,
	acf *AppConfigFile,
	name string,
	operator string,
	opts UpdateCfgFileOptions,
) error {
	curVersion := acf.CurrentVersion

	if opts.ExpectedCurrentVersion != nil && *opts.ExpectedCurrentVersion != curVersion {
		return ErrAppConfigFileVersionConflict
	}

	acf.CurrentVersion = curVersion + 1
	acf.Updater = operator

	version, err := s.buildVersionRecord(
		ctx,
		*acf,
		name,
		opts.OperationType,
		opts.Description,
		opts.RollbackFromVersion,
		operator,
	)
	if err != nil {
		return err
	}

	if _, err = s.FileStore.UpdateIfVersionMatches(ctx, *acf, curVersion); err != nil {
		return err
	}

	_, err = s.VersionStore.Add(ctx, version)
	return err
}

// Rollback 加载目标版本并回滚到当前文件，生成一条回滚版本记录。
func (s *BaseAppCfgFileService) Rollback(
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
	acf, err := s.FileStore.GetByID(ctx, targetVersion.AppConfigFileID)
	if err != nil {
		return nil, nil, err
	}
	rollbackFromVersion := targetVersion.Version

	acf.Type = targetVersion.Type
	acf.VersionedContent = targetVersion.VersionedContent

	if description == "" {
		description = fmt.Sprintf("回滚到 v%d", targetVersion.Version)
	}

	if err = s.UpdateFile(ctx, acf, targetVersion.Name, operator, UpdateCfgFileOptions{
		OperationType:          AppConfigFileVersionOperationTypeRollback,
		Description:            description,
		RollbackFromVersion:    &rollbackFromVersion,
		ExpectedCurrentVersion: expectedCurrentVersion,
	}); err != nil {
		return nil, nil, err
	}
	return acf, targetVersion, nil
}

// DeleteFile 删除配置文件、关联 def 和所有版本记录。
func (s *BaseAppCfgFileService) DeleteFile(
	ctx context.Context,
	appID string,
	fileID bson.ObjectID,
) (*AppConfigFile, error) {
	acf, err := s.FileStore.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if _, err = s.FileStore.DeleteByID(ctx, appID, fileID); err != nil {
		return nil, errors.Wrap(err, "delete app config file")
	}
	if _, err = s.VersionStore.DeleteByFileID(ctx, fileID); err != nil {
		return nil, errors.Wrap(err, "delete version records for deleted file")
	}

	// 若删除的是默认实例，同时清理同 def 下的环境实例和 def 记录，不区分 configKind。
	if acf.EnvName == EnvNameDefault {
		siblings, err := s.FileStore.ListByDefID(ctx, acf.DefID)
		if err != nil {
			return nil, errors.Wrap(err, "listing sibling env instances")
		}
		for _, sib := range siblings {
			if _, err = s.FileStore.DeleteByID(ctx, appID, sib.ID); err != nil {
				return nil, errors.Wrapf(err, "delete sibling env instance %s", sib.ID.Hex())
			}
			if _, err = s.VersionStore.DeleteByFileID(ctx, sib.ID); err != nil {
				return nil, errors.Wrapf(err, "delete versions for sibling %s", sib.ID.Hex())
			}
		}
		if _, err = s.DefStore.DeleteByID(ctx, acf.DefID); err != nil {
			return nil, errors.Wrap(err, "delete def record")
		}
		return acf, nil
	}

	// 若没有关联文件，def 定义一并删除
	remaining, err := s.FileStore.ListByDefID(ctx, acf.DefID)
	if err != nil {
		return nil, errors.Wrap(err, "listing remaining files for def cleanup")
	}
	if len(remaining) == 0 {
		if _, err = s.DefStore.DeleteByID(ctx, acf.DefID); err != nil {
			return nil, errors.Wrap(err, "delete orphan def record")
		}
	}

	return acf, nil
}

// --- 版本方法 ---

// ListVersions 按过滤条件返回版本记录。
func (s *BaseAppCfgFileService) ListVersions(
	ctx context.Context,
	opts AppConfigFileVersionListOptions,
) ([]AppConfigFileVersion, int64, error) {
	return s.VersionStore.List(ctx, opts)
}

// GetVersionByAppAndID 获取一条版本记录，并校验其归属于指定应用。
func (s *BaseAppCfgFileService) GetVersionByAppAndID(
	ctx context.Context,
	appID string,
	id bson.ObjectID,
) (*AppConfigFileVersion, error) {
	items, err := s.VersionStore.BatchGetByAppAndIDs(ctx, appID, []bson.ObjectID{id})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.Wrap(
			ErrAppCfgFileVersionNotFound,
			fmt.Sprintf("app config file version %s not found", id.Hex()),
		)
	}
	return &items[0], nil
}

// CompareVersions 返回两条版本记录用于 diff，同时校验归属关系。
func (s *BaseAppCfgFileService) CompareVersions(
	ctx context.Context,
	appID string,
	previousVersionID bson.ObjectID,
	currentVersionID bson.ObjectID,
) (*AppConfigFileVersion, *AppConfigFileVersion, error) {
	versions, err := s.VersionStore.BatchGetByAppAndIDs(
		ctx, appID, []bson.ObjectID{previousVersionID, currentVersionID},
	)
	if err != nil {
		return nil, nil, err
	}
	versionMap := make(map[bson.ObjectID]AppConfigFileVersion, len(versions))
	for _, v := range versions {
		versionMap[v.ID] = v
	}
	previous, ok := versionMap[previousVersionID]
	if !ok {
		return nil, nil, errors.Wrap(ErrAppCfgFileVersionNotFound, "previous app config file version not found")
	}
	current, ok := versionMap[currentVersionID]
	if !ok {
		return nil, nil, errors.Wrap(ErrAppCfgFileVersionNotFound, "current app config file version not found")
	}
	if previous.AppConfigFileID != current.AppConfigFileID {
		return nil, nil, ErrComparedVersionsBelongToDifferentFiles
	}
	return &previous, &current, nil
}

// DeleteVersion 软删除一条历史版本，禁止删除当前生效版本。
func (s *BaseAppCfgFileService) DeleteVersion(
	ctx context.Context,
	appID string,
	versionID bson.ObjectID,
	deleter string,
) (*AppConfigFileVersion, *AppConfigFile, error) {
	targetVersion, err := s.GetVersionByAppAndID(ctx, appID, versionID)
	if err != nil {
		return nil, nil, err
	}
	acf, err := s.FileStore.GetByID(ctx, targetVersion.AppConfigFileID)
	if err != nil {
		return nil, nil, err
	}
	if acf.CurrentVersion == targetVersion.Version {
		return nil, nil, ErrUsingVersionCannotBeDeleted
	}
	if _, err = s.VersionStore.SoftDeleteByID(ctx, versionID, deleter); err != nil {
		return nil, nil, err
	}
	return targetVersion, acf, nil
}

// CreateFileWithVersion 写入文件记录并创建初始版本。
func (s *BaseAppCfgFileService) CreateFileWithVersion(
	ctx context.Context,
	acf AppConfigFile,
	name string,
	description string,
	creator string,
) (*AppConfigFile, error) {
	fileID, err := s.FileStore.Add(ctx, acf)
	if err != nil {
		return nil, errors.Wrap(err, "adding file record")
	}
	acf.ID = fileID

	version, err := s.buildVersionRecord(
		ctx, acf, name,
		AppConfigFileVersionOperationTypeCreate, description, nil, creator,
	)
	if err != nil {
		return nil, errors.Wrap(err, "building initial version record")
	}
	if _, err = s.VersionStore.Add(ctx, version); err != nil {
		return nil, errors.Wrap(err, "adding initial version record")
	}
	return &acf, nil
}

// --- 内部辅助方法 ---

func (s *BaseAppCfgFileService) buildVersionRecord(
	ctx context.Context,
	acf AppConfigFile,
	name string,
	operationType AppConfigFileVersionOperationType,
	description string,
	rollbackFromVersion *int64,
	creator string,
) (AppConfigFileVersion, error) {
	version := AppConfigFileVersion{
		AppConfigFileID:     acf.ID,
		DefID:               acf.DefID,
		AppID:               acf.AppID,
		EnvName:             acf.EnvName,
		Name:                name,
		Type:                acf.Type,
		VersionedContent:    acf.VersionedContent,
		Version:             acf.CurrentVersion,
		Description:         description,
		OperationType:       operationType,
		RollbackFromVersion: rollbackFromVersion,
		IsDeleted:           false,
		Creator:             creator,
		CreatedAt:           time.Now(),
	}
	if acf.BaseAppConfigFileID != nil {
		baseAcf, err := s.FileStore.GetByID(ctx, *acf.BaseAppConfigFileID)
		if err != nil {
			return AppConfigFileVersion{}, err
		}
		baseVersion := baseAcf.CurrentVersion
		version.BaseVersion = &baseVersion
	}
	return version, nil
}

// GetDefaultFileWithDef 按 defID 加载默认文件记录及其 def，返回组合视图。
func (s *BaseAppCfgFileService) GetDefaultFileWithDef(
	ctx context.Context,
	defID bson.ObjectID,
) (*AppConfigFileWithDef, error) {
	def, err := s.DefStore.GetByID(ctx, defID)
	if err != nil {
		return nil, errors.Wrap(err, "loading def")
	}
	acf, err := s.FileStore.GetByDefIDAndEnv(ctx, defID, EnvNameDefault)
	if err != nil {
		return nil, errors.Wrap(err, "loading default file for def")
	}
	return &AppConfigFileWithDef{AppConfigFile: *acf, Def: def}, nil
}

func (s *BaseAppCfgFileService) hydrate(ctx context.Context, acf *AppConfigFile) (*AppConfigFileWithDef, error) {
	def, err := s.DefStore.GetByID(ctx, acf.DefID)
	if err != nil {
		return nil, errors.Wrap(err, "loading def for hydration")
	}
	return &AppConfigFileWithDef{AppConfigFile: *acf, Def: def}, nil
}
