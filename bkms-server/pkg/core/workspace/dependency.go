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

package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/cmdb"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// EnsureBkSystems 保证依赖的蓝鲸项目存在
func EnsureBkSystems(
	ctx context.Context, workspaceID, displayName, bkciProjectID string, bizID int64,
) (*BkSystems, error) {
	if CreateBCSProjectEnabled() {
		// bcs project 与 bkci 无关联，额外创建 bcs 项目
		return ensureBkSystemsWithIndependentBCSProject(ctx, workspaceID, displayName, bkciProjectID, bizID)
	}
	// bcs 与 bkci 共用 project
	return ensureBkSystemsWithSharedProjectIdentity(ctx, workspaceID, bkciProjectID, bizID)
}

func ensureBkSystemsWithSharedProjectIdentity(
	ctx context.Context, workspaceID, bkciProjectID string, bizID int64,
) (*BkSystems, error) {
	cmdbInfo, err := fetchCMDBInfo(ctx, bkciProjectID, bizID)
	if err != nil {
		return nil, errors.Wrap(err, "fetch cmdb cmdbInfo")
	}

	bkciProjMgr := bkci.NewProjectManager(workspaceID)
	// 请求里的 bkciProjectID 表示蓝盾项目 ID：有值绑定，空值新建。
	createProjResp, err := bkciProjMgr.Initialize(ctx, bkciProjectID, cmdbInfo.ObsProductID, cmdbInfo.ObsProductName)
	if err != nil {
		return nil, err
	}

	isBoundExistedBKCIProject := bkciProjectID != ""
	if !isBoundExistedBKCIProject {
		// 只有新建蓝盾项目时才补默认流水线，避免影响用户已有项目。
		if err = initializeDefaultBKCIPipelines(ctx, workspaceID); err != nil {
			return nil, err
		}
		bkciProjectID = createProjResp.Code
	}

	if err = bkrepo.NewProjectManager(workspaceID, auth.MustGetUser(ctx).ID).Initialize(ctx); err != nil {
		return nil, errors.Wrap(err, "init bkrepo project")
	}

	return buildBkSystems(
		createProjResp,
		createProjResp.ID,
		createProjResp.Code,
		bkciProjectID,
		isBoundExistedBKCIProject,
		cmdbInfo,
	), nil
}

func ensureBkSystemsWithIndependentBCSProject(
	ctx context.Context, workspaceID, displayName, bkciProjectID string, bizID int64,
) (*BkSystems, error) {
	// 无需绑定 ObsProductID/ObsProductName
	cmdbInfo := &cmdb.BusinessDetail{}
	if bizID > 0 {
		cmdbInfo.BizID = cast.ToString(bizID)
	}

	// 独立 BCS 模式下，请求里的 bkciProjectID 承载的是待绑定的 BCS project code。
	boundBCSProjectCode := bkciProjectID

	bkciProjMgr := bkci.NewProjectManager(workspaceID)
	// 独立 BCS 模式下蓝盾项目始终新建，传空强制走创建逻辑。
	createProjResp, err := bkciProjMgr.Initialize(ctx, "", cmdbInfo.ObsProductID, cmdbInfo.ObsProductName)
	if err != nil {
		return nil, err
	}
	if err = initializeDefaultBKCIPipelines(ctx, workspaceID); err != nil {
		return nil, err
	}

	bkciProjectID = createProjResp.Code
	if err = bkrepo.NewProjectManager(workspaceID, auth.MustGetUser(ctx).ID).Initialize(ctx); err != nil {
		return nil, errors.Wrap(err, "init bkrepo project")
	}

	bcsProject, err := resolveIndependentBCSProject(ctx, displayName, boundBCSProjectCode, createProjResp.Code, bizID)
	if err != nil {
		return nil, errors.Wrap(err, "resolve bcs project")
	}

	return buildBkSystems(createProjResp, bcsProject.ID, bcsProject.Code, bkciProjectID, false, cmdbInfo), nil
}

