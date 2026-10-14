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

// Package handler contains Gin handlers for deploy diagnosis APIs.
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
)

var _ deploydiagnosis.Handler = (*Handler)(nil)

// Handler handles Gin deploy diagnosis API requests.
type Handler struct {
	registry *storereg.Registry
}

// New 创建部署诊断 Gin Handler
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// GetDeployDiagnosis 获取指定环境的部署诊断结论
//
//	@ID			GetDeployDiagnosis
//	@Summary	获取部署诊断结论
//	@Tags		deploy-diagnosis
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200				{object}	serializer.GetDeployDiagnosisOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/deploy-diagnosis [get]
func (h *Handler) GetDeployDiagnosis(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.GetDeployDiagnosisQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	// 权限与实例列表一致，无查看权限时 403 且不进入聚合
	app, env, err := ginperm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	data, err := deploydiagnosis.NewDiagnoser(h.registry).Diagnose(ctx, app, env, queryInput.TrafficLaneName)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ginutils.OK(c, serializer.GetDeployDiagnosisOutput{Data: data})
}
