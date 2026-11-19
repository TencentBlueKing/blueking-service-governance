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

package server_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server"
)

// server 包引入了全部业务模块，此处 bkerrs.Definitions() 即为服务中定义的全部错误
var _ = Describe("Error messages", func() {
	It("should have a localized message for every defined error", func() {
		definitions := bkerrs.Definitions()
		Expect(definitions).NotTo(BeEmpty())
		for _, def := range definitions {
			Expect(
				i18n.HasKey(string(def.DetailCode())),
			).To(BeTrue(), "missing message for detail code %s", def.DetailCode())
		}
	})
})