func buildBkSystems(
	createProjResp *bkci.Project, bcsProjectID, bcsProjectCode, bkciProjectID string,
	isBoundExistedBKCIProject bool, cmdbInfo *cmdb.BusinessDetail,
) *BkSystems {
	return &BkSystems{
		// 蓝盾项目 Code, 可读唯一字符串，如：bkce
		BkCIProjectID: bkciProjectID,
		// 蓝盾项目 UID (32 位字符串)
		BkCIProjectUID: createProjResp.ID,
		// BkRepo 项目 ID 使用蓝盾项目可读 code (如 bkce)
		BkRepoProjectID: createProjResp.Code,
		// BCS 项目 ID：关闭 createBCSProject 时复用蓝盾 UID，开启时用 BCS 返回值
		BkBCSProjectID: bcsProjectID,
		// BCS 项目 Code：关闭 createBCSProject 时复用蓝盾 code，开启时用 BCS 返回值
		BkBCSProjectCode: bcsProjectCode,
		// 表明用户创建项目时是否绑定了已有的蓝盾项目
		IsBoundExistedBKCIProject: isBoundExistedBKCIProject,
		// 运营产品 ID
		ObsProductID: cmdbInfo.ObsProductID,
		// 运营产品名称
		ObsProductName: cmdbInfo.ObsProductName,
		// bkcc ID
		BkCCBizID: cmdbInfo.BizID,
		// 二级业务 ID
		Level2BizID: cmdbInfo.Level2BizID,
	}
}

func initializeDefaultBKCIPipelines(ctx context.Context, workspaceID string) error {
	_, err := bkci.NewPipelineManager(workspaceID).Initialize(ctx, string(bkci.PipelineTypeDockerfile))
	if err != nil {
		return errors.Wrap(err, "create builtin dockerfile pipeline")
	}
	_, err = bkci.NewPipelineManager(workspaceID).Initialize(ctx, string(bkci.PipelineTypeHelmGitBuild))
	if err != nil {
		return errors.Wrap(err, "create builtin helm-git-build pipeline")
	}
	return nil
}

func resolveIndependentBCSProject(
	ctx context.Context, displayName, boundBCSProjectCode, projectCode string, bizID int64,
) (*bcs.Project, error) {
	user := auth.MustGetUser(ctx)
	bcsClient, err := bcs.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial bcs client")
	}
	// 有绑定 code 表示复用已有 BCS 项目；空值则为当前新蓝盾项目创建新的 BCS 项目。
	if boundBCSProjectCode != "" {
		return getExistingBCSProject(ctx, bcsClient, boundBCSProjectCode)
	}
	return createBCSProject(ctx, bcsClient, displayName, projectCode, bizID)
}

// CreateExternalImageRegistry 创建外部镜像仓库信息
// 注：内置的镜像仓库在初始化 bkrepo 项目的时候，由 bkrepo.ProjectManager 创建
func CreateExternalImageRegistry(
	ctx context.Context,
	workspaceID string,
	bkciProjectID string,
	store bkmsreg.ImageRegistryStore,
	registry, username, password string,
) error {
	if registry == "" || username == "" || password == "" {
		return errors.Errorf("registry, username, password are required")
	}

	credID := strings.ReplaceAll(uuid.NewString(), "-", "")
	description := fmt.Sprintf("bkms image credential for registry %s", registry)

	client, err := bkciapi.New(auth.MustGetUser(ctx))
	if err != nil {
		return errors.Wrap(err, "create bkci client")
	}
	// 预先调用蓝盾 API 创建凭证（用于推送镜像）
	if err = client.CreateCredential(ctx, bkciProjectID, credID, description, username, password); err != nil {
		return errors.Wrap(err, "create bkci credential")
	}

	imageReg := &bkmsreg.ImageRegistry{
		WorkspaceID:      workspaceID,
		Type:             bkmsreg.ImageRegistryTypeExternal,
		Registry:         registry,
		Username:         username,
		Password:         password,
		BkCICredentialID: credID,
	}
	if _, err = store.Create(ctx, imageReg); err != nil {
		return errors.Wrap(err, "create image registry")
	}
	return nil
}

