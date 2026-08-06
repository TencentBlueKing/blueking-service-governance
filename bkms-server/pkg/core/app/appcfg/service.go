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

// AppConfigFileService encapsulates file and version persistence logic.
type AppConfigFileService struct {
	fileStore    AppConfigFileStore
	versionStore AppConfigFileVersionStore
}

// NewAppConfigFileService creates a service instance.
func NewAppConfigFileService(
	fileStore AppConfigFileStore,
	versionStore AppConfigFileVersionStore,
) *AppConfigFileService {
	return &AppConfigFileService{
		fileStore:    fileStore,
		versionStore: versionStore,
	}
}

// Create persists a new config file and creates the initial version.
func (s *AppConfigFileService) Create(
	ctx context.Context,
	params CreateCfgFileParams,
) (*AppConfigFile, error) {
	acf := AppConfigFile{
		AppConfigFileContentSpec: AppConfigFileContentSpec{
			AppID:             params.AppID,
			EnvName:           params.EnvName,
			Name:              params.Name,
			Type:              params.Type,
			ContentSourceType: params.ContentSourceType,
			Format:            params.Format,
			BSCPConfig:        params.BSCPConfig,
			Content:           params.Content,
			OverlayContent:    params.OverlayContent,
			Creator:           params.Creator,
		},
		Updater: params.Creator,
		// initial version is v1
		CurrentVersion: 1,
	}
	if params.BaseAppConfigFileID != nil {
		acf.BaseAppConfigFileID = params.BaseAppConfigFileID
	}
	acf.initializeContentFields(params.Type, params.ContentSourceType)
	if params.Content != nil {
		acf.Content = params.Content
	}
	if params.OverlayContent != nil {
		acf.OverlayContent = params.OverlayContent
	}
	// 确保 Format 有默认值
	if acf.Format == "" {
		acf.Format = FileFormatYAML
	}

	fileID, err := s.createFileWithInitialVersion(ctx, acf, params.Description)
	if err != nil {
		return nil, err
	}
	acf.ID = fileID
	return &acf, nil
}

func (s *AppConfigFileService) createFileWithInitialVersion(
	ctx context.Context,
	acf AppConfigFile,
	description string,
) (bson.ObjectID, error) {
	fileID, err := s.fileStore.Add(ctx, acf)
	if err != nil {
		return bson.NilObjectID, err
	}
	if s.versionStore == nil {
		return bson.NilObjectID, errors.New("versionStore is required but not configured")
	}

	acf.ID = fileID
	version, err := s.buildVersionRecord(
		ctx,
		acf,
		AppConfigFileVersionOperationTypeCreate,
		description,
		nil,
		acf.Creator,
	)
	if err != nil {
		return bson.NilObjectID, err
	}
	if _, err = s.versionStore.Add(ctx, version); err != nil {
		return bson.NilObjectID, err
	}
	return fileID, nil
}

// UpdateFile persists a file change and appends a version record when requested.
func (s *AppConfigFileService) UpdateFile(
	ctx context.Context,
	acf *AppConfigFile,
	operator string,
	opts UpdateCfgFileOptions,
) error {
	curVersion := acf.CurrentVersion

	// 如果前端传入了期望的版本号，校验是否与数据库中当前版本号一致
	if opts.ExpectedCurrentVersion != nil && *opts.ExpectedCurrentVersion != curVersion {
		return ErrAppConfigFileVersionConflict
	}

	acf.CurrentVersion = curVersion + 1
	acf.Updater = operator

	if s.versionStore == nil {
		return errors.New("versionStore is required but not configured")
	}

	version, err := s.buildVersionRecord(
		ctx,
		*acf,
		opts.OperationType,
		opts.Description,
		opts.RollbackFromVersion,
		operator,
	)
	if err != nil {
		return err
	}

	if _, err = s.fileStore.UpdateIfVersionMatches(ctx, *acf, curVersion); err != nil {
		return err
	}

	_, err = s.versionStore.Add(ctx, version)
	if err != nil {
		return err
	}
	return nil
}

// Rollback loads the target version, applies it to the live file, and persists a new rollback version.
// description is the reason for rollback.
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
	acf, err := s.fileStore.GetByID(ctx, targetVersion.AppConfigFileID)
	if err != nil {
		return nil, nil, err
	}
	rollbackFromVersion := targetVersion.Version
	acf.AppConfigFileContentSpec = targetVersion.AppConfigFileContentSpec

	// 如果未传入描述，则使用默认描述
	if description == "" {
		description = fmt.Sprintf("回滚到 v%d", targetVersion.Version)
	}

	if err = s.UpdateFile(
		ctx,
		acf,
		operator,
		UpdateCfgFileOptions{
			OperationType:          AppConfigFileVersionOperationTypeRollback,
			Description:            description,
			RollbackFromVersion:    &rollbackFromVersion,
			ExpectedCurrentVersion: expectedCurrentVersion,
		},
	); err != nil {
		return nil, nil, err
	}
	return acf, targetVersion, nil
}

