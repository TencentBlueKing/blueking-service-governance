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

package dbfactory

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger"
)

// TriggerPolicyOpts 定义创建测试触发策略时的可选参数
type TriggerPolicyOpts struct {
	ID               string
	AppID            string `validate:"required"`
	Name             string
	Event            trigger.Event
	BranchMatchMode  trigger.BranchMatchMode
	BranchMatchValue string
	PathFilter       string
	Status           trigger.Status
	PipelineID       string
	TriggerID        string
	Creator          string
}

// TriggerPolicy 创建一个已持久化的测试用触发策略
func TriggerPolicy(
	ctx context.Context,
	store trigger.PolicyStore,
	opts *TriggerPolicyOpts,
) *trigger.Policy {
	if opts == nil {
		opts = &TriggerPolicyOpts{}
	}
	validateOpts(opts)

	policy := &trigger.Policy{
		ID:               opts.ID,
		AppID:            opts.AppID,
		Name:             opts.Name,
		Event:            opts.Event,
		BranchMatchMode:  opts.BranchMatchMode,
		BranchMatchValue: opts.BranchMatchValue,
		PathFilter:       opts.PathFilter,
		Status:           opts.Status,
		PipelineID:       opts.PipelineID,
		TriggerID:        opts.TriggerID,
		Creator:          opts.Creator,
	}
	if policy.ID == "" {
		policy.ID = trigger.PolicyIDPrefix + stringx.Random(8)
	}
	if policy.Name == "" {
		policy.Name = "policy-" + stringx.Random(6)
	}
	if policy.Event == "" {
		policy.Event = trigger.EventPush
	}
	if policy.BranchMatchMode == "" {
		policy.BranchMatchMode = trigger.BranchMatchModeEq
		if policy.BranchMatchValue == "" {
			policy.BranchMatchValue = "master"
		}
	}
	if policy.Status == "" {
		policy.Status = trigger.StatusEnabled
	}
	if policy.Creator == "" {
		policy.Creator = "dbfactory"
	}

	err := store.Create(ctx, policy)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return policy
}
