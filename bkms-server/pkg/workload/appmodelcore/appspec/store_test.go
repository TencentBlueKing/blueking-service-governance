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

package appspec

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("AppSpecStore", func() {
	var ctx context.Context
	var store AppSpecStore
	var appID, envName string
	var spec *AppSpec

	BeforeEach(func() {
		var err error

		ctx = context.Background()
		store, err = NewAppSpecStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		appID = "test-app-" + stringx.Random(6)
		envName = "test-env-" + stringx.Random(6)
		spec = &AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				Replicas:       lo.ToPtr(int32(2)),
				CPURequests:    lo.ToPtr("100m"),
				CPULimits:      lo.ToPtr("200m"),
				MemoryRequests: lo.ToPtr("128Mi"),
			},
			UpdateStrategy: &UpdateStrategySpec{
				MaxUnavailable: lo.ToPtr("25%"),
				MaxSurge:       lo.ToPtr("1"),
			},
			DevMode: &DevModeSpec{
				Enabled: lo.ToPtr(true),
			},
		}
	})

	AfterEach(func() {
		_ = store.Delete(ctx, appID, DefaultEnvName)
		_ = store.Delete(ctx, appID, envName)
	})

	It("creates and gets app spec", func() {
		err := store.Create(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		saved, err := store.Get(ctx, appID, envName)
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.AppID).To(Equal(appID))
		Expect(saved.EnvName).To(Equal(envName))
		Expect(*saved.Resources.Replicas).To(Equal(int32(2)))
		Expect(*saved.Resources.CPURequests).To(Equal("100m"))
		Expect(*saved.Resources.CPULimits).To(Equal("200m"))
		Expect(*saved.Resources.MemoryRequests).To(Equal("128Mi"))
		Expect(*saved.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(saved.DevMode).NotTo(BeNil())
		Expect(saved.DevMode.WorkPath).To(BeNil())
	})

	It("upserts and deletes app spec", func() {
		err := store.Create(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		err = store.Upsert(ctx, &AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				Replicas:       lo.ToPtr(int32(5)),
				MemoryRequests: lo.ToPtr("256Mi"),
				MemoryLimits:   lo.ToPtr("512Mi"),
			},
			DevMode: &DevModeSpec{
				Enabled: lo.ToPtr(false),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		saved, err := store.Get(ctx, appID, envName)
		Expect(err).NotTo(HaveOccurred())
		Expect(*saved.Resources.Replicas).To(Equal(int32(5)))
		Expect(saved.Resources.CPURequests).To(BeNil())
		Expect(*saved.Resources.MemoryRequests).To(Equal("256Mi"))
		Expect(*saved.Resources.MemoryLimits).To(Equal("512Mi"))
		Expect(saved.UpdateStrategy).To(BeNil())
		Expect(*saved.DevMode.Enabled).To(BeFalse())

		err = store.Delete(ctx, appID, envName)
		Expect(err).NotTo(HaveOccurred())

		_, err = store.Get(ctx, appID, envName)
		Expect(errors.Is(err, ErrAppSpecNotFound)).To(BeTrue())
	})

	It("lists env names without the default entry", func() {
		err := store.Upsert(ctx, &AppSpec{
			AppID:   appID,
			EnvName: DefaultEnvName,
			Resources: &ResourcesSpec{
				Replicas: lo.ToPtr(int32(1)),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		err = store.Upsert(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		envNames, err := store.ListEnvNamesByApp(ctx, appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames).To(Equal([]string{envName}))
	})

	It("ignores env docs with no configured sections when listing env names", func() {
		err := store.SetSections(ctx, AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				Replicas: lo.ToPtr(int32(2)),
			},
		}, AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())

		err = store.SetSections(ctx, AppSpec{
			AppID:   appID,
			EnvName: envName,
		}, AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())

		envNames, err := store.ListEnvNamesByApp(ctx, appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames).To(BeEmpty())
	})

	It("patches existing spec with partial fields", func() {
		err := store.Create(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		err = store.Patch(ctx, AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				MemoryRequests: lo.ToPtr("256Mi"),
				MemoryLimits:   lo.ToPtr("512Mi"),
			},
			UpdateStrategy: &UpdateStrategySpec{
				MaxSurge: lo.ToPtr("3"),
			},
			DevMode: &DevModeSpec{
				Enabled: lo.ToPtr(false),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		saved, err := store.Get(ctx, appID, envName)
		Expect(err).NotTo(HaveOccurred())
		Expect(*saved.Resources.CPURequests).To(Equal("100m"))
		Expect(*saved.Resources.MemoryRequests).To(Equal("256Mi"))
		Expect(*saved.Resources.MemoryLimits).To(Equal("512Mi"))
		Expect(*saved.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*saved.UpdateStrategy.MaxSurge).To(Equal("3"))
		Expect(*saved.DevMode.Enabled).To(BeFalse())
	})

	It("patches non-existent spec by creating one", func() {
		err := store.Patch(ctx, AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				Replicas: lo.ToPtr(int32(3)),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		saved, err := store.Get(ctx, appID, envName)
		Expect(err).NotTo(HaveOccurred())
		Expect(*saved.Resources.Replicas).To(Equal(int32(3)))
		Expect(saved.UpdateStrategy).To(BeNil())
		Expect(saved.DevMode).To(BeNil())
	})

	It("sets only the specified sections with whole-section semantics", func() {
		err := store.Create(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		err = store.SetSections(ctx, AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				MemoryRequests: lo.ToPtr("256Mi"),
			},
		}, AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())

		saved, err := store.Get(ctx, appID, envName)
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.Resources.Replicas).To(BeNil())
		Expect(saved.Resources.CPURequests).To(BeNil())
		Expect(*saved.Resources.MemoryRequests).To(Equal("256Mi"))
		Expect(*saved.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*saved.DevMode.Enabled).To(BeTrue())
	})

	It("unsets a specified section without touching others", func() {
		err := store.Create(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		err = store.SetSections(ctx, AppSpec{
			AppID:   appID,
			EnvName: envName,
		}, AppSpecSectionUpdateStrategy)
		Expect(err).NotTo(HaveOccurred())

		saved, err := store.Get(ctx, appID, envName)
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.UpdateStrategy).To(BeNil())
		Expect(*saved.Resources.Replicas).To(Equal(int32(2)))
		Expect(*saved.DevMode.Enabled).To(BeTrue())
	})

	It("returns error for unknown section IDs", func() {
		err := store.SetSections(ctx, AppSpec{
			AppID:   appID,
			EnvName: envName,
		}, AppSpecSectionID("unknown"))
		Expect(err).To(MatchError(`unknown app spec section "unknown"`))
	})
})