// InitWorkspaceUser 初始化 workspace 默认设计的四类用户：管理员、开发者、SRE、运营
func InitWorkspaceUser(ctx context.Context, id, displayName string, managers []string, bkSystem BkSystems) error {
	permMgr := perm.NewManager()

	// 创建管理员角色
	if err := permMgr.CreateWorkspaceAdmin(
		ctx, id, displayName, managers, bkSystem.BkCIProjectID, bkSystem.BkBCSProjectID, bkSystem.BkRepoProjectID,
	); err != nil {
		return err
	}

	// 创建内置角色（developer, sre, operator）
	if err := permMgr.CreateWorkspaceScopeBuiltinRoles(
		ctx, id, displayName, bkSystem.BkCIProjectID, bkSystem.BkBCSProjectID, bkSystem.BkRepoProjectID,
	); err != nil {
		return err
	}

	return nil
}

// CreateBCSProjectEnabled 为 true 时由 BKMS 独立创建/绑定 BCS 项目，不再复用蓝盾项目标识。
func CreateBCSProjectEnabled() bool {
	return config.G != nil && config.G.FeatureFlags.CreateBCSProject
}

func getExistingBCSProject(ctx context.Context, client bcs.Client, projectCode string) (*bcs.Project, error) {
	project, err := client.GetProject(ctx, projectCode)
	if err != nil {
		return nil, errors.Wrapf(err, "get bcs project %s", projectCode)
	}
	if project == nil || project.ID == "" {
		return nil, errors.New("get bcs project returned empty id")
	}
	log.Infof(ctx, "bind existing bcs project: code=%s id=%s", project.Code, project.ID)
	return project, nil
}

func createBCSProject(
	ctx context.Context, client bcs.Client, displayName, projectCode string, bizID int64,
) (*bcs.Project, error) {
	name := displayName
	if name == "" {
		name = projectCode
	}
	in := bcs.CreateProjectInput{
		Name:        name,
		ProjectCode: projectCode,
		Kind:        bcs.ProjectKindK8s,
	}
	if bizID > 0 {
		in.BusinessID = cast.ToString(bizID)
	}
	log.Infof(ctx, "create bcs project: code=%s name=%s", projectCode, name)
	created, err := client.CreateProject(ctx, in)
	if err != nil {
		return nil, errors.Wrapf(err, "create bcs project %s", projectCode)
	}
	return created, nil
}

// fetchCMDBInfo 查询 CMDB 相关字段（二级业务 ID、运营产品 ID/名称）
func fetchCMDBInfo(ctx context.Context, bkciProjectID string, bizID int64) (*cmdb.BusinessDetail, error) {
	user := auth.MustGetUser(ctx)

	// 当 bizID 未传入时，通过 bkciProjectID 从 BCS 获取项目关联的业务 ID
	if bizID <= 0 && bkciProjectID != "" {
		bcsClient, err := bcs.New(user)
		if err != nil {
			return nil, errors.Wrap(err, "initial bcs client")
		}
		project, err := bcsClient.GetProject(ctx, bkciProjectID)
		if err != nil {
			return nil, errors.Wrapf(err, "get bcs project by bkciProjectID: %s", bkciProjectID)
		}
		bizID = cast.ToInt64(project.BizID)
		if bizID <= 0 {
			return nil, errors.Errorf("bcs project(%s) has no associated bizID", bkciProjectID)
		}
	}

	cmdbSvc, err := cmdb.NewService(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial cmdb service")
	}

	return cmdbSvc.GetCMDBInfo(ctx, bizID)
}
