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

package appmodel_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("AppEnvVarService", func() {
	var (
		diApp                     *fxtest.App
		ctx                       context.Context
		appStore                  bkmsapp.ApplicationStore
		appModelStore             appmodel.AppModelStore
		appConfigFileStore        appcfg.AppConfigFileStore
		appConfigFileVersionStore appcfg.AppConfigFileVersionStore
		buildConfigStore          imagebuild.ConfigStore
		service                   *appmodel.AppEnvVarService
	)

	BeforeEach(func() {
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appmodel.FxModule,
			appcfg.FxModule,
			imagebuild.FxModule,
			fx.Populate(
				&appStore,
				&appModelStore,
				&appConfigFileStore,
				&appConfigFileVersionStore,
				&buildConfigStore,
				&service,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	newTrpcApp := func(envVars []appmodel.Variable) *bkmsapp.Application {
		app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigFileVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			EnvVars: envVars,
		})
		return app
	}
	withoutTimestamps := func(envVar appmodel.Variable) appmodel.Variable {
		envVar.CreatedAt = time.Time{}
		envVar.UpdatedAt = time.Time{}
		return envVar
	}
	withoutTimestampsList := func(envVars []appmodel.Variable) []appmodel.Variable {
		result := make([]appmodel.Variable, len(envVars))
		for i, envVar := range envVars {
			result[i] = withoutTimestamps(envVar)
		}
		return result
	}

	It("lists app-defined env vars", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
			{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		})

		envVars, err := service.List(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(withoutTimestampsList(envVars)).To(ConsistOf(
			appmodel.Variable{Key: "MODE", Value: "prod", Description: "runtime mode"},
			appmodel.Variable{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		))
	})

	It("creates a new app-defined env var atomically", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
		})

		created, err := service.Create(ctx, app.ID, appmodel.Variable{
			Key:         "REGION",
			Value:       "ap-guangzhou",
			Description: "deploy region",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(created).NotTo(BeNil())
		Expect(created.CreatedAt.IsZero()).To(BeFalse())
		Expect(created.UpdatedAt).To(Equal(created.CreatedAt))
		Expect(withoutTimestamps(*created)).To(Equal(appmodel.Variable{
			Key:         "REGION",
			Value:       "ap-guangzhou",
			Description: "deploy region",
		}))

		envVars, err := service.List(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(withoutTimestampsList(envVars)).To(ConsistOf(
			appmodel.Variable{Key: "MODE", Value: "prod", Description: "runtime mode"},
			appmodel.Variable{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		))
	})

	It("rejects duplicated env var keys on create", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
		})

		_, err := service.Create(ctx, app.ID, appmodel.Variable{
			Key:         "MODE",
			Value:       "test",
			Description: "duplicate key",
		})
		Expect(err).To(MatchError(appmodel.ErrEnvVarKeyExists))
	})

	It("rejects invalid env var keys on create", func() {
		app := newTrpcApp(nil)

		_, err := service.Create(ctx, app.ID, appmodel.Variable{
			Key:         "INVALID-KEY",
			Value:       "test",
			Description: "invalid key",
		})
		Expect(err).To(MatchError(appmodel.ErrInvalidEnvVarKey))
	})

	It("updates an existing env var without replacing the whole list", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
			{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		})

		oldEnvVar, updatedEnvVar, err := service.Update(ctx, app.ID, "MODE", appmodel.AppEnvVarUpdateData{
			Key:         "MODE",
			Value:       lo.ToPtr("test"),
			Description: "updated mode",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(oldEnvVar).NotTo(BeNil())
		Expect(withoutTimestamps(*oldEnvVar)).To(Equal(appmodel.Variable{
			Key:         "MODE",
			Value:       "prod",
			Description: "runtime mode",
		}))
		Expect(updatedEnvVar).NotTo(BeNil())
		Expect(updatedEnvVar.CreatedAt).To(Equal(oldEnvVar.CreatedAt))
		Expect(updatedEnvVar.UpdatedAt).To(BeTemporally(">", oldEnvVar.UpdatedAt))
		Expect(withoutTimestamps(*updatedEnvVar)).To(Equal(appmodel.Variable{
			Key:         "MODE",
			Value:       "test",
			Description: "updated mode",
		}))

		envVars, err := service.List(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(withoutTimestampsList(envVars)).To(ConsistOf(
			appmodel.Variable{Key: "MODE", Value: "test", Description: "updated mode"},
			appmodel.Variable{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		))
		Expect(envVars).To(ContainElement(appmodel.Variable{
			Key:         "MODE",
			Value:       "test",
			Description: "updated mode",
			CreatedAt:   oldEnvVar.CreatedAt,
			// MongoDB stores time.Time with millisecond precision and decodes it as UTC.
			UpdatedAt: updatedEnvVar.UpdatedAt.UTC().Truncate(time.Millisecond),
		}))
	})

	It("updates a non-sensitive env var value to empty", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
		})

		_, updatedEnvVar, err := service.Update(ctx, app.ID, "MODE", appmodel.AppEnvVarUpdateData{
			Key:         "MODE",
			Value:       lo.ToPtr(""),
			Description: "runtime mode",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedEnvVar.Value).To(BeEmpty())

		envVars, err := service.List(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(withoutTimestampsList(envVars)).To(ConsistOf(
			appmodel.Variable{Key: "MODE", Value: "", Description: "runtime mode"},
		))
	})

	It("renames an env var key atomically", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
			{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		})

		oldEnvVar, updatedEnvVar, err := service.Update(ctx, app.ID, "MODE", appmodel.AppEnvVarUpdateData{
			Key:         "APP_MODE",
			Value:       lo.ToPtr("prod"),
			Description: "runtime mode",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(oldEnvVar).NotTo(BeNil())
		Expect(withoutTimestamps(*oldEnvVar)).To(Equal(appmodel.Variable{
			Key:         "MODE",
			Value:       "prod",
			Description: "runtime mode",
		}))
		Expect(updatedEnvVar).NotTo(BeNil())
		Expect(updatedEnvVar.CreatedAt).To(Equal(oldEnvVar.CreatedAt))
		Expect(updatedEnvVar.UpdatedAt).To(BeTemporally(">", oldEnvVar.UpdatedAt))
		Expect(withoutTimestamps(*updatedEnvVar)).To(Equal(appmodel.Variable{
			Key:         "APP_MODE",
			Value:       "prod",
			Description: "runtime mode",
		}))

		envVars, err := service.List(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(withoutTimestampsList(envVars)).To(ConsistOf(
			appmodel.Variable{Key: "APP_MODE", Value: "prod", Description: "runtime mode"},
			appmodel.Variable{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		))
	})

	It("rejects renaming to an existing key", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
			{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		})

		_, _, err := service.Update(ctx, app.ID, "MODE", appmodel.AppEnvVarUpdateData{
			Key:         "REGION",
			Value:       lo.ToPtr("prod"),
			Description: "runtime mode",
		})
		Expect(err).To(MatchError(appmodel.ErrEnvVarKeyExists))
	})

	It("rejects invalid env var keys on update", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
		})

		_, _, err := service.Update(ctx, app.ID, "MODE", appmodel.AppEnvVarUpdateData{
			Key:         "INVALID+KEY",
			Value:       lo.ToPtr("prod"),
			Description: "runtime mode",
		})
		Expect(err).To(MatchError(appmodel.ErrInvalidEnvVarKey))
	})

	It("deletes an existing env var atomically", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
			{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		})

		deleted, err := service.Delete(ctx, app.ID, "MODE")
		Expect(err).NotTo(HaveOccurred())
		Expect(deleted).NotTo(BeNil())
		Expect(withoutTimestamps(*deleted)).To(Equal(appmodel.Variable{
			Key:         "MODE",
			Value:       "prod",
			Description: "runtime mode",
		}))

		envVars, err := service.List(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(withoutTimestampsList(envVars)).To(ConsistOf(
			appmodel.Variable{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
		))
	})

	Context("with sensitive env vars", func() {
		It("creates a sensitive env var and lists it with the real value", func() {
			app := newTrpcApp(nil)

			created, err := service.Create(ctx, app.ID, appmodel.Variable{
				Key:         "SECRET",
				Value:       "supersecret",
				IsSensitive: true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.CreatedAt.IsZero()).To(BeFalse())
			Expect(created.UpdatedAt).To(Equal(created.CreatedAt))
			Expect(withoutTimestamps(*created)).To(Equal(appmodel.Variable{
				Key:         "SECRET",
				Value:       "supersecret",
				IsSensitive: true,
			}))

			// List returns the real value; masking happens only at the serializer layer.
			envVars, err := service.List(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(withoutTimestampsList(envVars)).To(ConsistOf(
				appmodel.Variable{Key: "SECRET", Value: "supersecret", IsSensitive: true},
			))
		})

		It("keeps a sensitive env var value when value is omitted", func() {
			app := newTrpcApp([]appmodel.Variable{
				{Key: "SECRET", Value: "supersecret", IsSensitive: true},
			})

			_, updatedEnvVar, err := service.Update(ctx, app.ID, "SECRET", appmodel.AppEnvVarUpdateData{
				Key:         "SECRET",
				IsSensitive: lo.ToPtr(true),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedEnvVar.Value).To(Equal("supersecret"))

			envVars, err := service.List(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(withoutTimestampsList(envVars)).To(ConsistOf(
				appmodel.Variable{Key: "SECRET", Value: "supersecret", IsSensitive: true},
			))
		})

		It("keeps sensitivity when isSensitive is omitted", func() {
			app := newTrpcApp([]appmodel.Variable{
				{Key: "SECRET", Value: "supersecret", IsSensitive: true},
			})

			_, updatedEnvVar, err := service.Update(ctx, app.ID, "SECRET", appmodel.AppEnvVarUpdateData{
				Key:   "SECRET",
				Value: lo.ToPtr("new-secret"),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedEnvVar.IsSensitive).To(BeTrue())

			envVars, err := service.List(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(withoutTimestampsList(envVars)).To(ConsistOf(
				appmodel.Variable{Key: "SECRET", Value: "new-secret", IsSensitive: true},
			))
		})

		It("updates a sensitive env var value to empty", func() {
			app := newTrpcApp([]appmodel.Variable{
				{Key: "SECRET", Value: "supersecret", IsSensitive: true},
			})

			oldEnvVar, updatedEnvVar, err := service.Update(ctx, app.ID, "SECRET", appmodel.AppEnvVarUpdateData{
				Key:         "SECRET",
				Value:       lo.ToPtr(""),
				IsSensitive: lo.ToPtr(true),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(oldEnvVar).NotTo(BeNil())
			Expect(withoutTimestamps(*oldEnvVar)).To(Equal(appmodel.Variable{
				Key:         "SECRET",
				Value:       "supersecret",
				IsSensitive: true,
			}))
			Expect(updatedEnvVar).NotTo(BeNil())
			Expect(updatedEnvVar.Value).To(BeEmpty())
			Expect(updatedEnvVar.CreatedAt).To(Equal(oldEnvVar.CreatedAt))
			Expect(updatedEnvVar.UpdatedAt).To(BeTemporally(">", oldEnvVar.UpdatedAt))

			envVars, err := service.List(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(withoutTimestampsList(envVars)).To(ConsistOf(
				appmodel.Variable{Key: "SECRET", Value: "", IsSensitive: true},
			))
		})

		It("updates a sensitive var with a new real value", func() {
			app := newTrpcApp([]appmodel.Variable{
				{Key: "SECRET", Value: "old-secret", IsSensitive: true},
			})

			_, updatedEnvVar, err := service.Update(ctx, app.ID, "SECRET", appmodel.AppEnvVarUpdateData{
				Key:         "SECRET",
				Value:       lo.ToPtr("new-secret"),
				IsSensitive: lo.ToPtr(true),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedEnvVar.Value).To(Equal("new-secret"))

			envVars, err := service.List(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(withoutTimestampsList(envVars)).To(ConsistOf(
				appmodel.Variable{Key: "SECRET", Value: "new-secret", IsSensitive: true},
			))
		})

		It("rejects changing a sensitive var to non-sensitive", func() {
			app := newTrpcApp([]appmodel.Variable{
				{Key: "SECRET", Value: "supersecret", IsSensitive: true},
			})

			_, _, err := service.Update(ctx, app.ID, "SECRET", appmodel.AppEnvVarUpdateData{
				Key:         "SECRET",
				Value:       lo.ToPtr("supersecret"),
				IsSensitive: lo.ToPtr(false),
			})
			Expect(err).To(MatchError(appmodel.ErrEnvVarSensitivityImmutable))

			// Verify the env var is unchanged.
			envVars, err := service.List(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(withoutTimestampsList(envVars)).To(ConsistOf(
				appmodel.Variable{Key: "SECRET", Value: "supersecret", IsSensitive: true},
			))
		})

		It("honors the sensitive flag sent by the frontend", func() {
			app := newTrpcApp([]appmodel.Variable{
				{Key: "SECRET", Value: "supersecret", IsSensitive: true},
				{Key: "MODE", Value: "prod", IsSensitive: false},
			})

			// Frontend re-sends the current flag (true) for the sensitive var.
			_, updatedSecret, err := service.Update(ctx, app.ID, "SECRET", appmodel.AppEnvVarUpdateData{
				Key:         "SECRET",
				Value:       lo.ToPtr("new-secret"),
				IsSensitive: lo.ToPtr(true),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedSecret.IsSensitive).To(BeTrue())

			// Frontend re-sends the current flag (false) for the non-sensitive var.
			_, updatedMode, err := service.Update(ctx, app.ID, "MODE", appmodel.AppEnvVarUpdateData{
				Key:         "MODE",
				Value:       lo.ToPtr("test"),
				IsSensitive: lo.ToPtr(false),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedMode.IsSensitive).To(BeFalse())
		})
	})

	It("batch upserts app-defined env vars atomically", func() {
		app := newTrpcApp([]appmodel.Variable{
			{Key: "MODE", Value: "prod", Description: "runtime mode"},
			{Key: "SECRET", Value: "supersecret", IsSensitive: true},
		})

		err := service.BatchUpsert(ctx, app.ID, []appmodel.Variable{
			{Key: "MODE", Value: "test", Description: "updated mode"},
			{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
			{Key: "SECRET", Value: "new-secret", Description: "sensitive secret", IsSensitive: true},
		})
		Expect(err).NotTo(HaveOccurred())

		envVars, err := service.List(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(withoutTimestampsList(envVars)).To(ConsistOf(
			appmodel.Variable{Key: "MODE", Value: "test", Description: "updated mode"},
			appmodel.Variable{Key: "REGION", Value: "ap-guangzhou", Description: "deploy region"},
			appmodel.Variable{Key: "SECRET", Value: "new-secret", Description: "sensitive secret", IsSensitive: true},
		))
	})
})
