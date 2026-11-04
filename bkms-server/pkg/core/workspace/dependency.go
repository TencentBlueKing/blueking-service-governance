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
	"go.mongodb.org/mongo-driver/v2/mongo"

	bcsmanager "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/cmdb"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// EnsureBkSystems 保证依赖的蓝鲸项目存在
func EnsureBkSystems(
	ctx context.Context, db *mongo.Database, workspaceID, displayName, bkciProjectID string, bizID int64,
) (*BkSystems, error) {
	if IndependentBCSProjectEnabled() {
		// bcs project 与 bkci 无关联，额外创建 bcs 项目
		return ensureBkSystemsWithIndependentBCSProject(ctx, db, workspaceID, displayName, bkciProjectID, bizID)
	}
	// bcs 与 bkci 共用 project
	return ensureBkSystemsWithSharedProjectIdentity(ctx, db, workspaceID, bkciProjectID, bizID)
}

func ensureBkSystemsWithSharedProjectIdentity(
	ctx context.Context, db *mongo.Database, workspaceID, bkciProjectID string, bizID int64,
) (*BkSystems, error) {
	cmdbInfo, err := fetchCMDBInfo(ctx, bkciProjectID, bizID)
	if err != nil {
		return nil, errors.Wrap(err, "fetch cmdb cmdbInfo")
	}

	bkciProjMgr := bkci.NewProjectManager(db, workspaceID)
	// 请求里的 bkciProjectID 表示蓝盾项目 ID：有值绑定，空值新建。
	createProjResp, err := bkciProjMgr.Initialize(ctx, bkciProjectID, cmdbInfo.ObsProductID, cmdbInfo.ObsProductName)
	if err != nil {
		return nil, err
	}

	isBoundExistedBKCIProject := bkciProjectID != ""
	if !isBoundExistedBKCIProject {
		// 只有新建蓝盾项目时才补默认流水线，避免影响用户已有项目。
		if err = initializeDefaultBKCIPipelines(ctx, db, workspaceID); err != nil {
			return nil, err
		}
		bkciProjectID = createProjResp.Code
	}

	if err = bkrepo.NewProjectManager(db, workspaceID, auth.MustGetUser(ctx).ID).Initialize(ctx); err != nil {
		return nil, errors.Wrap(err, "init bkrepo project")
	}

	// 绑定 BSCP Default 项目并确保 credential 存在
	bscpBinding, err := BindBscpProject(ctx, cmdbInfo.BizID, "")
	if err != nil {
		return nil, errors.Wrap(err, "bind bscp project")
	}

	return buildBkSystems(
		createProjResp,
		createProjResp.ID,
		createProjResp.Code,
		bkciProjectID,
		isBoundExistedBKCIProject,
		cmdbInfo,
		bscpBinding,
	), nil
}

func ensureBkSystemsWithIndependentBCSProject(
	ctx context.Context, db *mongo.Database, workspaceID, displayName, bkciProjectID string, bizID int64,
) (*BkSystems, error) {
	workspaces := store.Of(db, Meta)
	// 独立 BCS 模式下，请求里的 bkciProjectID 承载的是待绑定的 BCS project code；
	// 为空则表示由服务端为当前 workspace 确保一个独立 BCS 项目存在。
	boundBCSProject, cmdbInfo, err := prepareIndependentBCSBinding(ctx, workspaces, workspaceID, bkciProjectID, bizID)
	if err != nil {
		return nil, errors.Wrap(err, "prepare independent bcs binding")
	}

	createProjResp, err := initializeIndependentBKCIResources(ctx, db, workspaceID)
	if err != nil {
		return nil, err
	}

	bcsProject, err := resolveIndependentBCSProjectForWorkspace(
		ctx, workspaces, workspaceID, displayName, bizID, boundBCSProject,
	)
	if err != nil {
		return nil, errors.Wrap(err, "resolve workspace bcs project")
	}

	// 绑定 BSCP Default 项目并确保 credential 存在
	bscpBinding, err := BindBscpProject(ctx, cmdbInfo.BizID, "")
	if err != nil {
		return nil, errors.Wrap(err, "bind bscp project")
	}

	return buildBkSystems(
		createProjResp,
		bcsProject.ID,
		bcsProject.Code,
		createProjResp.Code,
		false,
		cmdbInfo,
		bscpBinding,
	), nil
}

