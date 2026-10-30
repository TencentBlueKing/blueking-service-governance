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

package releasestrategy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
)

func permanent(target string) releasestrategy.Batch {
	return releasestrategy.Batch{Target: target, PauseMode: releasestrategy.PauseModePermanent}
}

func pause30s(target string) releasestrategy.Batch {
	return releasestrategy.Batch{Target: target, PauseMode: releasestrategy.PauseModeDuration, PauseSeconds: 30}
}

func noPause(target string) releasestrategy.Batch {
	return releasestrategy.Batch{Target: target, PauseMode: releasestrategy.PauseModeNone}
}

var _ = Describe("Builtin release strategies", func() {
	It("provides three strategies ordered by batch count", func() {
		builtins := releasestrategy.BuiltinStrategies()
		Expect(builtins).To(HaveLen(3))

		expected := []struct {
			id      string
			name    string
			batches []releasestrategy.Batch
		}{
			{
				id:      "000000000000000000000003",
				name:    "分三批灰度(1个,50%,全量)",
				batches: []releasestrategy.Batch{permanent("1"), pause30s("50%"), noPause("100%")},
			},
			{
				id:      "000000000000000000000004",
				name:    "分四批灰度(1个,33%,66%,全量)",
				batches: []releasestrategy.Batch{permanent("1"), pause30s("33%"), pause30s("66%"), noPause("100%")},
			},
			{
				id:   "000000000000000000000005",
				name: "分五批灰度(1个,25%,50%,75%,全量)",
				batches: []releasestrategy.Batch{
					permanent("1"), pause30s("25%"), pause30s("50%"), pause30s("75%"), noPause("100%"),
				},
			},
		}
		for i, want := range expected {
			Expect(builtins[i].ID.Hex()).To(Equal(want.id))
			Expect(builtins[i].Name).To(Equal(want.name))
			Expect(builtins[i].Batches).To(Equal(want.batches))
			Expect(builtins[i].WorkspaceID).To(BeEmpty())
			Expect(builtins[i].IsBuiltin()).To(BeTrue())
			Expect(releasestrategy.IsBuiltinName(want.name)).To(BeTrue())
		}
	})

	It("uses IDs that never collide with generated ObjectIDs", func() {
		for _, s := range releasestrategy.BuiltinStrategies() {
			Expect(s.ID.Timestamp().Unix()).To(BeZero())
		}
		Expect(releasestrategy.ReleaseStrategy{ID: bson.NewObjectID()}.IsBuiltin()).To(BeFalse())
		Expect(releasestrategy.ReleaseStrategy{}.IsBuiltin()).To(BeFalse())
		Expect(releasestrategy.IsBuiltinName("生产谨慎发布")).To(BeFalse())
	})

	It("returns copies that callers cannot use to modify the builtins", func() {
		builtins := releasestrategy.BuiltinStrategies()
		builtins[0].Name = "changed"
		builtins[0].Batches[0].Target = "99%"

		again := releasestrategy.BuiltinStrategies()
		Expect(again[0].Name).To(Equal("分三批灰度(1个,50%,全量)"))
		Expect(again[0].Batches[0].Target).To(Equal("1"))
	})
})
