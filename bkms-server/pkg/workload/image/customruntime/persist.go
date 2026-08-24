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

package customruntime

import (
	"context"
	stderrors "errors"

	"github.com/pkg/errors"

	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// PersistManager 在构建配置保存成功后 get_or_create 自定义镜像记录，并按需初始化 TAG 快照
type PersistManager struct {
	store       Store
	checker     *ExistenceChecker
	snapshotSvc *snapshot.Service
}

// NewPersistManager 创建 PersistManager，内部同时装配路径判定与仓库存在性校验
func NewPersistManager(store Store, snapshotSvc *snapshot.Service) *PersistManager {
	return &PersistManager{
		store:       store,
		checker:     NewExistenceChecker(snapshotSvc),
		snapshotSvc: snapshotSvc,
	}
}

// MatchesWorkspaceRegistry 不含 tag 的镜像名是否落在工作空间生效镜像源路径下
func (m *PersistManager) MatchesWorkspaceRegistry(
	ctx context.Context, workspaceID, imageName string,
) (bool, error) {
	if m == nil {
		return false, nil
	}
	return m.checker.MatchesWorkspaceRegistry(ctx, workspaceID, imageName)
}

// ValidateTaggedReference 向生效镜像源确认镜像名与 tag 都真实存在
func (m *PersistManager) ValidateTaggedReference(ctx context.Context, workspaceID, image string) error {
	if m == nil || m.checker == nil {
		return errors.New("custom runtime image persist manager is not initialized")
	}
	return m.checker.ValidateTaggedReference(ctx, workspaceID, image)
}

// PersistAfterSave 对判定为自定义的 builder / runner 做 get_or_create，并按需初始化 TAG 快照
//
// 失败只返回 error 供调用方打日志，调用方不得回滚已成功的构建配置保存
func (m *PersistManager) PersistAfterSave(ctx context.Context, workspaceID string, cfg *imagebuild.Config) error {
	if m == nil || m.store == nil || m.checker == nil {
		return errors.New("custom runtime image persist manager is not initialized")
	}
	if cfg == nil || cfg.SourceType != imagebuild.SourceTypeCodeRepository || cfg.CodeRepo == nil {
		return nil
	}
	if cfg.CodeRepo.EffectiveImageBuildMode() != imagebuild.ImageBuildModePlatform {
		return nil
	}
	platformCfg := cfg.CodeRepo.PlatformBuildConfig
	if platformCfg == nil {
		return nil
	}

	// errors.Wrap 对 nil 返回 nil，两条互不影响，各自的失败都要能从日志里区分出来
	builderErr := errors.Wrap(
		m.persistOne(ctx, workspaceID, ImageTypeBuilder, platformCfg.BuilderImage), "persist builder image",
	)
	runnerErr := errors.Wrap(
		m.persistOne(ctx, workspaceID, ImageTypeRunner, platformCfg.RunnerImage), "persist runner image",
	)
	return stderrors.Join(builderErr, runnerErr)
}

// persistOne 对单条 builder / runner 做路径判定后 get_or_create，并按需初始化 TAG 快照
func (m *PersistManager) persistOne(ctx context.Context, workspaceID string, imageType ImageType, image string) error {
	// 保存前已校验过引用；解析失败说明不是可落库的 name:tag，跳过
	ref, err := workloadruntime.ParseTaggedImageReference(image)
	if err != nil {
		log.Errorf(ctx, "parse custom runtime image %s failed: %v", image, err)
		return nil // nolint: nilerr
	}

	// 未落在生效镜像源路径下的是官方镜像，不写 custom_runtime_images
	matches, err := m.checker.MatchesWorkspaceRegistry(ctx, workspaceID, ref.Name)
	if err != nil {
		return errors.Wrapf(err, "match image %s against workspace %s registry", ref.Name, workspaceID)
	}
	if !matches {
		return nil
	}

	if err = m.upsertRecord(ctx, workspaceID, imageType, ref.Name); err != nil {
		return err
	}
	return m.refreshIfNeeded(ctx, workspaceID, ref.Name)
}

// upsertRecord 按 workspace + type + name 幂等落库
func (m *PersistManager) upsertRecord(
	ctx context.Context, workspaceID string, imageType ImageType, imageName string,
) error {
	if err := m.store.Upsert(ctx, &Image{
		WorkspaceID: workspaceID,
		Type:        imageType,
		Name:        imageName,
	}); err != nil {
		return errors.Wrapf(err, "upsert custom runtime image %s/%s/%s", workspaceID, imageType, imageName)
	}
	return nil
}

// refreshIfNeeded 在该镜像还没有成功刷新过快照时触发一次初始化刷新。
//
// 判据取快照状态的 LastRefreshedAt 而非「记录是否刚新建」：刷新失败或进程在落库与刷新
// 之间中断时，记录已存在但快照仍是空的，下次保存构建配置必须能重试，否则 TAG 永远为空。
// 并发重复触发由 snapshot 内部的 TrySetRefreshing 兜住
func (m *PersistManager) refreshIfNeeded(ctx context.Context, workspaceID, imageName string) error {
	if m.snapshotSvc == nil {
		return nil
	}

	status, err := m.snapshotSvc.GetWorkspaceSnapshotStatus(ctx, workspaceID, imageName)
	if err != nil {
		return errors.Wrapf(err, "get snapshot status for image %s", imageName)
	}
	if status != nil && status.LastRefreshedAt != nil {
		return nil
	}

	if _, err = m.snapshotSvc.RefreshWorkspaceSnapshots(ctx, workspaceID, imageName); err != nil {
		return errors.Wrapf(err, "refresh snapshots for image %s", imageName)
	}
	return nil
}