func prepareIndependentBCSBinding(
	ctx context.Context, workspaces *WorkspaceStore, workspaceID, boundBCSProjectCode string, bizID int64,
) (*bcs.Project, *cmdb.BusinessDetail, error) {
	if boundBCSProjectCode == "" {
		if bizID <= 0 {
			return nil, nil, ErrIndependentWorkspaceBizRequired
		}
		if err := validateIndependentWorkspaceBiz(ctx, bizID); err != nil {
			return nil, nil, err
		}
		// 自动创建前确认确定性 BCS projectID 不存在，或属于本次重试可复用的项目。
		// 业务不一致或已被其他 workspace 绑定时，在创建蓝盾资源之前失败。
		if err := ensureAutoCreateBCSProjectReusable(ctx, workspaces, workspaceID, bizID); err != nil {
			return nil, nil, err
		}
		// 独立 BCS 模式只补齐业务 ID，不再回填二级业务/运营产品。
		return nil, &cmdb.BusinessDetail{BizID: cast.ToString(bizID)}, nil
	}

	project, err := bcsmanager.NewProjectManager().LoadBoundProject(ctx, boundBCSProjectCode)
	if err != nil {
		return nil, nil, errors.Wrap(err, "load bound bcs project")
	}
	if cast.ToInt64(project.BizID) <= 0 {
		return nil, nil, errors.Errorf("bcs project(%s) has no associated bizID", boundBCSProjectCode)
	}
	// 显式绑定已有 BCS 项目时，先校验是否已被其他 workspace 占用，避免后续先创建外部资源再失败。
	if err = ensureBCSProjectNotBound(ctx, workspaces, workspaceID, project); err != nil {
		return nil, nil, errors.Wrap(err, "ensure bcs project not bound")
	}
	return project, &cmdb.BusinessDetail{BizID: project.BizID}, nil
}

func initializeIndependentBKCIResources(
	ctx context.Context, db *mongo.Database, workspaceID string,
) (*bkci.Project, error) {
	// 独立 BCS 模式面向没有 TOF 的环境，创建蓝盾项目不查询用户组织
	bkciProjMgr := bkci.NewProjectManager(db, workspaceID).SkipUserOrganization()
	// 独立 BCS 模式下蓝盾项目始终新建，传空强制走创建逻辑。
	createProjResp, err := bkciProjMgr.Initialize(ctx, "", "", "")
	if err != nil {
		return nil, err
	}
	if err = initializeDefaultBKCIPipelines(ctx, db, workspaceID); err != nil {
		return nil, err
	}
	if err = bkrepo.NewProjectManager(db, workspaceID, auth.MustGetUser(ctx).ID).Initialize(ctx); err != nil {
		return nil, errors.Wrap(err, "init bkrepo project")
	}
	return createProjResp, nil
}

func resolveIndependentBCSProjectForWorkspace(
	ctx context.Context,
	workspaces *WorkspaceStore,
	workspaceID, displayName string,
	bizID int64,
	boundBCSProject *bcs.Project,
) (*bcs.Project, error) {
	if boundBCSProject != nil {
		return boundBCSProject, nil
	}
	return ensureIndependentBCSProject(ctx, workspaces, workspaceID, displayName, bizID)
}

