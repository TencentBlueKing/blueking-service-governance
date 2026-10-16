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

package repo

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
)

// ResolveConfig 根据应用的 HelmSource.RepoType，将不同类型的配置统一转换为 HelmRepoConfig。
// HelmGitRepo 类型会读取 bkci.Project、bkrepo.Project，并读写 credential.HelmRepoCredential（缺失时初始化）
func ResolveConfig(
	ctx context.Context,
	db *mongo.Database,
	app *bkmsapp.Application,
) (*bkmsapp.HelmRepoConfig, error) {
	hs := app.HelmSpec.HelmSource

	switch hs.RepoType {
	case bkmsapp.HelmSourceRepoTypeGit:
		// HelmGitRepo 类型：从工作空间绑定的 bkrepo HELM 仓库获取配置
		bkciProj, err := store.Of(db, bkci.ProjectMeta).Get(ctx, bkci.ByWorkspace(app.WorkspaceID))
		if err != nil {
			return nil, errors.Wrapf(err, "get bkci project for workspace %s", app.WorkspaceID)
		}

		helmRepoURL, err := config.G.Helm.GenBuiltinRepoURL(bkciProj.Code)
		if err != nil {
			return nil, errors.Wrap(err, "get helm repo endpoint")
		}

		// 确保凭证已初始化（幂等操作）
		bkrepoProj, err := store.Of(db, bkrepo.ProjectMeta).Get(ctx, bkrepo.ByWorkspace(app.WorkspaceID))
		if err != nil {
			return nil, errors.Wrapf(err, "get bkrepo project for workspace %s", app.WorkspaceID)
		}
		if err = credential.EnsureCredential(
			ctx, db, app.WorkspaceID, bkciProj.Code, bkrepoProj.Username, bkrepoProj.Password,
		); err != nil {
			return nil, errors.Wrapf(err, "ensure helm repo credential for workspace %s", app.WorkspaceID)
		}

		// 从凭证 store 获取用户名密码
		cred, err := store.Of(db, credential.Meta).Get(ctx, credential.ByWorkspace(app.WorkspaceID))
		if err != nil {
			return nil, errors.Wrapf(err, "get helm repo credential for workspace %s", app.WorkspaceID)
		}

		return &bkmsapp.HelmRepoConfig{
			RepoURL:   helmRepoURL,
			ChartName: app.Name,
			Username:  cred.Username,
			Password:  cred.Password,
		}, nil

	case bkmsapp.HelmSourceRepoTypeHelm:
		// HelmRepo 类型：直接使用 HelmRepoConfig
		cfg := hs.HelmRepoConfig
		return &bkmsapp.HelmRepoConfig{
			RepoURL:   cfg.RepoURL,
			ChartName: cfg.ChartName,
			Username:  cfg.Username,
			Password:  cfg.Password,
		}, nil
	default:
		return nil, errors.Errorf("invalid repo type %s", hs.RepoType)
	}
}
