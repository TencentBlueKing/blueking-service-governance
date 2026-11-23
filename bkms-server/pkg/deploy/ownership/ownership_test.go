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

package ownership_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/ownership"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

func newLive(labels map[string]string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetKind("ConfigMap")
	obj.SetName("web")
	obj.SetLabels(labels)
	return obj
}

var _ = Describe("Owner", func() {
	var (
		ctx         context.Context
		standardEnv *envmodel.Environment
		sharedEnv   *envmodel.Environment
	)

	forApp := func(env *envmodel.Environment, appID string) ownership.Owner {
		GinkgoHelper()
		owner, err := ownership.ForApp(ctx, testutil.NewDB(GinkgoT()), env, appID)
		Expect(err).NotTo(HaveOccurred())
		return owner
	}

	forEnv := func(env *envmodel.Environment) ownership.Owner {
		GinkgoHelper()
		owner, err := ownership.ForEnv(ctx, testutil.NewDB(GinkgoT()), env)
		Expect(err).NotTo(HaveOccurred())
		return owner
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		cluster := envmodel.BizCluster{ClusterID: "cluster-" + stringx.Random(6), Namespace: "ns-" + stringx.Random(6)}
		standardEnv = &envmodel.Environment{Name: "dev", Cluster: cluster}
		sharedEnv = &envmodel.Environment{Name: "feat-app-1-1", ResourcePrefix: "fab12", Cluster: cluster}
	})

	Describe("Stamp", func() {
		It("adds env and app labels while keeping existing labels", func() {
			obj := newLive(map[string]string{"app.kubernetes.io/name": "web"})
			forApp(standardEnv, "app-1").Stamp(obj)
			Expect(obj.GetLabels()).To(Equal(map[string]string{
				"app.kubernetes.io/name":  "web",
				ownership.LabelKeyEnvName: "dev",
				ownership.LabelKeyAppID:   "app-1",
			}))
		})

		It("adds only the env label for env scoped objects", func() {
			manifest := map[string]any{"metadata": map[string]any{"name": "pool"}}
			forEnv(sharedEnv).StampManifest(manifest)
			Expect((&unstructured.Unstructured{Object: manifest}).GetLabels()).To(Equal(map[string]string{
				ownership.LabelKeyEnvName: "feat-app-1-1",
			}))
		})

		It("stamps typed object metadata", func() {
			meta := &metav1.ObjectMeta{Name: "web"}
			forApp(standardEnv, "app-1").Stamp(meta)
			Expect(meta.Labels).To(HaveKeyWithValue(ownership.LabelKeyAppID, "app-1"))
		})
	})

	DescribeTable("Check",
		func(shared bool, appID string, labels map[string]string, wantErr bool) {
			env := standardEnv
			if shared {
				env = sharedEnv
			}
			if labels[ownership.LabelKeyEnvName] == "self" {
				labels[ownership.LabelKeyEnvName] = env.Name
			}
			owner := forEnv(env)
			if appID != "" {
				owner = forApp(env, appID)
			}

			err := owner.Check(newLive(labels))
			if wantErr {
				Expect(errors.Is(err, ownership.ErrNotOwned)).To(BeTrue())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("standard env owns its labeled object", false, "app-1",
			map[string]string{ownership.LabelKeyEnvName: "self", ownership.LabelKeyAppID: "app-1"}, false),
		Entry("standard env rejects an object of another env", false, "app-1",
			map[string]string{ownership.LabelKeyEnvName: "prod", ownership.LabelKeyAppID: "app-1"}, true),
		Entry("standard env rejects an object of another app", false, "app-1",
			map[string]string{ownership.LabelKeyEnvName: "self", ownership.LabelKeyAppID: "app-2"}, true),
		Entry("standard env takes over an unlabeled object", false, "app-1",
			map[string]string{}, false),
		Entry("shared env owns its labeled object", true, "app-1",
			map[string]string{ownership.LabelKeyEnvName: "self", ownership.LabelKeyAppID: "app-1"}, false),
		Entry("shared env rejects an object of the source env", true, "app-1",
			map[string]string{ownership.LabelKeyEnvName: "dev", ownership.LabelKeyAppID: "app-1"}, true),
		Entry("shared env rejects an unlabeled object", true, "app-1",
			map[string]string{}, true),
		Entry("shared env rejects an object missing the app label", true, "app-1",
			map[string]string{ownership.LabelKeyEnvName: "self"}, true),
		Entry("shared env owns its env scoped object without app label", true, "",
			map[string]string{ownership.LabelKeyEnvName: "self"}, false),
	)

	Describe("NeedVerify", func() {
		It("verifies in an env sharing namespace", func() {
			Expect(forApp(sharedEnv, "app-1").NeedVerify()).To(BeTrue())
		})

		It("skips verification in a standard env with an exclusive namespace", func() {
			Expect(forApp(standardEnv, "app-1").NeedVerify()).To(BeFalse())
		})

		It("verifies in a standard env whose namespace is shared by feature envs", func() {
			envs := testutil.NewStore(GinkgoT(), envmodel.EnvMeta)
			standardEnv.Name = "dev-" + stringx.Random(6)
			standardEnv.WorkspaceID = "ws-" + stringx.Random(6)
			Expect(envs.Create(ctx, standardEnv)).To(Succeed())
			sharedEnv.Name = "feat-" + stringx.Random(6)
			sharedEnv.WorkspaceID = standardEnv.WorkspaceID
			sharedEnv.Kind = envmodel.EnvironmentKindFeature
			sharedEnv.SourceEnvID = standardEnv.ID
			Expect(envs.Create(ctx, sharedEnv)).To(Succeed())
			DeferCleanup(func() {
				_, err := envs.DeleteMany(ctx, store.Where("workspaceID", standardEnv.WorkspaceID))
				Expect(err).NotTo(HaveOccurred())
			})

			Expect(forApp(standardEnv, "app-1").NeedVerify()).To(BeTrue())
		})
	})

	Describe("Claim", func() {
		It("stamps the manifest when no object with the same name exists", func() {
			mockey.PatchConvey("claim-not-found", GinkgoT(), func() {
				mockey.Mock((*k8sclient.Client).Get).Return(nil, k8sclient.ErrResourceNotFound).Build()

				manifest := map[string]any{"metadata": map[string]any{"name": "web"}}
				err := forApp(sharedEnv, "app-1").Claim(ctx, &k8sclient.Client{}, "ns-1", manifest)
				Expect(err).NotTo(HaveOccurred())
				Expect((&unstructured.Unstructured{Object: manifest}).GetLabels()).
					To(HaveKeyWithValue(ownership.LabelKeyEnvName, "feat-app-1-1"))
			})
		})

		It("leaves the manifest untouched when the object is not owned", func() {
			mockey.PatchConvey("claim-not-owned", GinkgoT(), func() {
				mockey.Mock((*k8sclient.Client).Get).Return(newLive(nil), nil).Build()

				manifest := map[string]any{"metadata": map[string]any{"name": "web"}}
				err := forApp(sharedEnv, "app-1").Claim(ctx, &k8sclient.Client{}, "ns-1", manifest)
				Expect(errors.Is(err, ownership.ErrNotOwned)).To(BeTrue())
				Expect((&unstructured.Unstructured{Object: manifest}).GetLabels()).To(BeEmpty())
			})
		})

		It("returns the error when getting the object fails", func() {
			mockey.PatchConvey("claim-get-failed", GinkgoT(), func() {
				mockey.Mock((*k8sclient.Client).Get).Return(nil, errors.New("connection refused")).Build()

				err := forApp(sharedEnv, "app-1").Claim(
					ctx, &k8sclient.Client{}, "ns-1", map[string]any{"metadata": map[string]any{"name": "web"}},
				)
				Expect(err).To(MatchError(ContainSubstring("connection refused")))
				Expect(errors.Is(err, ownership.ErrNotOwned)).To(BeFalse())
			})
		})

		It("only stamps without querying the cluster in a standard env", func() {
			mockey.PatchConvey("claim-skip-verify", GinkgoT(), func() {
				getMock := mockey.Mock((*k8sclient.Client).Get).Return(newLive(nil), nil).Build()

				manifest := map[string]any{"metadata": map[string]any{"name": "web"}}
				err := forApp(standardEnv, "app-1").Claim(ctx, &k8sclient.Client{}, "ns-1", manifest)
				Expect(err).NotTo(HaveOccurred())
				Expect(getMock.Times()).To(Equal(0))
				Expect((&unstructured.Unstructured{Object: manifest}).GetLabels()).
					To(HaveKeyWithValue(ownership.LabelKeyAppID, "app-1"))
			})
		})
	})
})
