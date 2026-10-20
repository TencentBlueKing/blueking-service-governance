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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// The name of the MongoDB collection for storing app config files.
const collectionName = "app_config_files"

// ListOrderByField defines the order choices when listing app config files.
type ListOrderByField string

const (
	// ListOrderByName 保留给上层兼容使用；
	// 由于 name 已迁移到 def 表，需在补齐 def 后于上层按原 name 排序语义处理。
	ListOrderByName ListOrderByField = "name"
)

var (
	// ErrAppConfigFileVersionConflict indicates the file was modified concurrently.
	ErrAppConfigFileVersionConflict = errors.New("app config file version conflict")
	// ErrAppConfigFileNotFound indicates a specific app config file record could not be found.
	ErrAppConfigFileNotFound = errors.New("app config file not found")
	// ErrAppConfigFileAlreadyExists indicates the app config file conflicts with an existing one.
	ErrAppConfigFileAlreadyExists = errors.New("app config file already exists")
)

// AppConfigFileStore 配置文件存储
type AppConfigFileStore = store.GenericStore[AppConfigFile]

// AppConfigFileMeta 配置文件的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一（partial，defID 为 ObjectID）：defID + envName
// - 查询提速：defID、baseAppConfigFileID
var AppConfigFileMeta = store.Meta[AppConfigFile]{
	Collection: collectionName,
	NotFound:   ErrAppConfigFileNotFound,
	Duplicate:  ErrAppConfigFileAlreadyExists,
}

// FilesByApp 按应用 ID 匹配文件记录
func FilesByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// FilesByDef 匹配指定 def 下的文件记录
func FilesByDef(defID bson.ObjectID) store.Opt {
	return store.Where("defID", defID)
}

// FileByDefEnv 匹配指定 def 在某个环境下的文件记录
func FileByDefEnv(defID bson.ObjectID, envName string) store.Opt {
	return store.Combine(FilesByDef(defID), store.Where("envName", envName))
}

// AcfListOption defines an option for listing app config files, options can change the behavior of ListFiles.
type AcfListOption interface {
	ApplyToOptions(*ListOptions)
}

// ListOptions holds the options for listing app config files.
type ListOptions struct {
	// The field to order the results by. Store 层当前仅支持空值（按 createdAt）。
	orderBy ListOrderByField
	// The type of app config file to filter, default to no filtering
	filterType AppConfigFileType
	// The environment name to filter, default to no filtering
	// Use EnvNameFilterAppLevel to filter only app-level config files
	filterEnvName *string
	// filterDefIDs 按 defID 集合过滤，为空则不过滤。
	filterDefIDs []bson.ObjectID
}

// AcfFilterType usage: ListFiles(..., AcfFilterType("normal"))
type AcfFilterType string

// ApplyToOptions applies the option to the given options.
func (ft AcfFilterType) ApplyToOptions(opts *ListOptions) {
	opts.filterType = AppConfigFileType(ft)
}

// AcfFilterEnvName usage: ListFiles(..., AcfFilterEnvName("prod")) or ListFiles(...,
// AcfFilterEnvName(EnvNameFilterAppLevel))
type AcfFilterEnvName string

// ApplyToOptions applies the option to the given options.
func (e AcfFilterEnvName) ApplyToOptions(opts *ListOptions) {
	s := string(e)
	opts.filterEnvName = &s
}

// AcfFilterDefIDs 按 defID 集合过滤文件记录。
type AcfFilterDefIDs []bson.ObjectID

// ApplyToOptions applies the option to the given options.
func (ids AcfFilterDefIDs) ApplyToOptions(opts *ListOptions) {
	opts.filterDefIDs = ids
}

// AcfOrderBy usage: ListFiles(..., AcfOrderBy(ListOrderByName))
type AcfOrderBy string

// ApplyToOptions applies the option to the given options.
func (o AcfOrderBy) ApplyToOptions(opts *ListOptions) {
	opts.orderBy = ListOrderByField(o)
}

