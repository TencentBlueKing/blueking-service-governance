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

package clustertoken_test

import (
	"context"
	"errors"
	"strings"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/clustertoken"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const (
	tokenA = "bkt-aaaaaaaaaaaaaaaa1111" //nolint:gosec // G101: 测试用的假 Token
	tokenB = "bkt-bbbbbbbbbbbbbbbb2222" //nolint:gosec // G101: 测试用的假 Token
)

var _ = Describe("Manager", func() {
	var (
		ctx         context.Context
		manager     *clustertoken.Manager
		resolver    *clustertoken.Resolver
		envSvc      *bkmsenv.EnvService
		tokenStore  *clustertoken.TokenStore
		wcStore     *clustertoken.WorkspaceClusterStore
		workspaceID string
		projectID   string
		bcsClient   *fakeBCS
	)

	mustCreateToken := func(token string) *clustertoken.ClusterToken {
		t, err := manager.CreateToken(ctx, workspaceID, token)
		Expect(err).NotTo(HaveOccurred())
		return t
	}

	mustAccess := func(ref clustertoken.TokenRef, clusterIDs ...string) *clustertoken.AccessResult {
		result, err := manager.Access(ctx, workspaceID, projectID, clusterIDs, ref)
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	BeforeEach(func() {
		ctx = auth.WithUser(testutil.NewDefaultTenantContext(), auth.User{ID: "ws-admin"})
		app := fxtest.New(GinkgoT(),
			clustertoken.FxModule, bkmsenv.FxModule,
			fx.Populate(&manager, &resolver, &envSvc),
		)
		app.RequireStart()
		DeferCleanup(app.RequireStop)

		tokenStore = testutil.NewStore(GinkgoT(), clustertoken.TokenMeta)
		wcStore = testutil.NewStore(GinkgoT(), clustertoken.WorkspaceClusterMeta)
		workspaceID = "ws-" + stringx.Random(8)
		projectID = "proj-" + stringx.Random(8)

		bcsClient = &fakeBCS{
			StubApiClient: bcs.NewStub(auth.User{}),
			clusters: []bcs.Cluster{
				{ID: "BCS-K8S-00001", Name: "dedicated-1", Type: "single", Status: "RUNNING"},
				{ID: "BCS-K8S-00002", Name: "dedicated-2", Type: "single", Status: "RUNNING"},
				{ID: "BCS-K8S-00003", Name: "shared", Type: "virtual", Status: "RUNNING", IsShared: true},
			},
		}
		mockey.Mock(bcs.New).Return(bcs.Client(bcsClient), nil).Build()
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		Expect(testutil.CleanupMetaColl(clustertoken.TokenMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(clustertoken.WorkspaceClusterMeta)).To(Succeed())
	})

	Describe("CreateToken", func() {
		It("should make the first token default and mask it", func() {
			first := mustCreateToken(tokenA)
			second := mustCreateToken(tokenB)

			Expect(first.IsDefault).To(BeTrue())
			Expect(second.IsDefault).To(BeFalse())
			Expect(first.Masked).To(Equal("bkt-****1111"))
			Expect(first.Creator).To(Equal("ws-admin"))
		})

		It("should store the token encrypted", func() {
			t := mustCreateToken(tokenA)

			var raw bson.M
			err := testutil.NewDB(GinkgoT()).Collection(clustertoken.TokenMeta.Collection).
				FindOne(ctx, bson.M{"_id": t.ID}).Decode(&raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(raw["token"]).NotTo(Equal(tokenA))

			got, err := tokenStore.Get(ctx, store.ByID(t.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Token).To(Equal(tokenA))
		})

		It("should trim spaces and reject duplicated tokens", func() {
			mustCreateToken(tokenA)

			_, err := manager.CreateToken(ctx, workspaceID, "  "+tokenA+"\n")

			Expect(errors.Is(err, clustertoken.ErrTokenExists)).To(BeTrue())
		})

		It("should allow the same token in another workspace", func() {
			mustCreateToken(tokenA)

			t, err := manager.CreateToken(ctx, "ws-"+stringx.Random(8), tokenA)

			Expect(err).NotTo(HaveOccurred())
			Expect(t.IsDefault).To(BeTrue())
		})

		DescribeTable("should reject invalid tokens",
			func(token string) {
				_, err := manager.CreateToken(ctx, workspaceID, token)
				Expect(errors.Is(err, clustertoken.ErrTokenInvalid)).To(BeTrue())
			},
			Entry("empty", "   "),
			Entry("inner space", "bkt abc"),
			Entry("non ascii", "bkt-令牌"),
		)
	})

	Describe("ReplaceToken", func() {
		It("should replace the value and keep the default flag", func() {
			t := mustCreateToken(tokenA)

			before, after, err := manager.ReplaceToken(ctx, workspaceID, t.ID.Hex(), tokenB)

			Expect(err).NotTo(HaveOccurred())
			Expect(before.Masked).To(Equal("bkt-****1111"))
			Expect(after.Masked).To(Equal("bkt-****2222"))
			got, err := tokenStore.Get(ctx, store.ByID(t.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Token).To(Equal(tokenB))
			Expect(got.IsDefault).To(BeTrue())
		})

		It("should reject a value used by another token", func() {
			mustCreateToken(tokenA)
			t := mustCreateToken(tokenB)

			_, _, err := manager.ReplaceToken(ctx, workspaceID, t.ID.Hex(), tokenA)

			Expect(errors.Is(err, clustertoken.ErrTokenExists)).To(BeTrue())
		})

		It("should not find a token of another workspace", func() {
			t := mustCreateToken(tokenA)

			_, _, err := manager.ReplaceToken(ctx, "ws-other", t.ID.Hex(), tokenB)

			Expect(errors.Is(err, clustertoken.ErrTokenNotFound)).To(BeTrue())
		})
	})

	Describe("SetDefaultToken", func() {
		It("should move the default flag", func() {
			first := mustCreateToken(tokenA)
			second := mustCreateToken(tokenB)

			t, err := manager.SetDefaultToken(ctx, workspaceID, second.ID.Hex())

			Expect(err).NotTo(HaveOccurred())
			Expect(t.IsDefault).To(BeTrue())
			defaults, err := tokenStore.List(ctx, clustertoken.DefaultToken(workspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(defaults).To(HaveLen(1))
			Expect(defaults[0].ID).To(Equal(second.ID))
			Expect(defaults[0].ID).NotTo(Equal(first.ID))
		})

		It("should restore the previous default token when setting the new one fails", func() {
			first := mustCreateToken(tokenA)
			second := mustCreateToken(tokenB)
			var origin func(*clustertoken.TokenStore, context.Context, any, ...store.Opt) error
			failed := false
			mockey.Mock((*clustertoken.TokenStore).Update, mockey.OptGeneric).To(
				func(s *clustertoken.TokenStore, ctx context.Context, target any, opts ...store.Opt) error {
					if !failed {
						failed = true
						return errors.New("connection reset")
					}
					return origin(s, ctx, target, opts...)
				},
			).Origin(&origin).Build()

			_, err := manager.SetDefaultToken(ctx, workspaceID, second.ID.Hex())

			Expect(err).To(MatchError(ContainSubstring("connection reset")))
			defaults, err := tokenStore.List(ctx, clustertoken.DefaultToken(workspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(defaults).To(ConsistOf(HaveField("ID", first.ID)))
		})

		It("should enforce at most one default token per workspace by index", func() {
			mustCreateToken(tokenA)

			err := tokenStore.Create(ctx, &clustertoken.ClusterToken{
				WorkspaceID: workspaceID, Token: tokenB, Fingerprint: "other", IsDefault: true,
			})

			Expect(errors.Is(err, store.ErrDuplicate)).To(BeTrue())
		})
	})

	Describe("DeleteToken", func() {
		It("should delete an unused token", func() {
			t := mustCreateToken(tokenA)

			_, err := manager.DeleteToken(ctx, workspaceID, t.ID.Hex())

			Expect(err).NotTo(HaveOccurred())
			exists, err := tokenStore.Exists(ctx, store.ByID(t.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should reject deleting a token used by clusters and list them", func() {
			t := mustCreateToken(tokenA)
			mustAccess(clustertoken.TokenRef{TokenID: t.ID.Hex()}, "BCS-K8S-00001")

			_, err := manager.DeleteToken(ctx, workspaceID, t.ID.Hex())

			var inUse *clustertoken.TokenInUseError
			Expect(errors.As(err, &inUse)).To(BeTrue())
			Expect(inUse.Clusters).To(Equal([]clustertoken.ClusterRef{
				{ClusterID: "BCS-K8S-00001", ClusterName: "dedicated-1"},
			}))

			apiErr := clustertoken.APIError(err, "delete").(*bkerrs.Error)
			Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(apiErr.Details()).To(Equal(bkerrs.Details{bkerrs.NewDetail(
				bkerrs.ErrDetailCodeClusterTokenInUse,
				apiErr.Error(),
				bkerrs.WithSystem(bkerrs.SystemName),
				bkerrs.WithModule("cluster"),
				bkerrs.WithExtras(map[string]string{
					"clusters": `[{"clusterID":"BCS-K8S-00001","clusterName":"dedicated-1"}]`,
				}),
			)}))
		})

		It("should reject deleting a token used by clusters when the id is in upper case", func() {
			t := mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001").Token
			upperID := strings.ToUpper(t.ID.Hex())
			Expect(upperID).NotTo(Equal(t.ID.Hex()), "the id must contain a-f to cover the case")

			_, err := manager.DeleteToken(ctx, workspaceID, upperID)

			Expect(errors.Is(err, clustertoken.ErrTokenInUse)).To(BeTrue())
			exists, err := tokenStore.Exists(ctx, store.ByID(t.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})

	Describe("ListTokens", func() {
		It("should count clusters using each token", func() {
			t := mustCreateToken(tokenA)
			mustCreateToken(tokenB)
			mustAccess(clustertoken.TokenRef{TokenID: t.ID.Hex()}, "BCS-K8S-00001", "BCS-K8S-00002")

			result, err := manager.ListTokens(ctx, workspaceID)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Tokens).To(HaveLen(2))
			Expect(result.ClusterCounts).To(Equal(map[string]int{t.ID.Hex(): 2}))
		})
	})

	Describe("Access", func() {
		It("should access clusters with a new token, including shared clusters", func() {
			result := mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001", "BCS-K8S-00003")

			Expect(result.Token.IsDefault).To(BeTrue())
			Expect(result.Clusters).To(HaveLen(2))
			shared, err := wcStore.Get(ctx, clustertoken.ByCluster(workspaceID, "BCS-K8S-00003"))
			Expect(err).NotTo(HaveOccurred())
			Expect(shared.IsShared).To(BeTrue())
			Expect(shared.ClusterName).To(Equal("shared"))
			Expect(shared.TokenID).To(Equal(result.Token.ID.Hex()))
		})

		It("should reuse an existing token with the same value", func() {
			t := mustCreateToken(tokenA)

			result := mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")

			Expect(result.Token.ID).To(Equal(t.ID))
			n, err := tokenStore.Count(ctx, clustertoken.InWorkspace(workspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeEquivalentTo(1))
		})

		It("should reject clusters outside the bcs project without creating the token", func() {
			_, err := manager.Access(ctx, workspaceID, projectID,
				[]string{"BCS-K8S-00001", "BCS-K8S-99999"}, clustertoken.TokenRef{Token: tokenA})

			Expect(errors.Is(err, clustertoken.ErrClusterNotInProject)).To(BeTrue())
			n, err := tokenStore.Count(ctx, clustertoken.InWorkspace(workspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeZero())
		})

		It("should reject clusters already accessed", func() {
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")

			_, err := manager.Access(ctx, workspaceID, projectID,
				[]string{"BCS-K8S-00001", "BCS-K8S-00002"}, clustertoken.TokenRef{Token: tokenA})

			Expect(errors.Is(err, clustertoken.ErrWorkspaceClusterExists)).To(BeTrue())
			n, err := wcStore.Count(ctx, clustertoken.InWorkspace(workspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeEquivalentTo(1))
		})

		DescribeTable("should require exactly one of tokenID and token",
			func(ref clustertoken.TokenRef) {
				_, err := manager.Access(ctx, workspaceID, projectID, []string{"BCS-K8S-00001"}, ref)
				Expect(errors.Is(err, clustertoken.ErrTokenRefInvalid)).To(BeTrue())
			},
			Entry("none", clustertoken.TokenRef{}),
			Entry("both", clustertoken.TokenRef{TokenID: bson.NewObjectID().Hex(), Token: tokenA}),
		)

		It("should reject an unknown token id", func() {
			_, err := manager.Access(ctx, workspaceID, projectID,
				[]string{"BCS-K8S-00001"}, clustertoken.TokenRef{TokenID: bson.NewObjectID().Hex()})

			Expect(errors.Is(err, clustertoken.ErrTokenNotFound)).To(BeTrue())
		})
	})

	Describe("ListCandidates", func() {
		It("should list all project clusters and mark accessed ones", func() {
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00003")

			candidates, err := manager.ListCandidates(ctx, workspaceID, projectID)

			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(candidates, func(c clustertoken.Candidate, _ int) string { return c.ID })).
				To(Equal([]string{"BCS-K8S-00001", "BCS-K8S-00002", "BCS-K8S-00003"}))
			Expect(lo.Map(candidates, func(c clustertoken.Candidate, _ int) bool { return c.Accessed })).
				To(Equal([]bool{false, false, true}))
		})

		It("should return nothing when the workspace has no bcs project", func() {
			candidates, err := manager.ListCandidates(ctx, workspaceID, "")

			Expect(err).NotTo(HaveOccurred())
			Expect(candidates).To(BeEmpty())
		})
	})

	Describe("SwitchToken", func() {
		It("should switch to another token", func() {
			result := mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")

			before, after, token, err := manager.SwitchToken(ctx, workspaceID, "BCS-K8S-00001",
				clustertoken.TokenRef{Token: tokenB})

			Expect(err).NotTo(HaveOccurred())
			Expect(before.TokenID).To(Equal(result.Token.ID.Hex()))
			Expect(after.TokenID).To(Equal(token.ID.Hex()))
			got, err := wcStore.Get(ctx, clustertoken.ByCluster(workspaceID, "BCS-K8S-00001"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.TokenID).To(Equal(token.ID.Hex()))
		})

		It("should reject a cluster not accessed", func() {
			_, _, _, err := manager.SwitchToken(
				ctx,
				workspaceID,
				"BCS-K8S-00001",
				clustertoken.TokenRef{Token: tokenB},
			)

			Expect(errors.Is(err, clustertoken.ErrWorkspaceClusterNotFound)).To(BeTrue())
		})
	})

	Describe("Remove and ListClusters", func() {
		createEnvOn := func(clusterID string) *envmodel.Environment {
			return dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
				WorkspaceID: workspaceID,
				Cluster: &envmodel.BizCluster{
					ProjectCode: "proj", ClusterID: clusterID, Namespace: "ns-" + stringx.Random(6),
				},
			})
		}

		It("should count envs on each accessed cluster", func() {
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001", "BCS-K8S-00002")
			createEnvOn("BCS-K8S-00001")
			createEnvOn("BCS-K8S-00001")

			result, err := manager.ListClusters(ctx, workspaceID, "")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Clusters).To(HaveLen(2))
			Expect(result.EnvCounts).To(HaveKeyWithValue("BCS-K8S-00001", 2))
			Expect(result.EnvCounts).NotTo(HaveKey("BCS-K8S-00002"))
			Expect(result.Tokens).To(HaveLen(1))
		})

		It("should list only clusters using the given token", func() {
			first := mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")
			mustAccess(clustertoken.TokenRef{Token: tokenB}, "BCS-K8S-00002")
			createEnvOn("BCS-K8S-00001")

			result, err := manager.ListClusters(ctx, workspaceID, first.Token.ID.Hex())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Clusters).To(ConsistOf(HaveField("ClusterID", "BCS-K8S-00001")))
			Expect(result.EnvCounts).To(HaveKeyWithValue("BCS-K8S-00001", 1))
		})

		It("should reject removing a cluster used by envs", func() {
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")
			env := createEnvOn("BCS-K8S-00001")

			_, err := manager.Remove(ctx, workspaceID, "BCS-K8S-00001", nil)

			var inUse *clustertoken.ClusterInUseError
			Expect(errors.As(err, &inUse)).To(BeTrue())
			Expect(inUse.Envs).To(Equal([]clustertoken.EnvRef{{Name: env.Name, DisplayName: env.DisplayName}}))
			Expect(inUse.PolarisBound).To(BeFalse())
		})

		It("should reject removing a cluster bound to a polaris access point", func() {
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")

			_, err := manager.Remove(ctx, workspaceID, "BCS-K8S-00001",
				func(context.Context, string, string) (bool, error) { return true, nil })

			var inUse *clustertoken.ClusterInUseError
			Expect(errors.As(err, &inUse)).To(BeTrue())
			Expect(inUse.PolarisBound).To(BeTrue())
			Expect(inUse.Extras()).To(Equal(map[string]string{"envs": "[]", "polarisBound": "true"}))
		})

		It("should remove an unused cluster", func() {
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")

			wc, err := manager.Remove(ctx, workspaceID, "BCS-K8S-00001",
				func(context.Context, string, string) (bool, error) { return false, nil })

			Expect(err).NotTo(HaveOccurred())
			Expect(wc.ClusterID).To(Equal("BCS-K8S-00001"))
			exists, err := wcStore.Exists(ctx, clustertoken.ByCluster(workspaceID, "BCS-K8S-00001"))
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("Resolver", func() {
		It("should return the token assigned to the cluster when the switch is on", func() {
			setBCSConfig(bcsConfig(true))
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")

			token, err := resolver.ResolveToken(ctx, workspaceID, "BCS-K8S-00001")

			Expect(err).NotTo(HaveOccurred())
			Expect(token).To(Equal(tokenA))
		})

		It("should not fall back to the platform token for clusters not accessed", func() {
			setBCSConfig(bcsConfig(true))
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")

			_, err := resolver.ResolveToken(ctx, "ws-other", "BCS-K8S-00001")

			Expect(errors.Is(err, clustertoken.ErrClusterNotAccessed)).To(BeTrue())
		})

		It("should not check cluster access when the switch is off", func() {
			setBCSConfig(bcsConfig(false))

			Expect(resolver.CheckAccessed(ctx, workspaceID, "BCS-K8S-00001")).To(Succeed())
		})

		It("should check cluster access when the switch is on", func() {
			setBCSConfig(bcsConfig(true))
			mustAccess(clustertoken.TokenRef{Token: tokenA}, "BCS-K8S-00001")

			Expect(resolver.CheckAccessed(ctx, workspaceID, "BCS-K8S-00001")).To(Succeed())
			err := resolver.CheckAccessed(ctx, workspaceID, "BCS-K8S-00002")
			Expect(errors.Is(err, clustertoken.ErrClusterNotAccessed)).To(BeTrue())
		})
	})
})
