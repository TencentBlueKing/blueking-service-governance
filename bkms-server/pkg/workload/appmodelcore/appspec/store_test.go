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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("AppSpec store", func() {
	var ctx context.Context
	var db *mongo.Database
	var specs *AppSpecStore
	var appID, envName string
	var spec *AppSpec

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		specs = store.Of(db, AppSpecMeta)

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
		_, _ = specs.DeleteMany(ctx, ByApp(appID))
	})

	upsert := func(spec *AppSpec) error {
		_, err := specs.Upsert(ctx, spec, ByAppEnv(spec.AppID, spec.EnvName))
		return err
	}

	It("returns ErrAppSpecNotFound when the app spec does not exist", func() {
		_, err := specs.Get(ctx, ByAppEnv(appID, envName))
		Expect(err).To(MatchError(ErrAppSpecNotFound))
	})

	It("lists env names without the default entry", func() {
		err := upsert(&AppSpec{
			AppID:   appID,
			EnvName: DefaultEnvName,
			Resources: &ResourcesSpec{
				Replicas: lo.ToPtr(int32(1)),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		err = upsert(spec)
		Expect(err).NotTo(HaveOccurred())

		envNames, err := ListEnvNamesByApp(ctx, db, appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames).To(Equal([]string{envName}))
	})

	It("ignores env docs with no configured sections when listing env names", func() {
		err := SetSections(ctx, db, AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				Replicas: lo.ToPtr(int32(2)),
			},
		}, AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())

		err = SetSections(ctx, db, AppSpec{
			AppID:   appID,
			EnvName: envName,
		}, AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())

		envNames, err := ListEnvNamesByApp(ctx, db, appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames).To(BeEmpty())
	})

	It("patches existing spec with partial fields", func() {
		err := specs.Create(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		err = Patch(ctx, db, AppSpec{
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

		saved, err := specs.Get(ctx, ByAppEnv(appID, envName))
		Expect(err).NotTo(HaveOccurred())
		Expect(*saved.Resources.CPURequests).To(Equal("100m"))
		Expect(*saved.Resources.MemoryRequests).To(Equal("256Mi"))
		Expect(*saved.Resources.MemoryLimits).To(Equal("512Mi"))
		Expect(*saved.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*saved.UpdateStrategy.MaxSurge).To(Equal("3"))
		Expect(*saved.DevMode.Enabled).To(BeFalse())
	})

	It("patches non-existent spec by creating one", func() {
		err := Patch(ctx, db, AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				Replicas: lo.ToPtr(int32(3)),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		saved, err := specs.Get(ctx, ByAppEnv(appID, envName))
		Expect(err).NotTo(HaveOccurred())
		Expect(*saved.Resources.Replicas).To(Equal(int32(3)))
		Expect(saved.UpdateStrategy).To(BeNil())
		Expect(saved.DevMode).To(BeNil())
	})

	It("sets only the specified sections with whole-section semantics", func() {
		err := specs.Create(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		err = SetSections(ctx, db, AppSpec{
			AppID:   appID,
			EnvName: envName,
			Resources: &ResourcesSpec{
				MemoryRequests: lo.ToPtr("256Mi"),
			},
		}, AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())

		saved, err := specs.Get(ctx, ByAppEnv(appID, envName))
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.Resources.Replicas).To(BeNil())
		Expect(saved.Resources.CPURequests).To(BeNil())
		Expect(*saved.Resources.MemoryRequests).To(Equal("256Mi"))
		Expect(*saved.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(*saved.DevMode.Enabled).To(BeTrue())
	})

	It("unsets a specified section without touching others", func() {
		err := specs.Create(ctx, spec)
		Expect(err).NotTo(HaveOccurred())

		err = SetSections(ctx, db, AppSpec{
			AppID:   appID,
			EnvName: envName,
		}, AppSpecSectionUpdateStrategy)
		Expect(err).NotTo(HaveOccurred())

		saved, err := specs.Get(ctx, ByAppEnv(appID, envName))
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.UpdateStrategy).To(BeNil())
		Expect(*saved.Resources.Replicas).To(Equal(int32(2)))
		Expect(*saved.DevMode.Enabled).To(BeTrue())
	})

	It("rejects creating a duplicate spec of the same app and env", func() {
		Expect(specs.Create(ctx, spec)).To(Succeed())

		err := specs.Create(ctx, spec)
		Expect(err).To(MatchError(ErrAppSpecAlreadyExists))
	})

	It("collapses empty sections when saving", func() {
		Expect(specs.Create(ctx, &AppSpec{
			AppID:   appID,
			EnvName: envName,
			Labels:  &LabelsSpec{Labels: map[string]string{}},
			DevMode: &DevModeSpec{},
		})).To(Succeed())

		// 空 section 不应落库
		raw, err := specs.Collection().FindOne(ctx, bson.M{"appID": appID, "envName": envName}).Raw()
		Expect(err).NotTo(HaveOccurred())
		_, err = raw.LookupErr("labels")
		Expect(err).To(HaveOccurred())
		_, err = raw.LookupErr("devMode")
		Expect(err).To(HaveOccurred())

		saved, err := specs.Get(ctx, ByAppEnv(appID, envName))
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.HasConfiguredSections()).To(BeFalse())
	})

	It("does not create a spec when only unsetting sections", func() {
		err := SetSections(ctx, db, AppSpec{
			AppID:   appID,
			EnvName: envName,
		}, AppSpecSectionResources, AppSpecSectionDevMode)
		Expect(err).NotTo(HaveOccurred())

		_, err = specs.Get(ctx, ByAppEnv(appID, envName))
		Expect(err).To(MatchError(ErrAppSpecNotFound))
	})

	It("keeps appID and envName from the filter when patch creates a spec", func() {
		Expect(Patch(ctx, db, AppSpec{
			AppID:   appID,
			EnvName: envName,
			Labels:  &LabelsSpec{Labels: map[string]string{"team": "a"}},
		})).To(Succeed())

		saved, err := specs.Get(ctx, ByAppEnv(appID, envName))
		Expect(err).NotTo(HaveOccurred())
		Expect(saved.AppID).To(Equal(appID))
		Expect(saved.EnvName).To(Equal(envName))
		Expect(saved.Labels.Labels).To(Equal(map[string]string{"team": "a"}))
	})

	It("returns error for unknown section IDs", func() {
		err := SetSections(ctx, db, AppSpec{
			AppID:   appID,
			EnvName: envName,
		}, AppSpecSectionID("unknown"))
		Expect(err).To(MatchError(`unknown app spec section "unknown"`))
	})
})
