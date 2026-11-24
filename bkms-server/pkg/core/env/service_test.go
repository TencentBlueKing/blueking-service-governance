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

package env

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Test EnvService", func() {
	var diApp *fxtest.App
	var envSvc *EnvService

	var err error
	var ctx context.Context

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()

		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(
				&envSvc,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		Expect(testutil.CleanupCollection("environments")).NotTo(HaveOccurred())
		ResetHooksForTest()
		diApp.RequireStop()
	})

	Context("test env service methods", func() {
		var envName string
		var workspaceID string
		var Cluster model.BizCluster
		var envID bson.ObjectID
		var envType string

		BeforeEach(func() {
			envName = stringx.Random(10)
			workspaceID = stringx.Random(10)
			envType = stringx.Random(10)
			Cluster = model.BizCluster{
				ClusterID:   stringx.Random(10),
				ClusterType: stringx.Random(10),
				Namespace:   stringx.Random(10),
			}
			// test: create environment
			env := &model.Environment{
				Name:        envName,
				WorkspaceID: workspaceID,
				Cluster:     Cluster,
				Type:        envType,
				Description: stringx.Random(10),
			}
			err = envSvc.Create(ctx, env)
			Expect(err).NotTo(HaveOccurred())
			envID = env.ID
		})
		AfterEach(func() {
			// test: delete environment
			err = envSvc.Delete(ctx, envID)
		})

		It("test list environments", func() {
			environments, _ := store.Of(testutil.NewDB(GinkgoT()), model.EnvMeta).
				List(ctx, model.StdEnvsIn(workspaceID))
			Expect(environments).To(HaveLen(1))
			Expect(environments[0].Name).To(Equal(envName))
			Expect(environments[0].Status).To(Equal(model.EnvStatusReady))
		})

		It("test update cluster namespace failed when app already deployed", func() {
			appID := stringx.Random(10)
			Expect(model.AddApp(ctx, testutil.NewDB(GinkgoT()), envID, appID)).NotTo(HaveOccurred())

			newNamespace := stringx.Random(10)
			updateErr := envSvc.Update(ctx, envID, &model.EnvironmentUpdateData{
				Namespace: &newNamespace,
			})
			Expect(updateErr.Error()).To(ContainSubstring("cannot update cluster"))
		})

		It("returns domain conflict error when creating an env with occupied cluster namespace", func() {
			createErr := envSvc.Create(ctx, &model.Environment{
				Name:        "conflict-" + stringx.Random(6),
				DisplayName: "conflict",
				Type:        "test",
				WorkspaceID: "ws-" + stringx.Random(6),
				Cluster:     Cluster,
			})
			Expect(IsErrEnvClusterNamespaceOccupied(createErr)).To(BeTrue())

			occupiedErr, ok := GetEnvClusterNamespaceConflictInfo(createErr)
			Expect(ok).To(BeTrue())
			Expect(occupiedErr.ClusterID).To(Equal(Cluster.ClusterID))
			Expect(occupiedErr.Namespace).To(Equal(Cluster.Namespace))
			Expect(occupiedErr.OccupiedByEnvName).To(Equal(envName))
			Expect(occupiedErr.OccupiedByWorkspaceID).To(Equal(workspaceID))
		})

		It("returns domain conflict error when updating an env to an occupied cluster namespace", func() {
			occupiedCluster := model.BizCluster{
				ClusterID:   "cluster-" + stringx.Random(6),
				ClusterType: "single",
				Namespace:   "ns-" + stringx.Random(6),
			}
			occupiedEnvName := "occupied-" + stringx.Random(6)
			occupiedWorkspaceID := "ws-" + stringx.Random(6)
			createErr := envSvc.Create(ctx, &model.Environment{
				Name:        occupiedEnvName,
				DisplayName: occupiedEnvName,
				Type:        "test",
				WorkspaceID: occupiedWorkspaceID,
				Cluster:     occupiedCluster,
			})
			Expect(createErr).NotTo(HaveOccurred())

			updateErr := envSvc.Update(ctx, envID, &model.EnvironmentUpdateData{
				ClusterID:   &occupiedCluster.ClusterID,
				ClusterType: &occupiedCluster.ClusterType,
				Namespace:   &occupiedCluster.Namespace,
			})
			Expect(IsErrEnvClusterNamespaceOccupied(updateErr)).To(BeTrue())

			occupiedErr, ok := GetEnvClusterNamespaceConflictInfo(updateErr)
			Expect(ok).To(BeTrue())
			Expect(occupiedErr.ClusterID).To(Equal(occupiedCluster.ClusterID))
			Expect(occupiedErr.Namespace).To(Equal(occupiedCluster.Namespace))
			Expect(occupiedErr.OccupiedByEnvName).To(Equal(occupiedEnvName))
			Expect(occupiedErr.OccupiedByWorkspaceID).To(Equal(occupiedWorkspaceID))
		})

		It("rejects moving a source env while feature envs share its namespace", func() {
			envs := testutil.NewStore(GinkgoT(), model.EnvMeta)
			Expect(envs.Create(ctx, &model.Environment{
				Name:        "feat-" + stringx.Random(6),
				WorkspaceID: workspaceID,
				Kind:        model.EnvironmentKindFeature,
				SourceEnvID: envID,
				NsSharing:   model.NsSharing{ResPrefix: "fab12"},
				Cluster:     Cluster,
			})).To(Succeed())

			newNamespace := stringx.Random(10)
			updateErr := envSvc.Update(ctx, envID, &model.EnvironmentUpdateData{Namespace: &newNamespace})
			Expect(errors.Is(updateErr, ErrEnvHasNamespaceSharers)).To(BeTrue())

			// 集群与命名空间未实际变化时不拦截
			updateErr = envSvc.Update(ctx, envID, &model.EnvironmentUpdateData{
				ClusterID: &Cluster.ClusterID, Namespace: &Cluster.Namespace,
			})
			Expect(updateErr).NotTo(HaveOccurred())
		})

		It("allows moving a source env whose feature envs use their own namespaces", func() {
			envs := testutil.NewStore(GinkgoT(), model.EnvMeta)
			ownCluster := Cluster
			ownCluster.Namespace = stringx.Random(10)
			Expect(envs.Create(ctx, &model.Environment{
				Name:        "feat-" + stringx.Random(6),
				WorkspaceID: workspaceID,
				Kind:        model.EnvironmentKindFeature,
				SourceEnvID: envID,
				Cluster:     ownCluster,
			})).To(Succeed())

			newNamespace := stringx.Random(10)
			updateErr := envSvc.Update(ctx, envID, &model.EnvironmentUpdateData{Namespace: &newNamespace})
			Expect(updateErr).NotTo(HaveOccurred())
		})

		It("rejects moving an env sharing namespace into another namespace", func() {
			otherCluster := model.BizCluster{
				ClusterID:   "cluster-" + stringx.Random(6),
				ClusterType: "single",
				Namespace:   "ns-" + stringx.Random(6),
			}
			Expect(envSvc.Create(ctx, &model.Environment{
				Name:        "other-" + stringx.Random(6),
				DisplayName: "other",
				Type:        "test",
				WorkspaceID: "ws-" + stringx.Random(6),
				Cluster:     otherCluster,
			})).To(Succeed())

			featEnv := &model.Environment{
				Name:        "feat-" + stringx.Random(6),
				WorkspaceID: workspaceID,
				Kind:        model.EnvironmentKindFeature,
				SourceEnvID: envID,
				NsSharing:   model.NsSharing{ResPrefix: "fab12"},
				Cluster:     Cluster,
			}
			Expect(envSvc.Create(ctx, featEnv)).To(Succeed())

			updateErr := envSvc.Update(ctx, featEnv.ID, &model.EnvironmentUpdateData{
				ClusterID: &otherCluster.ClusterID,
				Namespace: &otherCluster.Namespace,
			})
			Expect(errors.Is(updateErr, model.ErrSharedNamespaceUnsupported)).To(BeTrue())

			// 集群与命名空间未实际变化时不拦截
			updateErr = envSvc.Update(ctx, featEnv.ID, &model.EnvironmentUpdateData{
				ClusterID: &Cluster.ClusterID, Namespace: &Cluster.Namespace,
			})
			Expect(updateErr).NotTo(HaveOccurred())
		})

		It("runs update hooks with before and after env when type changes", func() {
			beforeType := envType
			afterType := "staging"
			hookCalled := make(chan struct{}, 1)

			registered := RegisterUpdateHook("test.update_env", func(
				_ context.Context,
				before model.Environment,
				after model.Environment,
			) error {
				Expect(before.ID).To(Equal(envID))
				Expect(before.Type).To(Equal(beforeType))
				Expect(after.ID).To(Equal(envID))
				Expect(after.Type).To(Equal(afterType))
				hookCalled <- struct{}{}
				return nil
			})
			Expect(registered).To(BeTrue())

			updateErr := envSvc.Update(ctx, envID, &model.EnvironmentUpdateData{
				Type: &afterType,
			})
			Expect(updateErr).NotTo(HaveOccurred())
			Eventually(hookCalled).Should(Receive())
		})

		It("resets delete and update hooks together", func() {
			Expect(RegisterDeleteHook("test.delete_env", func(context.Context, model.Environment) error {
				return nil
			})).To(BeTrue())
			Expect(RegisterUpdateHook("test.update_env", func(
				context.Context,
				model.Environment,
				model.Environment,
			) error {
				return nil
			})).To(BeTrue())

			Expect(IsDeleteHookRegistered("test.delete_env")).To(BeTrue())
			Expect(IsUpdateHookRegistered("test.update_env")).To(BeTrue())

			ResetHooksForTest()

			Expect(IsDeleteHookRegistered("test.delete_env")).To(BeFalse())
			Expect(IsUpdateHookRegistered("test.update_env")).To(BeFalse())
		})

		DescribeTable("rejects deleting a source env while derived feature envs exist",
			func(shareNamespace bool) {
				envs := testutil.NewStore(GinkgoT(), model.EnvMeta)
				featCluster := Cluster
				resourcePrefix := "fab12"
				if !shareNamespace {
					featCluster.Namespace = "feat-ns-" + stringx.Random(6)
					resourcePrefix = ""
				}
				featEnv := &model.Environment{
					Name:        "feat-" + stringx.Random(6),
					WorkspaceID: workspaceID,
					Kind:        model.EnvironmentKindFeature,
					SourceEnvID: envID,
					NsSharing:   model.NsSharing{ResPrefix: resourcePrefix},
					Cluster:     featCluster,
				}
				Expect(envs.Create(ctx, featEnv)).To(Succeed())

				deleteErr := envSvc.Delete(ctx, envID)
				Expect(errors.Is(deleteErr, ErrEnvHasFeatureEnvs)).To(BeTrue())
				Expect(deleteErr).To(MatchError(ContainSubstring(featEnv.Name)))

				// 来源环境在删除被拒后仍然存在
				_, getErr := envs.GetByID(ctx, envID)
				Expect(getErr).NotTo(HaveOccurred())

				// 先删除特性环境后，来源环境即可正常删除
				Expect(envSvc.Delete(ctx, featEnv.ID)).To(Succeed())
				Expect(envSvc.Delete(ctx, envID)).To(Succeed())
			},
			Entry("dedicated-namespace feature env", false),
			Entry("shared-namespace feature env", true),
		)
	})
})
