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

package serializer

import (
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

func init() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	if err := v.RegisterValidation("batch_target", validateBatchTarget); err != nil {
		panic("failed to register batch_target validator: " + err.Error())
	}
	if err := v.RegisterValidation("release_strategy_name", validateReleaseStrategyName); err != nil {
		panic("failed to register release_strategy_name validator: " + err.Error())
	}
}

// releaseStrategyNameMaxLen 是策略名称去掉首尾空白后的最大字符数。
const releaseStrategyNameMaxLen = 64

// validateReleaseStrategyName 校验策略名称：去掉首尾空白后为 1 ~ 64 个字符。
// 长度按字符计算；入库值由 ToModel 按同一口径 trim。
func validateReleaseStrategyName(fl validator.FieldLevel) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(fl.Field().String()))
	return n >= 1 && n <= releaseStrategyNameMaxLen
}

// validateBatchTarget 校验批次目标值：正整数（≥ 1）或百分比（1% ~ 100%）
func validateBatchTarget(fl validator.FieldLevel) bool {
	_, err := releasestrategy.ParseBatchTarget(fl.Field().String())
	return err == nil
}
