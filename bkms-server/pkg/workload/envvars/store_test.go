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

package envvars_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("ScopedEnvVarStoreMongo", func() {
	var diApp *fxtest.App
	var store envvars.ScopedEnvVarStore
	// rawColl reads documents straight from MongoDB, bypassing the store's
	// encrypt/decrypt layer, so tests can assert on the persisted ciphertext.
	var rawColl *mongo.Collection
	var ctx context.Context
	var workspaceID string
	var otherWorkspaceID string

	BeforeEach(func() {
		ctx = context.Background()

		var mongoClient *mongo.Client
		var dbName string
		diApp = fxtest.New(
			GinkgoT(),
			envvars.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			database.PrivateFxModule,
			fx.Populate(&store, &mongoClient, &dbName),
		)
		diApp.RequireStart()
		rawColl = mongoClient.Database(dbName).Collection(envvars.ScopedEnvVarCollectionName)

		workspaceID = "test-workspace-" + stringx.Random(6)
		otherWorkspaceID = "test-workspace-" + stringx.Random(6)
	})

	AfterEach(func() {
		Expect(store.DeleteAll(ctx)).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	Context("workspace scope", func() {
		var testEnvVar envvars.ScopedEnvVar

		BeforeEach(func() {
			testEnvVar = envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "TEST_KEY",
				Value:       "test-value",
				Description: "test description",
				IsSensitive: true,
			}
		})

		It("should create a scoped env var successfully", func() {
			oid, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())
			Expect(oid).NotTo(Equal(bson.NilObjectID))

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeWorkspace))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(1))
			Expect(envVars[0].ID).NotTo(Equal(bson.NilObjectID))
			Expect(envVars[0].WorkspaceID).To(Equal(testEnvVar.WorkspaceID))
			Expect(envVars[0].ScopeType).To(Equal(testEnvVar.ScopeType))
			Expect(envVars[0].ScopeValue).To(Equal(testEnvVar.ScopeValue))
			Expect(envVars[0].Key).To(Equal(testEnvVar.Key))
			Expect(envVars[0].Value).To(Equal(testEnvVar.Value))
			Expect(envVars[0].Description).To(Equal(testEnvVar.Description))
			Expect(envVars[0].IsSensitive).To(Equal(testEnvVar.IsSensitive))
			Expect(envVars[0].CreatedAt.IsZero()).To(BeFalse())
			Expect(envVars[0].UpdatedAt.IsZero()).To(BeFalse())
			Expect(envVars[0].UpdatedAt).To(Equal(envVars[0].CreatedAt))
		})

		It("should get a scoped env var by ID successfully", func() {
			oid, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())

			envVar, err := store.GetByID(ctx, workspaceID, oid)
			Expect(err).NotTo(HaveOccurred())
			Expect(envVar.ID).To(Equal(oid))
			Expect(envVar.WorkspaceID).To(Equal(testEnvVar.WorkspaceID))
			Expect(envVar.ScopeType).To(Equal(testEnvVar.ScopeType))
			Expect(envVar.ScopeValue).To(Equal(testEnvVar.ScopeValue))
			Expect(envVar.Key).To(Equal(testEnvVar.Key))
			Expect(envVar.Value).To(Equal(testEnvVar.Value))
			Expect(envVar.Description).To(Equal(testEnvVar.Description))
			Expect(envVar.IsSensitive).To(Equal(testEnvVar.IsSensitive))
		})

		It("should return error when creating a duplicate scoped env var in the same scope", func() {
			_, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, testEnvVar)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("an env var with the same key already exists in this scope"))
		})

		It("should list only env vars in the target workspace scope and order by key", func() {
			_, err := store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "Z_KEY",
				Value:       "value-z",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "A_KEY",
				Value:       "value-a",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "production",
				Key:         "B_KEY",
				Value:       "value-b",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: otherWorkspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "C_KEY",
				Value:       "value-c",
			})
			Expect(err).NotTo(HaveOccurred())

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeWorkspace))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(2))
			Expect(envVars[0].Key).To(Equal("A_KEY"))
			Expect(envVars[1].Key).To(Equal("Z_KEY"))
		})

		It("should list env vars ordered by created time when requested", func() {
			_, err := store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "Z_KEY",
				Value:       "value-z",
			})
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(time.Millisecond)

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "A_KEY",
				Value:       "value-a",
			})
			Expect(err).NotTo(HaveOccurred())

			envVars, err := store.List(
				ctx,
				workspaceID,
				envvars.WithScopes(envvartypes.ScopeWorkspace),
				envvars.WithOrdering("created"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(2))
			Expect(envVars[0].Key).To(Equal("Z_KEY"))
			Expect(envVars[1].Key).To(Equal("A_KEY"))
		})

		It("should list env vars from multiple scopes ordered by key", func() {
			_, err := store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "production",
				Key:         "B_ENV_TYPE_KEY",
				Value:       "env-type-a",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "C_WORKSPACE_KEY",
				Value:       "workspace-z",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "A_WORKSPACE_KEY",
				Value:       "workspace-a",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  "prod-env",
				Key:         "D_ENV_KEY",
				Value:       "env-a",
			})
			Expect(err).NotTo(HaveOccurred())

			envVars, err := store.List(
				ctx,
				workspaceID,
				envvars.WithScopes(
					envvartypes.ScopeWorkspace,
					envvartypes.ScopeEnvType("production"),
					envvartypes.ScopeEnv("prod-env"),
				),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(envVars, func(item envvars.ScopedEnvVar, _ int) string {
				return string(item.ScopeType) + ":" + item.ScopeValue + ":" + item.Key
			})).To(Equal([]string{
				"workspace::A_WORKSPACE_KEY",
				"envType:production:B_ENV_TYPE_KEY",
				"workspace::C_WORKSPACE_KEY",
				"env:prod-env:D_ENV_KEY",
			}))
		})

		It("should list only built-in env vars in workspace scope and order by key", func() {
			_, err := store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "Z_BUILTIN_KEY",
				Value:       "z-builtin-value",
				IsBuiltin:   true,
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "A_BUILTIN_KEY",
				Value:       "a-builtin-value",
				IsBuiltin:   true,
				IsSensitive: true,
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "USER_DEFINED_KEY",
				Value:       "user-defined-value",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: otherWorkspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "OTHER_WORKSPACE_BUILTIN_KEY",
				Value:       "other-workspace-builtin-value",
				IsBuiltin:   true,
			})
			Expect(err).NotTo(HaveOccurred())

			builtinVars, err := store.List(
				ctx,
				workspaceID,
				envvars.WithScopes(envvartypes.ScopeWorkspace),
				envvars.WithOnlyBuiltin(),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(builtinVars).To(HaveLen(2))
			Expect(builtinVars[0].Key).To(Equal("A_BUILTIN_KEY"))
			Expect(builtinVars[0].Value).To(Equal("a-builtin-value"))
			Expect(builtinVars[0].IsBuiltin).To(BeTrue())
			Expect(builtinVars[0].IsSensitive).To(BeTrue())
			Expect(builtinVars[1].Key).To(Equal("Z_BUILTIN_KEY"))
			Expect(builtinVars[1].Value).To(Equal("z-builtin-value"))
		})

		It("should delete the scoped env var in the target workspace scope only", func() {
			_, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: otherWorkspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         testEnvVar.Key,
				Value:       "other-value",
			})
			Expect(err).NotTo(HaveOccurred())

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeWorkspace))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(1))

			err = store.DeleteByID(ctx, workspaceID, envVars[0].ID)
			Expect(err).NotTo(HaveOccurred())

			envVars, err = store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeWorkspace))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(BeEmpty())

			otherEnvVars, err := store.List(ctx, otherWorkspaceID, envvars.WithScopes(envvartypes.ScopeWorkspace))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherEnvVars).To(HaveLen(1))
			Expect(otherEnvVars[0].Key).To(Equal(testEnvVar.Key))
		})

		It("should update key value description and isSensitive by ID", func() {
			oid, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())

			originalEnvVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeWorkspace))
			Expect(err).NotTo(HaveOccurred())
			Expect(originalEnvVars).To(HaveLen(1))
			originalUpdatedAt := originalEnvVars[0].UpdatedAt

			time.Sleep(time.Millisecond)

			err = store.UpdateByID(ctx, workspaceID, oid, envvars.ScopedEnvVarUpdateData{
				Key:         "UPDATED_KEY",
				Value:       lo.ToPtr("updated-value"),
				Description: "updated description",
				IsSensitive: lo.ToPtr(false),
			})
			Expect(err).NotTo(HaveOccurred())

			updatedEnvVar, err := store.GetByID(ctx, workspaceID, oid)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedEnvVar.ID).To(Equal(oid))
			Expect(updatedEnvVar.Key).To(Equal("UPDATED_KEY"))
			Expect(updatedEnvVar.Value).To(Equal("updated-value"))
			Expect(updatedEnvVar.Description).To(Equal("updated description"))
			Expect(updatedEnvVar.IsSensitive).To(BeFalse())
			Expect(updatedEnvVar.UpdatedAt.After(originalUpdatedAt)).To(BeTrue())
		})

		It("should keep isSensitive unchanged when update isSensitive is nil", func() {
			oid, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())

			err = store.UpdateByID(ctx, workspaceID, oid, envvars.ScopedEnvVarUpdateData{
				Key:         "UPDATED_KEY",
				IsSensitive: nil,
			})
			Expect(err).NotTo(HaveOccurred())

			updatedEnvVar, err := store.GetByID(ctx, workspaceID, oid)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedEnvVar.Key).To(Equal("UPDATED_KEY"))
			Expect(updatedEnvVar.Value).To(Equal(testEnvVar.Value))
			Expect(updatedEnvVar.IsSensitive).To(Equal(testEnvVar.IsSensitive))
		})

		It("should encrypt the updated value at rest when Value is provided", func() {
			oid, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())

			err = store.UpdateByID(ctx, workspaceID, oid, envvars.ScopedEnvVarUpdateData{
				Key:   testEnvVar.Key,
				Value: lo.ToPtr("updated-value"),
			})
			Expect(err).NotTo(HaveOccurred())

			// Read the raw document, bypassing the store's decrypt layer, to verify
			// the value is stored encrypted rather than as plaintext.
			var raw struct {
				Value string `bson:"value"`
			}
			Expect(rawColl.FindOne(ctx, bson.M{"_id": oid}).Decode(&raw)).NotTo(HaveOccurred())
			Expect(raw.Value).NotTo(Equal("updated-value"))
			decryptedValue, err := crypto.AESDecrypt(config.G.Encrypt.Secret, raw.Value)
			Expect(err).NotTo(HaveOccurred())
			Expect(decryptedValue).To(Equal("updated-value"))

			// The store decrypts it back to plaintext on read.
			got, err := store.GetByID(ctx, workspaceID, oid)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Value).To(Equal("updated-value"))
		})

		It("should update value to empty string by ID", func() {
			oid, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())

			err = store.UpdateByID(ctx, workspaceID, oid, envvars.ScopedEnvVarUpdateData{
				Key:   testEnvVar.Key,
				Value: lo.ToPtr(""),
			})
			Expect(err).NotTo(HaveOccurred())

			// Read the raw document to distinguish an encrypted empty value from a
			// missing or plaintext empty value in MongoDB.
			var raw struct {
				Value string `bson:"value"`
			}
			Expect(rawColl.FindOne(ctx, bson.M{"_id": oid}).Decode(&raw)).NotTo(HaveOccurred())
			Expect(raw.Value).NotTo(BeEmpty())
			decryptedValue, err := crypto.AESDecrypt(config.G.Encrypt.Secret, raw.Value)
			Expect(err).NotTo(HaveOccurred())
			Expect(decryptedValue).To(BeEmpty())

			updatedEnvVar, err := store.GetByID(ctx, workspaceID, oid)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedEnvVar.Value).To(BeEmpty())
		})
	})

	Context("public scoped env vars", func() {
		It("should sort public scoped env vars by scope type and env type order", func() {
			vars := []envvars.ScopedEnvVar{
				{ScopeType: envvartypes.ScopeTypeEnvType, ScopeValue: "production", Key: "B_KEY"},
				{ScopeType: envvartypes.ScopeTypeWorkspace, ScopeValue: "", Key: "Z_KEY"},
				{ScopeType: envvartypes.ScopeTypeEnvType, ScopeValue: "staging", Key: "A_KEY"},
				{ScopeType: envvartypes.ScopeTypeEnvType, ScopeValue: "test", Key: "B_KEY"},
				{ScopeType: envvartypes.ScopeTypeWorkspace, ScopeValue: "", Key: "A_KEY"},
				{ScopeType: envvartypes.ScopeTypeEnvType, ScopeValue: "development", Key: "C_KEY"},
				{ScopeType: envvartypes.ScopeTypeEnvType, ScopeValue: "development", Key: "A_KEY"},
			}

			envvars.SortPublicScopedEnvVars(vars)

			got := lo.Map(vars, func(item envvars.ScopedEnvVar, _ int) string {
				return string(item.ScopeType) + ":" + item.ScopeValue + ":" + item.Key
			})
			Expect(got).To(Equal([]string{
				"workspace::A_KEY",
				"workspace::Z_KEY",
				"envType:development:A_KEY",
				"envType:development:C_KEY",
				"envType:test:B_KEY",
				"envType:staging:A_KEY",
				"envType:production:B_KEY",
			}))
		})

		It("should list workspace and env type scoped env vars in public order only", func() {
			for _, item := range []envvars.ScopedEnvVar{
				{
					WorkspaceID: workspaceID,
					ScopeType:   envvartypes.ScopeTypeEnvType,
					ScopeValue:  "production",
					Key:         "B_KEY",
					Value:       "env-type-production-value",
				},
				{
					WorkspaceID: workspaceID,
					ScopeType:   envvartypes.ScopeTypeWorkspace,
					ScopeValue:  "",
					Key:         "Z_KEY",
					Value:       "workspace-z-value",
				},
				{
					WorkspaceID: workspaceID,
					ScopeType:   envvartypes.ScopeTypeEnvType,
					ScopeValue:  "development",
					Key:         "C_KEY",
					Value:       "env-type-development-c-value",
				},
				{
					WorkspaceID: workspaceID,
					ScopeType:   envvartypes.ScopeTypeWorkspace,
					ScopeValue:  "",
					Key:         "A_KEY",
					Value:       "workspace-a-value",
				},
				{
					WorkspaceID: workspaceID,
					ScopeType:   envvartypes.ScopeTypeEnvType,
					ScopeValue:  "development",
					Key:         "A_KEY",
					Value:       "env-type-development-a-value",
				},
				{
					WorkspaceID: workspaceID,
					ScopeType:   envvartypes.ScopeTypeEnv,
					ScopeValue:  "dev",
					Key:         "ENV_SCOPED_KEY",
					Value:       "env-scoped-value",
				},
				{
					WorkspaceID: otherWorkspaceID,
					ScopeType:   envvartypes.ScopeTypeWorkspace,
					ScopeValue:  "",
					Key:         "OTHER_WORKSPACE_KEY",
					Value:       "other-workspace-value",
				},
				{
					WorkspaceID: workspaceID,
					ScopeType:   envvartypes.ScopeTypeWorkspace,
					ScopeValue:  "",
					Key:         "BUILTIN_KEY",
					Value:       "builtin-value",
					IsBuiltin:   true,
				},
			} {
				_, err := store.Create(ctx, item)
				Expect(err).NotTo(HaveOccurred())
			}

			publicVars, err := store.ListPublic(ctx, workspaceID)
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(publicVars, func(item envvars.ScopedEnvVar, _ int) string {
				return string(item.ScopeType) + ":" + item.ScopeValue + ":" + item.Key
			})).To(Equal([]string{
				"workspace::A_KEY",
				"workspace::Z_KEY",
				"envType:development:A_KEY",
				"envType:development:C_KEY",
				"envType:production:B_KEY",
			}))
			Expect(publicVars[0].Value).To(Equal("workspace-a-value"))
		})
	})

	Context("env type scope", func() {
		var testEnvVar envvars.ScopedEnvVar

		BeforeEach(func() {
			testEnvVar = envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "production",
				Key:         "TEST_KEY",
				Value:       "test-value",
				Description: "test description",
				IsSensitive: false,
			}
		})

		It("should create a scoped env var successfully", func() {
			oid, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())
			Expect(oid).NotTo(Equal(bson.NilObjectID))

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnvType("production")))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(1))
			Expect(envVars[0].ID).NotTo(Equal(bson.NilObjectID))
			Expect(envVars[0].WorkspaceID).To(Equal(testEnvVar.WorkspaceID))
			Expect(envVars[0].ScopeType).To(Equal(testEnvVar.ScopeType))
			Expect(envVars[0].ScopeValue).To(Equal(testEnvVar.ScopeValue))
			Expect(envVars[0].Key).To(Equal(testEnvVar.Key))
			Expect(envVars[0].Value).To(Equal(testEnvVar.Value))
			Expect(envVars[0].Description).To(Equal(testEnvVar.Description))
			Expect(envVars[0].IsSensitive).To(Equal(testEnvVar.IsSensitive))
			Expect(envVars[0].CreatedAt.IsZero()).To(BeFalse())
			Expect(envVars[0].UpdatedAt.IsZero()).To(BeFalse())
			Expect(envVars[0].UpdatedAt).To(Equal(envVars[0].CreatedAt))
		})

		It("should return empty list for non-existent scope", func() {
			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnvType("development")))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(BeEmpty())
		})

		It("should list only env vars in the target env type scope and order by key", func() {
			_, err := store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "production",
				Key:         "Z_KEY",
				Value:       "value-z",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "production",
				Key:         "A_KEY",
				Value:       "value-a",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "test",
				Key:         "B_KEY",
				Value:       "value-b",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: otherWorkspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "production",
				Key:         "C_KEY",
				Value:       "value-c",
			})
			Expect(err).NotTo(HaveOccurred())

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnvType("production")))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(2))
			Expect(envVars[0].Key).To(Equal("A_KEY"))
			Expect(envVars[1].Key).To(Equal("Z_KEY"))
		})

		It("should delete the scoped env var in the target env type scope only", func() {
			_, err := store.Create(ctx, testEnvVar)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "test",
				Key:         testEnvVar.Key,
				Value:       "other-scope-value",
			})
			Expect(err).NotTo(HaveOccurred())

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnvType("production")))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(1))

			err = store.DeleteByID(ctx, workspaceID, envVars[0].ID)
			Expect(err).NotTo(HaveOccurred())

			envVars, err = store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnvType("production")))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(BeEmpty())

			otherEnvVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnvType("test")))
			Expect(err).NotTo(HaveOccurred())
			Expect(otherEnvVars).To(HaveLen(1))
			Expect(otherEnvVars[0].Key).To(Equal(testEnvVar.Key))
		})

		It("should allow the same key to exist in different scopes", func() {
			_, err := store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "SHARED_KEY",
				Value:       "workspace-value",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnvType,
				ScopeValue:  "production",
				Key:         "SHARED_KEY",
				Value:       "env-type-value",
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("env scope helpers", func() {
		It("should create a simple env scoped var", func() {
			environment := envmodel.Environment{WorkspaceID: workspaceID, Name: "prod-env"}

			oid, err := store.CreateSimpleEnvScopeVar(ctx, environment, "SIMPLE_KEY", "simple-value", "simple desc")
			Expect(err).NotTo(HaveOccurred())
			Expect(oid).NotTo(Equal(bson.NilObjectID))

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnv("prod-env")))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(1))
			Expect(envVars[0].Key).To(Equal("SIMPLE_KEY"))
			Expect(envVars[0].Value).To(Equal("simple-value"))
			Expect(envVars[0].Description).To(Equal("simple desc"))
			Expect(envVars[0].ScopeType).To(Equal(envvartypes.ScopeTypeEnv))
			Expect(envVars[0].ScopeValue).To(Equal("prod-env"))
			Expect(envVars[0].IsBuiltin).To(BeFalse())
			Expect(envVars[0].IsSensitive).To(BeFalse())
		})

		It("should delete vars in the target env only", func() {
			environment := envmodel.Environment{WorkspaceID: workspaceID, Name: "prod-env"}

			_, err := store.CreateSimpleEnvScopeVar(ctx, environment, "ENV_KEY", "env-value", "env desc")
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "WORKSPACE_KEY",
				Value:       "workspace-value",
			})
			Expect(err).NotTo(HaveOccurred())

			err = store.DeleteByEnv(ctx, environment)
			Expect(err).NotTo(HaveOccurred())

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnv("prod-env")))
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(BeEmpty())

			workspaceVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeWorkspace))
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaceVars).To(HaveLen(1))
			Expect(workspaceVars[0].Key).To(Equal("WORKSPACE_KEY"))
		})

		It("should batch upsert vars by key and only update value and description", func() {
			_, err := store.Create(ctx, envvars.ScopedEnvVar{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  "prod-env",
				Key:         "EXISTING_KEY",
				Value:       "old-value",
				Description: "old desc",
				IsBuiltin:   false,
				IsSensitive: true,
			})
			Expect(err).NotTo(HaveOccurred())

			err = store.BatchUpsertByKey(ctx, workspaceID, envvartypes.ScopeEnv("prod-env"), []envvars.ScopedEnvVar{
				{
					Key:         "EXISTING_KEY",
					Value:       "new-value",
					Description: "new desc",
					IsBuiltin:   true,
					IsSensitive: false,
				},
				{
					Key:         "NEW_KEY",
					Value:       "created-value",
					Description: "created desc",
					IsBuiltin:   true,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			envVars, err := store.List(ctx, workspaceID, envvars.WithScopes(envvartypes.ScopeEnv("prod-env")))
			Expect(err).NotTo(HaveOccurred())
			varsByKey := lo.SliceToMap(envVars, func(item envvars.ScopedEnvVar) (string, envvars.ScopedEnvVar) {
				return item.Key, item
			})

			existing := varsByKey["EXISTING_KEY"]
			Expect(existing.Value).To(Equal("new-value"))
			Expect(existing.Description).To(Equal("new desc"))
			Expect(existing.IsBuiltin).To(BeFalse())
			Expect(existing.IsSensitive).To(BeTrue())

			builtinVars, err := store.List(
				ctx,
				workspaceID,
				envvars.WithScopes(envvartypes.ScopeEnv("prod-env")),
				envvars.WithOnlyBuiltin(),
			)
			Expect(err).NotTo(HaveOccurred())
			builtinVarsByKey := lo.SliceToMap(
				builtinVars,
				func(item envvars.ScopedEnvVar) (string, envvars.ScopedEnvVar) { return item.Key, item },
			)

			created := builtinVarsByKey["NEW_KEY"]
			Expect(created.Value).To(Equal("created-value"))
			Expect(created.Description).To(Equal("created desc"))
			Expect(created.IsBuiltin).To(BeTrue())
			Expect(created.IsSensitive).To(BeFalse())
		})
	})
})
