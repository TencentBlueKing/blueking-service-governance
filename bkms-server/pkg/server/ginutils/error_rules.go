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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bscpapierrs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp/apierrs"
)

// GlobalErrorRules 在任何接口中含义都相同的错误规则，注册到 bkerrs.ErrorHandler 后 handler 无需逐个指定：
//   - bscp.ErrDisabled：未挂 RequireBSCP、但会间接调用 BSCP 的接口（如配置文件校验）
//
// 全局规则优先于 handler 的映射，新增规则需同时满足：
//   - 含义与具体接口无关，不是某个操作的业务约束
//   - 错误码与文案对任何调用场景都成立，反例见 bscpapierrs.NoPermission
var GlobalErrorRules = []bkerrs.Rule{
	bscpapierrs.Disabled,
}