func buildBkSystems(
	createProjResp *bkci.Project, bcsProjectID, bcsProjectCode, bkciProjectID string,
	isBoundExistedBKCIProject bool, cmdbInfo *cmdb.BusinessDetail, bscpBinding *BscpBinding,
) *BkSystems {
	return &BkSystems{
		// 蓝盾项目 Code, 可读唯一字符串，如：bkce
		BkCIProjectID: bkciProjectID,
		// 蓝盾项目 UID (32 位字符串)
		BkCIProjectUID: createProjResp.ID,
		// BkRepo 项目 ID 使用蓝盾项目可读 code (如 bkce)
		BkRepoProjectID: createProjResp.Code,
		// BCS 项目 ID：关闭独立 BCS 项目时复用蓝盾 UID，开启时用 BCS 返回值
		BkBCSProjectID: bcsProjectID,
		// BCS 项目 Code：关闭独立 BCS 项目时复用蓝盾 code，开启时用 BCS 返回值
		BkBCSProjectCode: bcsProjectCode,
		// BSCP 项目 ID / Key
		BkBSCPProjectID:  bscpBinding.ProjectID,
		BkBSCPProjectKey: bscpBinding.ProjectKey,
		// BSCP Credential ID / Token
		BscpToken:        bscpBinding.Token,
		BscpCredentialID: bscpBinding.CredentialID,
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

func initializeDefaultBKCIPipelines(ctx context.Context, db *mongo.Database, workspaceID string) error {
	_, err := bkci.NewPipelineManager(db, workspaceID).Initialize(ctx, string(bkci.PipelineTypeDockerfile))
	if err != nil {
		return errors.Wrap(err, "create builtin dockerfile pipeline")
	}
	_, err = bkci.NewPipelineManager(db, workspaceID).Initialize(ctx, string(bkci.PipelineTypeHelmGitBuild))
	if err != nil {
		return errors.Wrap(err, "create builtin helm-git-build pipeline")
	}
	return nil
}

func validateIndependentWorkspaceBiz(ctx context.Context, bizID int64) error {
	cmdbSvc, err := cmdb.NewService(auth.MustGetUser(ctx))
	if err != nil {
		return errors.Wrap(err, "initial cmdb service")
	}
	if _, err = cmdbSvc.GetBusinessByID(ctx, bizID); err != nil {
		return errors.Wrapf(ErrIndependentWorkspaceBizInvalid, "validate bkcc biz %d: %v", bizID, err)
	}
	return nil
}

func ensureAutoCreateBCSProjectReusable(
	ctx context.Context, workspaces *WorkspaceStore, workspaceID string, bizID int64,
) error {
	projectID, err := independentBCSProjectID(ctx, workspaceID)
	if err != nil {
		return err
	}
	project, err := bcsmanager.NewProjectManager().LoadBoundProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, bcs.ErrProjectNotFound) {
			return nil
		}
		return errors.Wrapf(err, "check bcs project %s before create", projectID)
	}
	if cast.ToInt64(project.BizID) != bizID {
		return errors.Wrapf(
			ErrBCSProjectAlreadyExists,
			"bcs project(%s) belongs to biz %s, not requested biz %d",
			project.Code,
			project.BizID,
			bizID,
		)
	}
	if err = ensureBCSProjectNotBound(ctx, workspaces, workspaceID, project); err != nil {
		return errors.Wrap(err, "ensure bcs project not bound")
	}
	return nil
}

// independentBCSProjectID 返回工作空间独立 BCS 项目的确定性 projectID，自动创建与重试回查都使用它
func independentBCSProjectID(ctx context.Context, workspaceID string) (string, error) {
	tenantID, ok := tenant.GetTenantID(ctx)
	if !ok {
		return "", errors.Wrapf(tenant.ErrTenantIDRequired, "bcs project id of workspace %s", workspaceID)
	}
	return bcsmanager.WorkspaceProjectID(tenantID, workspaceID), nil
}

func ensureIndependentBCSProject(
	ctx context.Context, workspaces *WorkspaceStore, workspaceID, displayName string, bizID int64,
) (*bcs.Project, error) {
	projectID, err := independentBCSProjectID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	project, err := bcsmanager.NewProjectManager().EnsureProject(ctx, bcsmanager.ProjectSpec{
		ID:    projectID,
		Code:  bkci.DefaultProjectCode(workspaceID),
		Name:  bcsmanager.WorkspaceProjectName(displayName, workspaceID),
		BizID: bizID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "ensure bcs project")
	}
	if err = ensureBCSProjectNotBound(ctx, workspaces, workspaceID, project); err != nil {
		return nil, errors.Wrap(err, "ensure bcs project not bound")
	}
	return project, nil
}

// CreateExternalImageRegistry 创建外部镜像仓库信息
// 注：内置的镜像仓库在初始化 bkrepo 项目的时候，由 bkrepo.ProjectManager 创建
func CreateExternalImageRegistry(
	ctx context.Context,
	workspaceID string,
	bkciProjectID string,
	db *mongo.Database,
	registry, username, password string,
) error {
	if registry == "" || username == "" || password == "" {
		return errors.Errorf("registry, username, password are required")
	}

	// 预先调用蓝盾 API 创建凭证（用于推送镜像）
	credID, err := CreateImageCredential(ctx, bkciProjectID, registry, username, password)
	if err != nil {
		return err
	}

	imageReg := &bkmsreg.ImageRegistry{
		WorkspaceID:      workspaceID,
		Type:             bkmsreg.ImageRegistryTypeExternal,
		Registry:         registry,
		Username:         username,
		Password:         password,
		BkCICredentialID: credID,
	}
	if err = store.Of(db, bkmsreg.ImageRegistryMeta).Create(ctx, imageReg); err != nil {
		return errors.Wrap(err, "create image registry")
	}
	return nil
}

