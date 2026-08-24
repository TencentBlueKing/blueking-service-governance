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
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	infrasreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// ExistenceChecker 判断镜像是否属于工作空间生效镜像源，并向仓库确认 name + tag 存在
type ExistenceChecker struct {
	snapshotSvc *snapshot.Service
}

// NewExistenceChecker 创建 ExistenceChecker
func NewExistenceChecker(snapshotSvc *snapshot.Service) *ExistenceChecker {
	return &ExistenceChecker{snapshotSvc: snapshotSvc}
}

// MatchesWorkspaceRegistry 不含 tag 的镜像名是否落在工作空间生效镜像源路径下
//
// 未绑定镜像源时返回 false，交给官方口径处理，不在这里报错
func (c *ExistenceChecker) MatchesWorkspaceRegistry(
	ctx context.Context, workspaceID, imageName string,
) (bool, error) {
	if c == nil {
		return false, nil
	}
	reg, err := c.lookupRegistry(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	if reg == nil {
		return false, nil
	}
	return nameBelongsToRegistry(imageName, reg.Registry), nil
}

// ValidateTaggedReference 向生效镜像源确认镜像名与 tag 都真实存在
func (c *ExistenceChecker) ValidateTaggedReference(
	ctx context.Context, workspaceID, image string,
) error {
	if c == nil || c.snapshotSvc == nil {
		return errors.New("custom runtime image existence checker is not initialized")
	}

	ref, err := workloadruntime.ParseTaggedImageReference(image)
	if err != nil {
		return err
	}

	info, err := c.snapshotSvc.ResolveRepoKeyForWorkspace(ctx, workspaceID, ref.Name)
	if err != nil {
		return errors.Wrap(err, "resolve workspace registry credentials")
	}

	client := infrasreg.New(info.Username, info.Password, true)
	if err = client.HeadManifest(info.RepoName, ref.Tag); err == nil {
		return nil
	}

	if classified := classifyRegistryAccessError(err); classified != nil &&
		!errors.Is(classified, ErrImageTagNotFound) {
		return classified
	}
	if !infrasreg.IsTagNotFound(err) {
		return errors.Wrap(ErrRegistryAccessFailed, err.Error())
	}

	// Head 404 时再列 tag：列表成功说明镜像名在、tag 不在；列表也 404 则镜像名不存在
	_, listErr := client.ListAllTags(info.RepoName)
	if listErr == nil {
		return errors.Wrapf(ErrImageTagNotFound, "tag %s", ref.Tag)
	}
	if infrasreg.IsTagNotFound(listErr) {
		return errors.Wrapf(ErrImageNameNotFound, "image %s", ref.Name)
	}
	if classified := classifyRegistryAccessError(listErr); classified != nil {
		return classified
	}
	return errors.Wrap(ErrRegistryAccessFailed, listErr.Error())
}

// lookupRegistry 取工作空间生效镜像源
//
// 未绑定、地址为空都视为没有镜像源，返回 nil 交给官方口径，不在这里报错
func (c *ExistenceChecker) lookupRegistry(
	ctx context.Context, workspaceID string,
) (*bkmsreg.ImageRegistry, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, nil
	}
	reg, err := workspace.GetWorkspaceImageRegistry(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, bkmsreg.ErrImageRegistryNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "get workspace image registry")
	}
	if reg == nil || strings.TrimSpace(reg.Registry) == "" {
		return nil, nil
	}
	return reg, nil
}

// classifyRegistryAccessError 把仓库错误归到鉴权失败或 tag 不存在
//
// 404 不在这里当成镜像名不存在，调用方还要再 ListAllTags 才能区分；其余错误返回 nil，按仓库访问失败处理
func classifyRegistryAccessError(err error) error {
	if infrasreg.IsAuthRequired(err) {
		return errors.Wrap(ErrRegistryAccessDenied, err.Error())
	}
	if infrasreg.IsTagNotFound(err) {
		return ErrImageTagNotFound
	}
	return nil
}

// registryPathPrefix 生成带路径边界的前缀，避免 .../repo 误匹配 .../repo-evil
func registryPathPrefix(registryAddr string) string {
	return strings.TrimRight(strings.TrimSpace(registryAddr), "/") + "/"
}

// nameBelongsToRegistry 判断不含 tag 的镜像名是否落在生效镜像源路径下
func nameBelongsToRegistry(imageName, registryAddr string) bool {
	imageName = strings.TrimSpace(imageName)
	registryAddr = strings.TrimSpace(registryAddr)
	if imageName == "" || registryAddr == "" {
		return false
	}
	return strings.HasPrefix(imageName, registryPathPrefix(registryAddr))
}
