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

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbcs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

// workspaceProjectIDNamespace 生成工作空间 BCS projectID 的 UUID v5 命名空间，修改会导致已创建项目无法按 ID 回查
var workspaceProjectIDNamespace = uuid.MustParse("752ae529-863a-4659-a883-2e7d4b0f7cfa")

// WorkspaceProjectID 生成工作空间独立 BCS 项目的 projectID（32 位，与 BCS 自动生成的格式一致）。
//
// 多租户下 BCS 会把 projectCode 改写为 {tenantID}-{code}，按请求里的 code 查不到项目；projectID 不会被改写，
// 由 BKMS 指定后，创建失败重试时可以按 ID 找回上一次创建的项目。
// BCS 按 ID 查询不区分租户，ID 需要包含租户，避免不同租户先后使用同一个工作空间 ID 时撞到对方的项目
func WorkspaceProjectID(tenantID, workspaceID string) string {
	id := uuid.NewSHA1(workspaceProjectIDNamespace, []byte(tenantID+"/"+workspaceID))
	return strings.ReplaceAll(id.String(), "-", "")
}

// WorkspaceProjectName 生成工作空间独立 BCS 项目的名称。
// BCS 项目名全局唯一且长度为 2-64，带上全局唯一的工作空间 ID，避免同名工作空间冲突
func WorkspaceProjectName(displayName, workspaceID string) string {
	return fmt.Sprintf("%s(%s)", displayName, workspaceID)
}

// ProjectSpec 待确保存在的 BCS 项目
type ProjectSpec struct {
	// ID 由调用方确定，查询与创建都使用该 ID
	ID string
	// Code 创建时传给 BCS 的 code，多租户下实际 projectCode 以 BCS 返回值为准
	Code  string
	Name  string
	BizID int64
}

// ProjectManager 负责 BCS 项目的查询、校验与幂等 ensure。
type ProjectManager struct{}

func NewProjectManager() *ProjectManager {
	return &ProjectManager{}
}

// LoadBoundProject 按 projectID 或 BCS 返回的 projectCode 加载一个已存在且可绑定的 BCS 项目。
func (m *ProjectManager) LoadBoundProject(ctx context.Context, projectIDOrCode string) (*cloudbcs.Project, error) {
	client, err := m.newClient(ctx)
	if err != nil {
		return nil, err
	}
	return getExistingProject(ctx, client, projectIDOrCode)
}

// EnsureProject 确保 spec.ID 对应的 BCS 项目存在，已存在时校验业务一致后复用。
func (m *ProjectManager) EnsureProject(ctx context.Context, spec ProjectSpec) (*cloudbcs.Project, error) {
	client, err := m.newClient(ctx)
	if err != nil {
		return nil, err
	}

	project, err := getExistingProject(ctx, client, spec.ID)
	if err == nil {
		if spec.BizID > 0 && cast.ToInt64(project.BizID) != spec.BizID {
			return nil, errors.Errorf(
				"existing bcs project(%s) belongs to biz %s, not requested biz %d",
				project.Code,
				project.BizID,
				spec.BizID,
			)
		}
		return project, nil
	}
	if !errors.Is(err, cloudbcs.ErrProjectNotFound) {
		return nil, errors.Wrapf(err, "get bcs project %s before create", spec.ID)
	}
	return createProject(ctx, client, spec)
}

func (m *ProjectManager) newClient(ctx context.Context) (cloudbcs.Client, error) {
	client, err := cloudbcs.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "initial bcs client")
	}
	return client, nil
}

func getExistingProject(
	ctx context.Context, client cloudbcs.Client, projectIDOrCode string,
) (*cloudbcs.Project, error) {
	project, err := client.GetProject(ctx, projectIDOrCode)
	if err != nil {
		return nil, errors.Wrapf(err, "get bcs project %s", projectIDOrCode)
	}
	if project == nil || project.ID == "" {
		return nil, errors.New("get bcs project returned empty id")
	}
	if project.Kind != cloudbcs.ProjectKindK8s {
		return nil, errors.Errorf("bcs project(%s) is not a k8s project", projectIDOrCode)
	}
	return project, nil
}

func createProject(ctx context.Context, client cloudbcs.Client, spec ProjectSpec) (*cloudbcs.Project, error) {
	in := cloudbcs.CreateProjectInput{
		ProjectID:   spec.ID,
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
