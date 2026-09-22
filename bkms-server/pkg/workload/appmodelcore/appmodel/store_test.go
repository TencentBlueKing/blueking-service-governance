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
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("AppModel store", func() {
	var db *mongo.Database
	var appStore *bkmsapp.ApplicationStore
	var models *appmodel.AppModelStore
	// rawColl reads documents straight from MongoDB, bypassing the store's
	// encrypt/decrypt layer, so tests can assert on the persisted ciphertext.
	var rawColl *mongo.Collection

	var ctx context.Context

	BeforeEach(func() {
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		models = store.Of(db, appmodel.AppModelMeta)
		rawColl = db.Collection(appmodel.CollectionName)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(appmodel.AppModelMeta)).To(Succeed())
	})

	Describe("CreateAppModel", func() {
		Context("when creating a valid app model", func() {
			It("round-trips old TrpcConfig and new frameworkConfig including integers", func() {
				app := dbfactory.Application(ctx, appStore)
				appModel := &appmodel.AppModel{
					AppID: app.ID,
					Workload: appmodel.Workload{
						Type: appmodel.WorkloadTypeTrpc,
						TrpcConfig: appmodel.TrpcConfig{
							FileName:    "trpc_go.yaml",
							FilePath:    "/etc/",
							FileContent: "server:\n  port: 8080\n",
							Language:    appmodel.LanguageGo,
						},
						FrameworkConfig: map[string]any{
							"fileName": "trpc_go.yaml",
							"filePath": "/etc/",
							"retries":  3,
							"nested":   map[string]any{"ok": true},
							"tags":     []any{"a"},
						},
						FrameworkConfigVersion: 1,
					},
				}
				err := appmodel.CreateAppModel(ctx, db, appModel)
				Expect(err).NotTo(HaveOccurred())

				got, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(got.Workload.TrpcConfig.FileName).To(Equal("trpc_go.yaml"))
				Expect(got.Workload.TrpcConfig.FileContent).To(Equal("server:\n  port: 8080\n"))
				Expect(got.Workload.TrpcConfig.Language).To(Equal(appmodel.LanguageGo))
				Expect(got.Workload.FrameworkConfigVersion).To(Equal(1))
				Expect(got.Workload.FrameworkConfig["fileName"]).To(Equal("trpc_go.yaml"))

				// The driver must not widen stored integers into float64 on read back.
				Expect(got.Workload.FrameworkConfig["retries"]).NotTo(BeAssignableToTypeOf(float64(0)))
				Expect(fmt.Sprint(got.Workload.FrameworkConfig["retries"])).To(Equal("3"))

				snap := got.Workload.RuntimeSnapshot(appruntime.Snapshot{
					AppID:            app.ID,
					AppType:          appruntime.AppTypeTRPC,
					TrpcSpecLanguage: appruntime.LanguageGo,
				})
				report := appruntime.Inspect(snap)
				Expect(report.Blocking()).To(BeFalse())
				Expect(report.Stack).To(Equal(appruntime.Stack{
					Type:      appruntime.AppTypeBKMSApp,
					Framework: appruntime.FrameworkTRPC,
					Language:  appruntime.LanguageGo,
				}))
			})

			It("encrypts sensitive env vars at rest but returns plaintext on read", func() {
				app := dbfactory.Application(ctx, appStore)

				err := appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{
					AppID: app.ID,
					Workload: appmodel.Workload{
						EnvVars: []appmodel.Variable{
							{Key: "SECRET", Value: "supersecret", IsSensitive: true},
							{Key: "MODE", Value: "prod", IsSensitive: false},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				// Verify the values are stored encrypted at rest by reading the raw
				// document, bypassing the store's decrypt layer.
				var raw struct {
					Workload struct {
						EnvVars []struct {
							Key   string `bson:"key"`
							Value string `bson:"value"`
						} `bson:"envVars,omitempty"`
					} `bson:"workload"`
				}
				Expect(rawColl.FindOne(ctx, bson.M{"appID": app.ID}).Decode(&raw)).NotTo(HaveOccurred())
				varsByKey := map[string]string{}
				for _, v := range raw.Workload.EnvVars {
					varsByKey[v.Key] = v.Value
				}
				// The sensitive value is encrypted at rest: it is not the plaintext,
				// and it round-trips through AES decrypt back to the plaintext.
				Expect(varsByKey["SECRET"]).NotTo(Equal("supersecret"))
				decryptedSecret, err := crypto.AESDecrypt(svccfg.G.Encrypt.Secret, varsByKey["SECRET"])
				Expect(err).NotTo(HaveOccurred())
				Expect(decryptedSecret).To(Equal("supersecret"))
				// The non-sensitive value is stored as plaintext.
				Expect(varsByKey["MODE"]).To(Equal("prod"))

				// Reads decrypt sensitive values, so callers see plaintext.
				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				for i := range am.Workload.EnvVars {
					Expect(am.Workload.EnvVars[i].CreatedAt.IsZero()).To(BeFalse())
					Expect(am.Workload.EnvVars[i].UpdatedAt).To(Equal(am.Workload.EnvVars[i].CreatedAt))
					am.Workload.EnvVars[i].CreatedAt = time.Time{}
					am.Workload.EnvVars[i].UpdatedAt = time.Time{}
				}
				Expect(am.Workload.EnvVars).To(ConsistOf(
					appmodel.Variable{Key: "SECRET", Value: "supersecret", IsSensitive: true},
					appmodel.Variable{Key: "MODE", Value: "prod", IsSensitive: false},
				))
			})

			It("keeps the caller's sensitive env vars in plaintext", func() {
				app := dbfactory.Application(ctx, appStore)
				appModel := &appmodel.AppModel{
					AppID: app.ID,
					Workload: appmodel.Workload{
						EnvVars: []appmodel.Variable{{Key: "SECRET", Value: "supersecret", IsSensitive: true}},
					},
				}
				Expect(appmodel.CreateAppModel(ctx, db, appModel)).To(Succeed())
				Expect(appModel.Workload.EnvVars[0].Value).To(Equal("supersecret"))
			})
		})

		Context("when the app already has an app model", func() {
			It("should return ErrAppModelAlreadyExists", func() {
				app := dbfactory.Application(ctx, appStore)
				Expect(appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{AppID: app.ID})).To(Succeed())

				err := appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{AppID: app.ID})
				Expect(err).To(MatchError(appmodel.ErrAppModelAlreadyExists))
			})
		})

		Context("when Properties contains nested map types", func() {
			It("should normalize Properties to Go native types after retrieval", func() {
				app := dbfactory.Application(ctx, appStore)
				// Create AppModel with various nested types in Properties
				appModel := &appmodel.AppModel{
					AppID: app.ID,
					Components: []*component.Component{{
						Name: "test",
						ComponentInst: component.ComponentInst{
							Type:    "test-comp",
							Version: "v1.0.0",
							Properties: map[string]any{
								"stringVal": "hello",
								"intVal":    int64(42),
								"nestedMap": map[string]string{
									"key1": "value1",
									"key2": "value2",
								},
								"nestedMapAny": map[string]any{
									"innerKey": "innerValue",
									"innerNum": int64(100),
								},
								"arrayVal": []string{"a", "b", "c"},
							},
						},
					}},
				}

				err := appmodel.CreateAppModel(ctx, db, appModel)
				Expect(err).NotTo(HaveOccurred())

				// Retrieve the AppModel
				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(am.Components).To(HaveLen(1))

				props := am.Components[0].Properties

				// Verify string value
				Expect(props["stringVal"]).To(Equal("hello"))

				// Verify int value (will be converted to float64 by JSON)
				Expect(props["intVal"]).To(Equal(int64(42)))

				// Verify nested map[string]string is now map[string]any with string values
				nestedMap, ok := props["nestedMap"].(map[string]any)
				Expect(ok).To(BeTrue(), "nestedMap should be assertable as map[string]any")
				Expect(nestedMap["key1"]).To(Equal("value1"))
				Expect(nestedMap["key2"]).To(Equal("value2"))

				// Verify nested map[string]any
				nestedMapAny, ok := props["nestedMapAny"].(map[string]any)
				Expect(ok).To(BeTrue(), "nestedMapAny should be assertable as map[string]any")
				Expect(nestedMapAny["innerKey"]).To(Equal("innerValue"))
				Expect(nestedMapAny["innerNum"]).To(Equal(float64(100)))

				// Verify array value
				Expect(props["arrayVal"]).To(ConsistOf("a", "b", "c"))
			})
		})

		Context("when Component has invalid Type/RefWorkspaceCompName combination", func() {
			It("should return error when both Type and RefWorkspaceCompName are empty", func() {
				app := dbfactory.Application(ctx, appStore)
				appModel := &appmodel.AppModel{
					AppID: app.ID,
					Components: []*component.Component{{
						Name: "test-comp",
						// Both Type and RefWorkspaceCompName are empty
					}},
				}
				err := appmodel.CreateAppModel(ctx, db, appModel)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must have only one of"))
			})

			It("should return error when both Type and RefWorkspaceCompName are set", func() {
				app := dbfactory.Application(ctx, appStore)
				appModel := &appmodel.AppModel{
					AppID: app.ID,
					Components: []*component.Component{{
						Name: "test-comp",
						ComponentInst: component.ComponentInst{
							Type:    "test-type",
							Version: "v1.0.0",
						},
						ComponentRef: component.ComponentRef{
							RefWorkspaceCompName: "ref-comp",
						},
					}},
				}
				err := appmodel.CreateAppModel(ctx, db, appModel)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must have only one of"))
			})

			It("should accept Component with only Type set", func() {
				app := dbfactory.Application(ctx, appStore)
				appModel := &appmodel.AppModel{
					AppID: app.ID,
					Components: []*component.Component{{
						Name: "test-comp",
						ComponentInst: component.ComponentInst{
							Type:    "test-type",
							Version: "v1.0.0",
						},
					}},
				}
				err := appmodel.CreateAppModel(ctx, db, appModel)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept Component with only RefWorkspaceCompName set", func() {
				app := dbfactory.Application(ctx, appStore)
				appModel := &appmodel.AppModel{
					AppID: app.ID,
					Components: []*component.Component{{
						Name: "test-comp",
						ComponentRef: component.ComponentRef{
							RefWorkspaceCompName: "ref-comp",
						},
					}},
				}
				err := appmodel.CreateAppModel(ctx, db, appModel)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("EnvVars uniqueness validation", func() {
		It("rejects duplicate env var keys when creating app model", func() {
			app := dbfactory.Application(ctx, appStore)
			appModel := &appmodel.AppModel{
				AppID: app.ID,
				Workload: appmodel.Workload{
					EnvVars: []appmodel.Variable{
						{Key: "DUP_KEY", Value: "v1"},
						{Key: "DUP_KEY", Value: "v2"},
					},
				},
			}

			err := appmodel.CreateAppModel(ctx, db, appModel)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate env var key 'DUP_KEY'"))
		})

		It("rejects duplicate env var keys when updating app model", func() {
			app := dbfactory.Application(ctx, appStore)
			appModel := &appmodel.AppModel{
				AppID: app.ID,
				Workload: appmodel.Workload{
					EnvVars: []appmodel.Variable{
						{Key: "ENV_A", Value: "v1"},
					},
				},
			}
			err := appmodel.CreateAppModel(ctx, db, appModel)
			Expect(err).NotTo(HaveOccurred())

			appModel.Workload.EnvVars = []appmodel.Variable{
				{Key: "DUP_KEY", Value: "v1"},
				{Key: "DUP_KEY", Value: "v2"},
			}

			err = appmodel.UpdateAppModel(ctx, db, appModel)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate env var key 'DUP_KEY'"))
		})
	})

	Describe("UpdateAppModel and DeleteAppModel", func() {
		It("should return ErrAppModelNotFound when the app model does not exist", func() {
			Expect(appmodel.UpdateAppModel(ctx, db, &appmodel.AppModel{AppID: "non-existent-app"})).
				To(MatchError(appmodel.ErrAppModelNotFound))
			Expect(appmodel.DeleteAppModel(ctx, db, "non-existent-app")).
				To(MatchError(appmodel.ErrAppModelNotFound))
		})

		It("should replace the whole app model and generate missing component names", func() {
			app := dbfactory.Application(ctx, appStore)
			Expect(appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{
				AppID:  app.ID,
				Labels: map[string]string{"old": "label"},
			})).To(Succeed())

			Expect(appmodel.UpdateAppModel(ctx, db, &appmodel.AppModel{
				AppID: app.ID,
				Components: []*component.Component{{
					ComponentInst: component.ComponentInst{Type: "VolumeSecret", Version: "v1.0.0"},
				}},
			})).To(Succeed())

			am, err := models.Get(ctx, appmodel.ByApp(app.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(am.Labels).To(BeEmpty())
			Expect(am.Components).To(HaveLen(1))
			Expect(am.Components[0].Name).To(HavePrefix("volumesecret-"))
		})
	})

	Describe("AddComponent", func() {
		var app *bkmsapp.Application

		BeforeEach(func() {
			app = dbfactory.Application(ctx, appStore)
			appModel := &appmodel.AppModel{
				AppID:      app.ID,
				Components: []*component.Component{},
			}
			err := appmodel.CreateAppModel(ctx, db, appModel)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when adding a valid component", func() {
			It("should add component successfully", func() {
				comp := &component.Component{
					Name: "test-comp-1",
					ComponentInst: component.ComponentInst{
						Type:    "VolumeSecret",
						Version: "v1.0.0",
						Properties: map[string]any{
							"key": "value",
						},
					},
				}

				err := appmodel.AddComponent(ctx, db, app.ID, comp)
				Expect(err).NotTo(HaveOccurred())

				// Verify component was added
				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(am.Components).To(HaveLen(1))
				Expect(am.Components[0].Name).To(Equal("test-comp-1"))
				Expect(am.Components[0].Type).To(Equal("VolumeSecret"))
			})

			It("should auto-generate name if not provided", func() {
				comp := &component.Component{
					ComponentInst: component.ComponentInst{
						Type:    "VolumeSecret",
						Version: "v1.0.0",
					},
				}

				err := appmodel.AddComponent(ctx, db, app.ID, comp)
				Expect(err).NotTo(HaveOccurred())

				// Verify component name was auto-generated
				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(am.Components).To(HaveLen(1))
				Expect(am.Components[0].Name).To(HavePrefix(strings.ToLower(comp.Type) + "-"))
			})
		})

		Context("when app model does not exist", func() {
			It("should return ErrAppModelNotFound", func() {
				comp := &component.Component{
					Name: "test-comp",
					ComponentInst: component.ComponentInst{
						Type:    "VolumeSecret",
						Version: "v1.0.0",
					},
				}

				err := appmodel.AddComponent(ctx, db, "non-existent-app", comp)
				Expect(err).To(MatchError(appmodel.ErrAppModelNotFound))
			})
		})

		Context("when component type and workspace reference are both set", func() {
			It("should reject the component before persistence", func() {
				comp := &component.Component{
					Name: "invalid-component",
					ComponentInst: component.ComponentInst{
						Type:    "VolumeSecret",
						Version: "v1.0.0",
					},
					ComponentRef: component.ComponentRef{
						RefWorkspaceCompName: "workspace-component",
					},
				}

				err := appmodel.AddComponent(ctx, db, app.ID, comp)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must have only one of"))

				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(am.Components).To(BeEmpty())
			})
		})

		Context("when component name already exists", func() {
			It("should return ErrComponentNameExists", func() {
				// Add first component
				comp1 := &component.Component{
					Name: "duplicate-name",
					ComponentInst: component.ComponentInst{
						Type:    "VolumeSecret",
						Version: "v1.0.0",
					},
				}
				err := appmodel.AddComponent(ctx, db, app.ID, comp1)
				Expect(err).NotTo(HaveOccurred())

				// Try to add second component with same name
				comp2 := &component.Component{
					Name: "duplicate-name",
					ComponentInst: component.ComponentInst{
						Type:    "NodePortService",
						Version: "v1.0.0",
					},
				}
				err = appmodel.AddComponent(ctx, db, app.ID, comp2)
				Expect(err).To(MatchError(appmodel.ErrComponentNameExists))
			})
		})
	})

	Describe("UpdateComponent", func() {
		var app *bkmsapp.Application
		const compName = "test-comp-update"

		BeforeEach(func() {
			app = dbfactory.Application(ctx, appStore)
			appModel := &appmodel.AppModel{
				AppID: app.ID,
				Components: []*component.Component{{
					Name: compName,
					ComponentInst: component.ComponentInst{
						Type:    "VolumeSecret",
						Version: "v1.0.0",
						Properties: map[string]any{
							"key1": "value1",
						},
					},
				}},
			}
			err := appmodel.CreateAppModel(ctx, db, appModel)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when updating component properties", func() {
			It("should update properties successfully", func() {
				newProps := map[string]any{
					"key1": "updated-value",
					"key2": "new-value",
				}
				updateData := &appmodel.ComponentUpdateData{
					Properties: newProps,
				}

				err := appmodel.UpdateComponent(ctx, db, app.ID, compName, updateData)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(am.Components[0].Properties["key1"]).To(Equal("updated-value"))
				Expect(am.Components[0].Properties["key2"]).To(Equal("new-value"))
			})
		})

		Context("when updating multiple fields", func() {
			It("should update all fields successfully", func() {
				newVersion := "v2.0.0"
				newName := "renamed-comp"
				updateData := &appmodel.ComponentUpdateData{
					Version: &newVersion,
					Name:    &newName,
				}

				err := appmodel.UpdateComponent(ctx, db, app.ID, compName, updateData)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(am.Components[0].Version).To(Equal("v2.0.0"))
				Expect(am.Components[0].Name).To(Equal("renamed-comp"))
			})
		})

		Context("when app model does not exist", func() {
			It("should return ErrAppModelNotFound", func() {
				newVersion := "v2.0.0"
				updateData := &appmodel.ComponentUpdateData{
					Version: &newVersion,
				}

				err := appmodel.UpdateComponent(ctx, db, "non-existent-app", compName, updateData)
				Expect(err).To(MatchError(appmodel.ErrAppModelNotFound))
			})
		})

		Context("when updating name to an existing name", func() {
			It("should return ErrComponentNameExists", func() {
				// Add another component
				comp2 := &component.Component{
					Name: "another-comp",
					ComponentInst: component.ComponentInst{
						Type:    "NodePortService",
						Version: "v1.0.0",
					},
				}
				err := appmodel.AddComponent(ctx, db, app.ID, comp2)
				Expect(err).NotTo(HaveOccurred())

				// Try to rename first component to second component's name
				existingName := "another-comp"
				updateData := &appmodel.ComponentUpdateData{
					Name: &existingName,
				}

				err = appmodel.UpdateComponent(ctx, db, app.ID, compName, updateData)
				Expect(err).To(MatchError(appmodel.ErrComponentNameExists))
			})
		})

		Context("when the component does not exist", func() {
			It("should return ErrComponentNotFound", func() {
				newVersion := "v2.0.0"
				err := appmodel.UpdateComponent(ctx, db, app.ID, "non-existent-comp",
					&appmodel.ComponentUpdateData{Version: &newVersion})
				Expect(err).To(MatchError(appmodel.ErrComponentNotFound))

				newName := "renamed-comp"
				err = appmodel.UpdateComponent(ctx, db, app.ID, "non-existent-comp",
					&appmodel.ComponentUpdateData{Name: &newName})
				Expect(err).To(MatchError(appmodel.ErrComponentNotFound))
			})
		})

		Context("when updateData is nil", func() {
			It("should return nil without error", func() {
				err := appmodel.UpdateComponent(ctx, db, app.ID, compName, nil)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("RemoveComponent", func() {
		var app *bkmsapp.Application
		const compName = "test-comp-remove"

		BeforeEach(func() {
			app = dbfactory.Application(ctx, appStore)
			appModel := &appmodel.AppModel{
				AppID: app.ID,
				Components: []*component.Component{{
					Name: compName,
					ComponentInst: component.ComponentInst{
						Type:    "VolumeSecret",
						Version: "v1.0.0",
					},
				}},
			}
			err := appmodel.CreateAppModel(ctx, db, appModel)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when removing an existing component", func() {
			It("should remove component successfully", func() {
				err := appmodel.RemoveComponent(ctx, db, app.ID, compName)
				Expect(err).NotTo(HaveOccurred())

				// Verify removal
				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(am.Components).To(BeEmpty())
			})
		})

		Context("when app model does not exist", func() {
			It("should return ErrAppModelNotFound", func() {
				err := appmodel.RemoveComponent(ctx, db, "non-existent-app", compName)
				Expect(err).To(MatchError(appmodel.ErrAppModelNotFound))
			})
		})

		Context("when component does not exist", func() {
			It("should return ErrComponentNotFound", func() {
				err := appmodel.RemoveComponent(ctx, db, app.ID, "non-existent-comp")
				Expect(err).To(MatchError(appmodel.ErrComponentNotFound))
			})
		})

		Context("when removing one of multiple components", func() {
			It("should only remove the specified component", func() {
				// Add another component
				comp2 := &component.Component{
					Name: "another-comp",
					ComponentInst: component.ComponentInst{
						Type:    "NodePortService",
						Version: "v1.0.0",
					},
				}
				err := appmodel.AddComponent(ctx, db, app.ID, comp2)
				Expect(err).NotTo(HaveOccurred())

				// Remove the first component
				err = appmodel.RemoveComponent(ctx, db, app.ID, compName)
				Expect(err).NotTo(HaveOccurred())

				// Verify only the second component remains
				am, err := models.Get(ctx, appmodel.ByApp(app.ID))
				Expect(err).NotTo(HaveOccurred())
				Expect(am.Components).To(HaveLen(1))
				Expect(am.Components[0].Name).To(Equal("another-comp"))
			})
		})
	})

	Describe("AddAppDefinedEnvVar", func() {
		It("initializes missing timestamps", func() {
			app := dbfactory.Application(ctx, appStore)
			Expect(appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{AppID: app.ID})).To(Succeed())

			err := appmodel.AddAppDefinedEnvVar(ctx, db, app.ID, appmodel.Variable{
				Key:   "MODE",
				Value: "prod",
			})
			Expect(err).NotTo(HaveOccurred())

			envVars, err := appmodel.ListAppDefinedEnvVars(ctx, db, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(1))
			Expect(envVars[0].CreatedAt.IsZero()).To(BeFalse())
			Expect(envVars[0].UpdatedAt).To(Equal(envVars[0].CreatedAt))
		})
	})

	Describe("UpdateAppDefinedEnvVar", func() {
		It("initializes a missing updated timestamp", func() {
			app := dbfactory.Application(ctx, appStore)

			err := appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{
				AppID: app.ID,
				Workload: appmodel.Workload{
					EnvVars: []appmodel.Variable{{Key: "MODE", Value: "prod"}},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			beforeUpdate, err := appmodel.ListAppDefinedEnvVars(ctx, db, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(beforeUpdate).To(HaveLen(1))

			err = appmodel.UpdateAppDefinedEnvVar(ctx, db, app.ID, "MODE", appmodel.Variable{
				Key:   "MODE",
				Value: "test",
			})
			Expect(err).NotTo(HaveOccurred())

			afterUpdate, err := appmodel.ListAppDefinedEnvVars(ctx, db, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(afterUpdate).To(HaveLen(1))
			Expect(afterUpdate[0].CreatedAt).To(Equal(beforeUpdate[0].CreatedAt))
			Expect(afterUpdate[0].UpdatedAt.IsZero()).To(BeFalse())
		})

		It("encrypts an empty sensitive value at rest", func() {
			app := dbfactory.Application(ctx, appStore)

			err := appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{
				AppID: app.ID,
				Workload: appmodel.Workload{
					EnvVars: []appmodel.Variable{
						{Key: "SECRET", Value: "old-secret", IsSensitive: true},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = appmodel.UpdateAppDefinedEnvVar(ctx, db, app.ID, "SECRET", appmodel.Variable{
				Key:         "SECRET",
				Value:       "",
				IsSensitive: true,
			})
			Expect(err).NotTo(HaveOccurred())

			// Read MongoDB directly so a successful decrypted read cannot hide a
			// regression where the empty sensitive value was stored as plaintext.
			var raw struct {
				Workload struct {
					EnvVars []struct {
						Key   string `bson:"key"`
						Value string `bson:"value"`
					} `bson:"envVars,omitempty"`
				} `bson:"workload"`
			}
			Expect(rawColl.FindOne(ctx, bson.M{"appID": app.ID}).Decode(&raw)).NotTo(HaveOccurred())
			varsByKey := map[string]string{}
			for _, envVar := range raw.Workload.EnvVars {
				varsByKey[envVar.Key] = envVar.Value
			}
			ciphertext, ok := varsByKey["SECRET"]
			Expect(ok).To(BeTrue())
			Expect(ciphertext).NotTo(BeEmpty())
			decryptedValue, err := crypto.AESDecrypt(svccfg.G.Encrypt.Secret, ciphertext)
			Expect(err).NotTo(HaveOccurred())
			Expect(decryptedValue).To(BeEmpty())

			envVars, err := appmodel.ListAppDefinedEnvVars(ctx, db, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(1))
			Expect(envVars[0].Key).To(Equal("SECRET"))
			Expect(envVars[0].Value).To(BeEmpty())
			Expect(envVars[0].IsSensitive).To(BeTrue())
		})
	})

	Describe("env var error paths", func() {
		var app *bkmsapp.Application

		BeforeEach(func() {
			app = dbfactory.Application(ctx, appStore)
			Expect(appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{
				AppID: app.ID,
				Workload: appmodel.Workload{
					EnvVars: []appmodel.Variable{{Key: "ENV_A", Value: "a"}, {Key: "ENV_B", Value: "b"}},
				},
			})).To(Succeed())
		})

		It("distinguishes a missing app model from a missing or conflicting env var", func() {
			Expect(appmodel.AddAppDefinedEnvVar(ctx, db, "non-existent-app", appmodel.Variable{Key: "ENV_C"})).
				To(MatchError(appmodel.ErrAppModelNotFound))
			Expect(appmodel.AddAppDefinedEnvVar(ctx, db, app.ID, appmodel.Variable{Key: "ENV_A"})).
				To(MatchError(appmodel.ErrEnvVarKeyExists))

			Expect(
				appmodel.UpdateAppDefinedEnvVar(ctx, db, "non-existent-app", "ENV_A", appmodel.Variable{Key: "ENV_A"}),
			).
				To(MatchError(appmodel.ErrAppModelNotFound))
			Expect(appmodel.UpdateAppDefinedEnvVar(ctx, db, app.ID, "ENV_X", appmodel.Variable{Key: "ENV_X"})).
				To(MatchError(appmodel.ErrEnvVarNotFound))
			Expect(appmodel.UpdateAppDefinedEnvVar(ctx, db, app.ID, "ENV_A", appmodel.Variable{Key: "ENV_B"})).
				To(MatchError(appmodel.ErrEnvVarKeyExists))

			Expect(appmodel.RemoveAppDefinedEnvVar(ctx, db, "non-existent-app", "ENV_A")).
				To(MatchError(appmodel.ErrAppModelNotFound))
			Expect(appmodel.RemoveAppDefinedEnvVar(ctx, db, app.ID, "ENV_X")).
				To(MatchError(appmodel.ErrEnvVarNotFound))

			envVars, err := appmodel.ListAppDefinedEnvVars(ctx, db, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(envVars, func(v appmodel.Variable, _ int) string { return v.Key })).
				To(ConsistOf("ENV_A", "ENV_B"))
		})

		It("renames and removes env vars", func() {
			Expect(
				appmodel.UpdateAppDefinedEnvVar(ctx, db, app.ID, "ENV_A", appmodel.Variable{Key: "ENV_C", Value: "c"}),
			).
				To(Succeed())
			Expect(appmodel.RemoveAppDefinedEnvVar(ctx, db, app.ID, "ENV_B")).To(Succeed())

			envVars, err := appmodel.ListAppDefinedEnvVars(ctx, db, app.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(envVars).To(HaveLen(1))
			Expect(envVars[0].Key).To(Equal("ENV_C"))
			Expect(envVars[0].Value).To(Equal("c"))
		})
	})
})

var _ = Describe("ComponentDef ref count maintained by AppModel functions", func() {
	var db *mongo.Database
	var compDefStore *component.ComponentDefStore
	var appStore *bkmsapp.ApplicationStore
	var ctx context.Context
	var compDefName string

	BeforeEach(func() {
		var err error

		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

		ctx = context.Background()

		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		db = testutil.NewDB(GinkgoT())
		compDefStore = store.Of(db, component.ComponentDefMeta)

		compDef := dbfactory.CompDef(ctx, compDefStore, &dbfactory.ComponentDefOpts{
			Properties: []component.Property{{Name: "key", Type: "STRING", Description: "A key"}},
		})
		compDefName = compDef.Name
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(appmodel.AppModelMeta)).To(Succeed())
	})

	getAppCompInstanceCount := func() int32 {
		def, err := compDefStore.Get(ctx, component.ByNameVersion(compDefName, component.DefaultComponentDefVersion))
		Expect(err).NotTo(HaveOccurred())
		return def.AppCompInstanceCount
	}

	It("should track appCompInstanceCount through the full AppModel lifecycle", func() {
		app := dbfactory.Application(ctx, appStore)

		// 创建空 AppModel
		err := appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{AppID: app.ID})
		Expect(err).NotTo(HaveOccurred())
		Expect(getAppCompInstanceCount()).To(Equal(int32(0)))

		// AddComponent -> appCompInstanceCount = 1
		err = appmodel.AddComponent(ctx, db, app.ID, &component.Component{
			Name: "comp-a",
			ComponentInst: component.ComponentInst{
				Type:    compDefName,
				Version: component.DefaultComponentDefVersion,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(getAppCompInstanceCount()).To(Equal(int32(1)))

		// 再添加同类型组件 -> appCompInstanceCount = 2
		err = appmodel.AddComponent(ctx, db, app.ID, &component.Component{
			Name: "comp-b",
			ComponentInst: component.ComponentInst{
				Type:    compDefName,
				Version: component.DefaultComponentDefVersion,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(getAppCompInstanceCount()).To(Equal(int32(2)))

		// 添加引用类型组件（RefWorkspaceCompName 非空）不应增加计数
		err = appmodel.AddComponent(ctx, db, app.ID, &component.Component{
			Name:         "comp-ref",
			ComponentRef: component.ComponentRef{RefWorkspaceCompName: "some-ws-comp"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(getAppCompInstanceCount()).To(Equal(int32(2)))

		// RemoveComponent comp-a -> appCompInstanceCount = 1
		err = appmodel.RemoveComponent(ctx, db, app.ID, "comp-a")
		Expect(err).NotTo(HaveOccurred())
		Expect(getAppCompInstanceCount()).To(Equal(int32(1)))

		// DeleteAppModel -> 剩余的 comp-b 被一并清除，appCompInstanceCount = 0
		err = appmodel.DeleteAppModel(ctx, db, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(getAppCompInstanceCount()).To(Equal(int32(0)))
	})

	It("should track appCompInstanceCount correctly across multiple AppModels", func() {
		app1 := dbfactory.Application(ctx, appStore)
		app2 := dbfactory.Application(ctx, appStore)

		for _, appID := range []string{app1.ID, app2.ID} {
			err := appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{AppID: appID})
			Expect(err).NotTo(HaveOccurred())

			err = appmodel.AddComponent(ctx, db, appID, &component.Component{
				Name: "comp-1",
				ComponentInst: component.ComponentInst{
					Type:    compDefName,
					Version: component.DefaultComponentDefVersion,
				},
			})
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(getAppCompInstanceCount()).To(Equal(int32(2)))

		// 删除第一个 AppModel -> appCompInstanceCount = 1
		err := appmodel.DeleteAppModel(ctx, db, app1.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(getAppCompInstanceCount()).To(Equal(int32(1)))

		// 删除第二个 AppModel -> appCompInstanceCount = 0
		err = appmodel.DeleteAppModel(ctx, db, app2.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(getAppCompInstanceCount()).To(Equal(int32(0)))
	})
})