// CreateImageCredential 以当前用户身份在蓝盾项目下创建镜像仓库账密凭证，返回新凭证 ID
//
// 凭证 ID 为去掉连字符的 uuid，每次调用都新建，不复用同名凭证
func CreateImageCredential(
	ctx context.Context, bkciProjectID, registry, username, password string,
) (string, error) {
	client, err := bkciapi.New(auth.MustGetUser(ctx))
	if err != nil {
		return "", errors.Wrap(err, "create bkci client")
	}

	credID := strings.ReplaceAll(uuid.NewString(), "-", "")
	description := fmt.Sprintf("bkms image credential for registry %s", registry)
	if err = client.CreateCredential(ctx, bkciProjectID, credID, description, username, password); err != nil {
		return "", errors.Wrapf(err, "create bkci credential in project %s", bkciProjectID)
	}
	return credID, nil
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

// IndependentBCSProjectEnabled 为 true 时由 BKMS 独立创建/绑定 BCS 项目，不再复用蓝盾项目标识。
func IndependentBCSProjectEnabled() bool {
	return config.G != nil && config.G.FeatureIntegrations.EnableIndependentBCSProject
}

// ensureBCSProjectNotBound 检查 BCS 项目未被其他工作空间绑定。
// 存储的 projectID / projectCode 都是 BCS 返回值，全局唯一（多租户下 projectCode 为 {tenantID}-{code}），
// 因此不按租户过滤，与唯一索引 bkSystems.bkBcsProjectCode 的范围保持一致
func ensureBCSProjectNotBound(
	ctx context.Context, workspaces *WorkspaceStore, workspaceID string, project *bcs.Project,
) error {
	if project == nil {
		return errors.New("bcs project is nil")
	}
	boundWorkspace, err := workspaces.Get(ctx, store.TenantAll(), ByBCSProject(project.ID, project.Code))
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			return nil
		}
		return errors.Wrapf(err, "find workspace by bcs project %s", project.Code)
	}
	if boundWorkspace.ID == workspaceID {
		return nil
	}
	return errors.Wrapf(
		ErrBCSProjectAlreadyBound,
		"bcs project(%s) already bound by workspace %s",
		project.Code,
		boundWorkspace.ID,
	)
}

func getBCSProjectBizID(ctx context.Context, projectCode string, bizID int64) (string, error) {
	if bizID > 0 {
		return cast.ToString(bizID), nil
	}
	if projectCode == "" {
		return "", nil
	}

	user := auth.MustGetUser(ctx)
	bcsClient, err := bcs.New(user)
	if err != nil {
		return "", errors.Wrap(err, "initial bcs client")
	}
	project, err := bcsClient.GetProject(ctx, projectCode)
	if err != nil {
		return "", errors.Wrapf(err, "get bcs project by code: %s", projectCode)
	}

	resolvedBizID := cast.ToInt64(project.BizID)
	if resolvedBizID <= 0 {
		return "", errors.Errorf("bcs project(%s) has no associated bizID", projectCode)
	}
	return cast.ToString(resolvedBizID), nil
}

// fetchCMDBInfo 查询 CMDB 相关字段（二级业务 ID、运营产品 ID/名称）
func fetchCMDBInfo(ctx context.Context, bkciProjectID string, bizID int64) (*cmdb.BusinessDetail, error) {
	user := auth.MustGetUser(ctx)

	// 当 bizID 未传入时，通过 BCS 项目获取其关联的业务 ID。
	if bizID <= 0 {
		resolvedBizID, err := getBCSProjectBizID(ctx, bkciProjectID, bizID)
		if err != nil {
			return nil, errors.Wrap(err, "get bcs bizID")
		}
		bizID = cast.ToInt64(resolvedBizID)
	}

	cmdbSvc, err := cmdb.NewService(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial cmdb service")
	}

	return cmdbSvc.GetCMDBInfo(ctx, bizID)
}
