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

package model_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
)

var _ = Describe("EnvBindingStore", func() {
	var store *model.EnvBindingStore
	var ctx context.Context
	var testAppID string

	BeforeEach(func() {
		store = testutil.NewStore(GinkgoT(), model.EnvBindingMeta)
		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		_, _ = store.DeleteMany(ctx, model.ByApp(testAppID))
	})

	// newBinding 构造一个满足校验的 EnvBinding。
	newBinding := func(envName string) *model.EnvBinding {
		return &model.EnvBinding{
			AppID:       testAppID,
			EnvName:     envName,
			BscpEnvID:   "12",
			BscpEnvName: envName,
			BscpAppID:   "541",
			Operator:    "tester",
		}
	}

	Describe("Create", func() {
		Context("when creating a valid binding", func() {
			It("should create successfully", func() {
				err := store.Create(ctx, newBinding("dev"))
				Expect(err).NotTo(HaveOccurred())

				stored, err := store.Get(ctx, model.ByAppAndEnv(testAppID, "dev"))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.BscpEnvID).To(Equal("12"))
				Expect(stored.BscpEnvName).To(Equal("dev"))
				Expect(stored.BscpAppID).To(Equal("541"))
				Expect(stored.Operator).To(Equal("tester"))
			})
		})

		Context("when creating a duplicate app+env binding", func() {
			It("should return ErrEnvBindingAlreadyExists", func() {
				err := store.Create(ctx, newBinding("dev"))
				Expect(err).NotTo(HaveOccurred())

				err = store.Create(ctx, newBinding("dev"))
				Expect(err).To(MatchError(model.ErrEnvBindingAlreadyExists))
			})
		})

		Context("when required fields are missing", func() {
			It("should return validation error for missing appID", func() {
				binding := newBinding("dev")
				binding.AppID = ""

				err := store.Create(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})

			It("should return validation error for missing envName", func() {
				binding := newBinding("")
				err := store.Create(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})
		})
	})

	Describe("Get", func() {
		Context("when binding does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				_, err := store.Get(ctx, model.ByAppAndEnv(testAppID, "staging"))
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})
})