// ListFiles lists the app config files of the given app, ordered by createdAt.
func ListFiles(
	ctx context.Context,
	files *AppConfigFileStore,
	appID string,
	listOpts ...AcfListOption,
) ([]AppConfigFile, error) {
	listOptsObj := &ListOptions{}
	for _, opt := range listOpts {
		opt.ApplyToOptions(listOptsObj)
	}

	// name 已迁移到 def 表。为避免把“按 name 排序”错误降级成其他排序，
	// store 层仅保留默认 createdAt 排序；兼容原 name 排序语义由上层补齐 def 后完成。
	if listOptsObj.orderBy != "" {
		return nil, errors.New("unsupported orderBy field")
	}

	opts := []store.Opt{FilesByApp(appID), store.Asc("createdAt")}
	if listOptsObj.filterType != "" {
		opts = append(opts, store.Where("type", listOptsObj.filterType))
	}
	if listOptsObj.filterEnvName != nil {
		opts = append(opts, store.Where("envName", *listOptsObj.filterEnvName))
	}
	if len(listOptsObj.filterDefIDs) > 0 {
		opts = append(opts, store.Where("defID", bson.M{"$in": listOptsObj.filterDefIDs}))
	}
	list, err := files.List(ctx, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "list app config files of app %s", appID)
	}
	return list, nil
}

// UpdateFileIfVersionMatches 仅在库中的 currentVersion 等于 expectedVersion 时用 acf 覆盖文件记录，
// 版本不匹配（或文件已被删除）时返回 ErrAppConfigFileVersionConflict。
func UpdateFileIfVersionMatches(
	ctx context.Context,
	files *AppConfigFileStore,
	acf *AppConfigFile,
	expectedVersion int64,
) error {
	if acf.ID.IsZero() {
		return errors.New("object's ID is required for update")
	}
	err := files.Update(ctx, acf,
		store.UpdateSelected("*"), store.ByID(acf.ID), store.Where("currentVersion", expectedVersion),
	)
	if errors.Is(err, store.ErrNotFound) {
		return errors.Wrapf(ErrAppConfigFileVersionConflict,
			"app config file %s expected version %d", acf.ID.Hex(), expectedVersion)
	}
	if err != nil {
		return errors.Wrapf(err, "update app config file %s", acf.ID.Hex())
	}
	return nil
}

// UpdateEnvInstanceForStrategySwitch 仅更新策略切换涉及的字段，不产生版本记录，也不修改 currentVersion。
func UpdateEnvInstanceForStrategySwitch(
	ctx context.Context,
	files *AppConfigFileStore,
	fileID bson.ObjectID,
	fileType AppConfigFileType,
	content *string,
	overlayContent *string,
	baseAppConfigFileID *bson.ObjectID,
	updater string,
) error {
	if fileID.IsZero() {
		return errors.New("object's ID is required for update")
	}
	patch := &AppConfigFile{
		ID:   fileID,
		Type: fileType,
		VersionedContent: VersionedContent{
			Content:             content,
			OverlayContent:      overlayContent,
			BaseAppConfigFileID: baseAppConfigFileID,
		},
		Updater: updater,
	}
	err := files.Update(
		ctx,
		patch,
		store.UpdateSelected("type", "content", "overlayContent", "baseAppConfigFileID", "updater"),
		store.ByID(fileID),
	)
	if err != nil {
		return errors.Wrapf(err, "update app config file %s for strategy switch", fileID.Hex())
	}
	return nil
}

// DeleteFile 删除应用下的一个文件记录，文件被其它 overlay 文件引用时返回 ErrAppConfigFileReferenced；
// 文件不存在或不属于该应用时不做任何操作。
func DeleteFile(ctx context.Context, files *AppConfigFileStore, appID string, id bson.ObjectID) error {
	isRefed, err := files.Exists(ctx, store.Where("baseAppConfigFileID", id))
	if err != nil {
		return errors.Wrapf(err, "check if app config file %s is referenced", id.Hex())
	}
	if isRefed {
		return ErrAppConfigFileReferenced
	}
	if err = files.DeleteIfExists(ctx, store.ByID(id), FilesByApp(appID)); err != nil {
		return errors.Wrapf(err, "delete app config file %s", id.Hex())
	}
	return nil
}
