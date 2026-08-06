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

package helm

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
	helmrepo "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/source"
)

// 解析获取 HelmRepo 配置（根据不同的原始类型）
func resolveRepoConfig(
	ctx context.Context,
	bkciProjectStore bkci.ProjectStore,
	bkrepoProjectStore bkrepo.ProjectStore,
	credentialStore credential.HelmRepoCredentialStore,
	app *bkmsapp.Application,
) (*bkmsapp.HelmRepoConfig, error) {
	if err := validateApp(app); err != nil {
		return nil, errors.Wrap(err, "invalid app")
	}

	repoConfig, err := helmrepo.ResolveConfig(ctx, bkciProjectStore, bkrepoProjectStore, credentialStore, app)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve helm repo config for app %s in workspace %s", app.ID, app.WorkspaceID)
	}
	if repoConfig == nil {
		return nil, errors.New("helm repo config is nil")
	}
	if repoConfig.RepoURL == "" {
		return nil, errors.New("helm repo config repo url is empty")
	}
	if repoConfig.ChartName == "" {
		return nil, errors.New("helm repo config chart name is empty")
	}
	return repoConfig, nil
}
