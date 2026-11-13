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

package bcs

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbcs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// ErrProjectNotK8s BCS 项目不是 k8s 类型，不能绑定到工作空间
var ErrProjectNotK8s = errors.New("bcs project is not a k8s project")

// ErrProjectBizMismatch 同 code 的 BCS 项目已存在，但关联的业务与请求不一致，不能复用
var ErrProjectBizMismatch = errors.New("existing bcs project belongs to another biz")

// AddBCSProjectTenantPrefix 为项目 Code 补齐租户前缀，得到 BCS 侧的项目 Code（多租户 BCS 为 {tenantID}-{code}）。
// 多租户下 BCS 创建项目时会按该规则改写 projectCode，按补齐后的 Code 即可回查此前创建的项目。
// 已带前缀时原样返回。
// 参考 bcs https://github.com/TencentBlueKing/bk-bcs/blob/5cf513fc90cb60332265087b361032c76ef37dd5/bcs-services
//
//	/bcs-project-manager/internal/actions/project/create.go#L182
func AddBCSProjectTenantPrefix(ctx context.Context, code string) string {
	prefix := getBCSProjectTenantPrefix(ctx)
	if prefix == "" || strings.HasPrefix(code, prefix) {
		return code
	}
	return prefix + code
}

// getBCSProjectTenantPrefix 返回多租户 BCS 项目 Code 的租户前缀（{tenantID}-），非多租户模式或 ctx 无租户时返回空
func getBCSProjectTenantPrefix(ctx context.Context) string {
	if config.G == nil || !config.G.Tenant.EnableMultiTenantMode {
		return ""
	}
	tenantID, ok := tenant.GetTenantID(ctx)
	if !ok {
		return ""
	}
	return tenantID + "-"
}

// WorkspaceProjectName 生成工作空间独立 BCS 项目的名称。
// BCS 项目名全局唯一且长度为 2-64，带上全局唯一的工作空间 ID，避免同名工作空间冲突
func WorkspaceProjectName(displayName, workspaceID string) string {
	return fmt.Sprintf("%s(%s)", displayName, workspaceID)
}

// ProjectSpec 待确保存在的 BCS 项目
type ProjectSpec struct {
	// Code 创建时传给 BCS 的 code（不带租户前缀），查询时使用补齐租户前缀后的 code
	Code  string
	Name  string
	BizID int64
}

// ProjectManager 负责 BCS 项目的查询、校验与幂等 ensure。
type ProjectManager struct{}

// NewProjectManager 创建project manager
func NewProjectManager() *ProjectManager {
	return &ProjectManager{}
}

// LoadBoundProject 按 projectID 或 BCS 返回的 projectCode 加载一个已存在且可绑定的 BCS 项目。
func (m *ProjectManager) LoadBoundProject(ctx context.Context, projectIDOrCode string) (*cloudbcs.Project, error) {
	client, err := cloudbcs.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "initial bcs client")
	}
	return getExistingProject(ctx, client, projectIDOrCode)
}

// EnsureProject 确保 spec.Code 对应的 BCS 项目存在，已存在时校验业务一致后复用。
func (m *ProjectManager) EnsureProject(ctx context.Context, spec ProjectSpec) (*cloudbcs.Project, error) {
	client, err := cloudbcs.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "initial bcs client")
	}

	projectCode := AddBCSProjectTenantPrefix(ctx, spec.Code)
	project, err := getExistingProject(ctx, client, projectCode)
	if err == nil {
		if spec.BizID > 0 && cast.ToInt64(project.BizID) != spec.BizID {
			return nil, errors.Wrapf(
				ErrProjectBizMismatch,
				"bcs project(%s) belongs to biz %s, not requested biz %d",
				project.Code,
				project.BizID,
				spec.BizID,
			)
		}
		return project, nil
	}
	if !errors.Is(err, cloudbcs.ErrProjectNotFound) {
		return nil, errors.Wrap(err, "check bcs project before create")
	}
	return createProject(ctx, client, spec)
}

// getExistingProject 加载已存在的 k8s 项目；client.GetProject 的错误里已带项目 ID/Code，这里不再重复
func getExistingProject(
	ctx context.Context, client cloudbcs.Client, projectIDOrCode string,
) (*cloudbcs.Project, error) {
	project, err := client.GetProject(ctx, projectIDOrCode)
	if err != nil {
		return nil, errors.Wrap(err, "load bcs project")
	}
	if project == nil || project.ID == "" {
		return nil, errors.Errorf("bcs project(%s) returned empty id", projectIDOrCode)
	}
	if project.Kind != cloudbcs.ProjectKindK8s {
		return nil, errors.Wrapf(ErrProjectNotK8s, "bcs project(%s)", projectIDOrCode)
	}
	return project, nil
}

func createProject(ctx context.Context, client cloudbcs.Client, spec ProjectSpec) (*cloudbcs.Project, error) {
	in := cloudbcs.CreateProjectInput{
		Name:        spec.Name,
		ProjectCode: spec.Code,
		Kind:        cloudbcs.ProjectKindK8s,
	}
	if spec.BizID > 0 {
		in.BusinessID = cast.ToString(spec.BizID)
	}

	created, err := client.CreateProject(ctx, in)
	if err != nil {
		return nil, errors.Wrapf(err, "create bcs project %s", spec.Code)
	}
	return created, nil
}