// ListVersions returns version records by filter.
func (s *AppConfigFileService) ListVersions(
	ctx context.Context,
	opts AppConfigFileVersionListOptions,
) ([]AppConfigFileVersion, int64, error) {
	return s.versionStore.List(ctx, opts)
}

// GetVersionByAppAndID gets one version record and ensures it belongs to the specified app.
func (s *AppConfigFileService) GetVersionByAppAndID(
	ctx context.Context,
	appID string,
	id bson.ObjectID,
) (*AppConfigFileVersion, error) {
	items, err := s.versionStore.BatchGetByAppAndIDs(ctx, appID, []bson.ObjectID{id})
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

// CompareVersions returns two versions for diff after ownership and relation checks.
func (s *AppConfigFileService) CompareVersions(
	ctx context.Context,
	appID string,
	previousVersionID bson.ObjectID,
	currentVersionID bson.ObjectID,
) (*AppConfigFileVersion, *AppConfigFileVersion, error) {
	versions, err := s.versionStore.BatchGetByAppAndIDs(
		ctx,
		appID,
		[]bson.ObjectID{previousVersionID, currentVersionID},
	)
	if err != nil {
		return nil, nil, err
	}
	versionMap := make(map[bson.ObjectID]AppConfigFileVersion, len(versions))
	for _, version := range versions {
		versionMap[version.ID] = version
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

// DeleteFile deletes a config file and hard-deletes all its version records.
func (s *AppConfigFileService) DeleteFile(
	ctx context.Context,
	appID string,
	fileID bson.ObjectID,
) (*AppConfigFile, error) {
	acf, err := s.fileStore.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if _, err = s.fileStore.DeleteByID(ctx, appID, fileID); err != nil {
		return nil, errors.Wrap(err, "delete app config file")
	}

	if s.versionStore != nil {
		if _, err = s.versionStore.DeleteByFileID(ctx, fileID); err != nil {
			return nil, errors.Wrap(err, "delete version records for deleted file")
		}
	}
	return acf, nil
}

// DeleteVersion soft deletes one historical version after checking it is not the live version.
func (s *AppConfigFileService) DeleteVersion(
	ctx context.Context,
	appID string,
	versionID bson.ObjectID,
	deleter string,
) (*AppConfigFileVersion, *AppConfigFile, error) {
	targetVersion, err := s.GetVersionByAppAndID(ctx, appID, versionID)
	if err != nil {
		return nil, nil, err
	}
	acf, err := s.fileStore.GetByID(ctx, targetVersion.AppConfigFileID)
	if err != nil {
		return nil, nil, err
	}
	if acf.CurrentVersion == targetVersion.Version {
		return nil, nil, ErrUsingVersionCannotBeDeleted
	}
	if _, err = s.versionStore.SoftDeleteByID(ctx, versionID, deleter); err != nil {
		return nil, nil, err
	}
	return targetVersion, acf, nil
}

func (s *AppConfigFileService) buildVersionRecord(
	ctx context.Context,
	acf AppConfigFile,
	operationType AppConfigFileVersionOperationType,
	description string,
	rollbackFromVersion *int64,
	creator string,
) (AppConfigFileVersion, error) {
	version := AppConfigFileVersion{
		AppConfigFileID:          acf.ID,
		AppConfigFileContentSpec: acf.AppConfigFileContentSpec,
		Version:                  acf.CurrentVersion,
		Description:              description,
		OperationType:            operationType,
		RollbackFromVersion:      rollbackFromVersion,
		IsDeleted:                false,
	}
	// 版本记录的 creator 是该版本的操作者，非文件的创建者
	version.Creator = creator
	// 版本记录的 createdAt 记录版本生成时间，非文件首次创建时间
	version.CreatedAt = time.Now()
	if acf.BaseAppConfigFileID != nil {
		baseAcf, err := s.fileStore.GetByID(ctx, *acf.BaseAppConfigFileID)
		if err != nil {
			return AppConfigFileVersion{}, err
		}
		baseVersion := baseAcf.CurrentVersion
		version.BaseVersion = &baseVersion
	}
	return version, nil
}
