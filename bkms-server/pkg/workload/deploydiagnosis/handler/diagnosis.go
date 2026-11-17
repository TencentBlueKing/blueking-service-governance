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
//	@ID				GetDeployDiagnosis
//	@Summary		获取部署诊断结论
//	@Description	诊断该环境（泳道）最新一次部署为什么没有正常运行，给出出错原因、处理步骤和跳转按钮，供部署页面板与 CLI 展示。
//	@Description	1) 诊断对象固定为最新一次部署，AppModel 应用会把一键构建部署记录也纳入比较，构建阶段失败同样能被诊断。
//	@Description	2) 采集最新部署的实例、异常实例的日志（重启前日志优先）、本次部署开始后的 Pod 与工作负载事件，以及框架配置。
//	@Description	3) 有异常实例时按实例规则逐个匹配并按规则分组；没有异常实例时用阶段规则判断部署 / 构建是否已失败。
//	@Description	4) 部署进行中不做诊断，返回 emptyReason=deploying；日志或事件未取全时仍给结论，并通过 partialReasons 标明。
//	@Description	5) 只有列实例失败会使整个请求失败。设计说明见 design_notes/deploy_diagnosis.md。
//	@Tags			deploy-diagnosis
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

	diagnosis, err := deploydiagnosis.NewDiagnoser(h.registry).Diagnose(ctx, app, env, queryInput.TrafficLaneName)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ginutils.OK(c, serializer.GetDeployDiagnosisOutput{
		Data: new(serializer.DeployDiagnosisOutputObj).FromDiagnosis(diagnosis),
	})
}
