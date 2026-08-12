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
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// WatchAppInstances 订阅应用实例投影变更（契约骨架）
//
// 推送的是平台投影事件（对齐 AppInstanceOutputObj，含 polarisInfos），不是原生 Pod JSON。
// 传输形态（SSE / WebSocket）待定；本接口当前未实现推送，仅用于 OpenAPI 契约声明。
// 鉴权与 List 同级（须具备目标应用在目标环境的查看权限），由后续实现卡落地。
//
//	@ID			WatchAppInstances
//	@Summary	订阅应用实例投影变更
//	@Description	事件类型为 ADDED / MODIFIED / DELETED；object 对齐 AppInstanceOutputObj（含 polarisInfos）。
//	传输形态 SSE/WS 待定；当前返回未实现。
//
//	@Tags		instance
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200				{object}	serializer.AppInstanceWatchEvent	"Watch 事件逻辑结构（编码格式随传输形态而定）"
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Failure	501				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/watch [get]
func (h *Handler) WatchAppInstances(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.WatchAppInstancesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotImplemented, "watch app instances is not implemented yet"))
}
