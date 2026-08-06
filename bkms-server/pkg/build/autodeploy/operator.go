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

package autodeploy

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Operator 封装 build auto deploy 记录的 db 操作
type Operator struct {
	store RecordStore
}

// Locator 描述需要更新的记录定位条件
type Locator struct {
	AppID    string
	BuildID  string
	DeployID string
}

// StatusPatch 描述状态字段更新内容
type StatusPatch struct {
	Stage    Stage
	Status   string
	Message  string
	DeployID *string
	EndedAt  *time.Time
}

// NewOperator 创建 Operator
func NewOperator(store RecordStore) (*Operator, error) {
	if store == nil {
		return nil, errors.New("build auto deploy record store is nil")
	}
	return &Operator{store: store}, nil
}

// GetByBuildID 根据 buildID 获取记录
func (u *Operator) GetByBuildID(ctx context.Context, appID, buildID string) (*Record, error) {
	return u.store.GetByBuildID(ctx, appID, buildID)
}

// UpdateStatus 更新 build auto deploy 记录状态
func (u *Operator) UpdateStatus(ctx context.Context, locator Locator, patch StatusPatch) error {
	record, err := u.getRecord(ctx, locator)
	if err != nil {
		return err
	}
	record.Stage = patch.Stage
	record.Status = patch.Status
	record.Message = patch.Message
	if patch.DeployID != nil {
		record.DeployID = *patch.DeployID
	}
	if patch.EndedAt != nil {
		record.EndedAt = *patch.EndedAt
	}
	return u.store.Update(ctx, record)
}

func (u *Operator) getRecord(ctx context.Context, locator Locator) (*Record, error) {
	if locator.AppID == "" {
		return nil, errors.New("appID is required")
	}
	hasBuildID := locator.BuildID != ""
	hasDeployID := locator.DeployID != ""
	if hasBuildID == hasDeployID {
		return nil, errors.New("exactly one of buildID or deployID is required")
	}
	if hasBuildID {
		return u.store.GetByBuildID(ctx, locator.AppID, locator.BuildID)
	}
	return u.store.GetByDeployID(ctx, locator.AppID, locator.DeployID)
}
