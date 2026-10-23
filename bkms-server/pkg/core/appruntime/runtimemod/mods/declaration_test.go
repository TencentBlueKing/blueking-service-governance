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

package mods_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
)

var _ = Describe("Declarations", func() {
	It("keeps every Set field in the public projection", func() {
		// 用反射取出 Set 中每个能力的 ID，与 Public 输出逐一比对：
		// 新增能力若漏加到 all()，对外输出会缺少该能力
		set := mustResolve(appruntime.FrameworkTRPC, appruntime.LanguageGo)
		value := reflect.ValueOf(set)
		ids := make([]runtimemod.ModID, 0, value.NumField())
		for i := range value.NumField() {
			idField := value.Field(i).FieldByName("ID")
			if !idField.IsValid() {
				continue
			}
			ids = append(ids, idField.Interface().(runtimemod.ModID))
		}

		Expect(ids).To(HaveLen(8))
		Expect(lo.Uniq(ids)).To(HaveLen(len(ids)))
		Expect(lo.Keys(set.Public().Mods)).To(ConsistOf(ids))
	})
})
