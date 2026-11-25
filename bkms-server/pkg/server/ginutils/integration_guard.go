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

package ginutils

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/kubeinsight"
)

const bscpDisabledMsg = i18n.Msg("bscp is disabled in current environment")

// GlobalErrorRules 在任何接口中含义都相同的错误规则，注册到 bkerrs.ErrorHandler 后 handler 无需逐个指定：
//   - bscp.ErrDisabled：未挂 RequireBSCP、但会间接调用 BSCP 的接口（如配置文件校验）
//   - bscp.ErrNoPermission：用户态调用 BSCP 被拒绝
var GlobalErrorRules = []bkerrs.Rule{
	bkerrs.OnFunc(bscp.ErrDisabled, WrapBSCPDisabled),
	bkerrs.OnFunc(bscp.ErrNoPermission, WrapBSCPNoPermission),
}

// RequireBSCP 当前环境关闭了 BSCP 对接时，拦截 BSCP 相关接口并返回明确的未启用错误
func RequireBSCP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if bscp.IsDisabled() {
			bkerrs.AbortWithErr(c, WrapBSCPDisabled(bscp.ErrDisabled))
			return
		}
		c.Next()
	}
}

// RequireKubeInsight 当前环境关闭了 KubeInsight（集群诊断）对接时，拦截集群诊断相关接口并返回明确的未启用错误
func RequireKubeInsight() gin.HandlerFunc {
	return func(c *gin.Context) {
		if kubeinsight.IsDisabled() {
			bkerrs.AbortWithErr(c, bkerrs.NewKubeInsightDisabled())
			return
		}
		c.Next()
	}
}

// WrapBSCPDisabled 包装为当前环境关闭了 BSCP 对接的错误
func WrapBSCPDisabled(err error) *bkerrs.Error {
	return bkerrs.Wrap(err, bkerrs.ErrCodeNotImplemented, bscpDisabledMsg).SetDetails(
		bkerrs.NewDetail(
			bkerrs.ErrDetailCodeBSCPDisabled,
			bscpDisabledMsg,
			bkerrs.WithSystem(bkerrs.SystemName),
			bkerrs.WithModule(bscp.ModuleName),
		),
	)
}

// WrapBSCPNoPermission 包装为无权限访问 BSCP 资源的错误
func WrapBSCPNoPermission(err error) *bkerrs.Error {
	return bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, i18n.Msg("no permission to access bscp resources")).SetDetails(
		bkerrs.NewDetail(
			bkerrs.ErrDetailCodeBSCPNoPermission,
			"bscp service permission denied",
			bkerrs.WithSystem(bkerrs.SystemName),
			bkerrs.WithModule(bscp.ModuleName),
		),
	)
}
