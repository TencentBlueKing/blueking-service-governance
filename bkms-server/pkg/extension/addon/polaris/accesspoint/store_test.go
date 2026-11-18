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

package accesspoint_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("AccessPointMeta", func() {
	var (
		ctx         context.Context
		apStore     *accesspoint.AccessPointStore
		workspaceID string
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		apStore = testutil.NewStore(GinkgoT(), accesspoint.AccessPointMeta)
		workspaceID = "ws-" + stringx.Random(8)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(accesspoint.AccessPointMeta)).To(Succeed())
	})

	It("should store the token encrypted and decrypt it on read", func() {
		p := &accesspoint.PolarisAccessPoint{
			WorkspaceID: workspaceID,
			Name:        "tse",
			ClientHosts: []string{"10.0.0.1"},
			Token:       "secret-token",
		}
		Expect(apStore.Create(ctx, p)).To(Succeed())

		var raw bson.M
		Expect(apStore.Collection().FindOne(ctx, bson.M{"_id": p.ID}).Decode(&raw)).To(Succeed())
		Expect(raw["token"]).NotTo(BeEmpty())
		Expect(raw["token"]).NotTo(Equal("secret-token"))

		got, err := apStore.Get(ctx, store.ByID(p.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Token).To(Equal("secret-token"))
	})

	It("should keep an empty token empty", func() {
		p := &accesspoint.PolarisAccessPoint{WorkspaceID: workspaceID, Name: "tse", ClientHosts: []string{"10.0.0.1"}}
		Expect(apStore.Create(ctx, p)).To(Succeed())

		got, err := apStore.Get(ctx, store.ByID(p.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Token).To(BeEmpty())
	})

	It("should reject a duplicate name in the same workspace only", func() {
		Expect(apStore.Create(ctx, &accesspoint.PolarisAccessPoint{WorkspaceID: workspaceID, Name: "tse"})).
			To(Succeed())

		err := apStore.Create(ctx, &accesspoint.PolarisAccessPoint{WorkspaceID: workspaceID, Name: "tse"})
		Expect(err).To(MatchError(accesspoint.ErrAccessPointNameExists))

		Expect(apStore.Create(ctx, &accesspoint.PolarisAccessPoint{WorkspaceID: workspaceID + "-other", Name: "tse"})).
			To(Succeed())
	})
})

var _ = Describe("BindingMeta", func() {
	var (
		ctx         context.Context
		bindStore   *accesspoint.BindingStore
		workspaceID string
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		bindStore = testutil.NewStore(GinkgoT(), accesspoint.BindingMeta)
		workspaceID = "ws-" + stringx.Random(8)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(accesspoint.BindingMeta)).To(Succeed())
	})

	It("should allow a cluster to be bound only once in a workspace", func() {
		binding := func(wsID, accessPointID string) *accesspoint.PolarisClusterBinding {
			return &accesspoint.PolarisClusterBinding{
				WorkspaceID: wsID, ClusterID: "BCS-K8S-00001", AccessPointID: accessPointID,
			}
		}
		Expect(bindStore.Create(ctx, binding(workspaceID, "ap-1"))).To(Succeed())

		err := bindStore.Create(ctx, binding(workspaceID, "ap-2"))
		Expect(err).To(MatchError(accesspoint.ErrBindingExists))

		// 不做跨空间唯一
		Expect(bindStore.Create(ctx, binding(workspaceID+"-other", "ap-3"))).To(Succeed())
	})
})
