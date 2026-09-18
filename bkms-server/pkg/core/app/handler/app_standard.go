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

package handler

import (
	"context"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	standardapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/standard"
)

func (h *Handler) newStandardService() *standardapp.Service {
	return standardapp.NewService(
		h.registry.AppModelStore,
		h.registry.AppSpecStore,
		h.registry.AppDefaultRuleStore,
		h.registry.EnvStore,
		h.registry.AppStore,
	)
}

// createStandardApp 创建 standard 应用（handler 层，负责参数转换）
func (h *Handler) createStandardApp(
	ctx context.Context,
	app *bkmsapp.Application,
	input *serializer.AppModelSpecInput,
) error {
	// 参数转换：serializer 输入 -> 内部类型
	params := input.ToStandardCreateParams()

	return h.newStandardService().Create(ctx, app, params)
}
