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

package workspace

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

type fakeBCSProjectClient struct {
	*bcs.StubApiClient
	existing     *bcs.Project
	getErr       error
	created      *bcs.Project
	createErr    error
	lastCreate   bcs.CreateProjectInput
	createCalled bool
}

func (f *fakeBCSProjectClient) GetProject(_ context.Context, _ string) (*bcs.Project, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.existing, nil
}

func (f *fakeBCSProjectClient) CreateProject(_ context.Context, in bcs.CreateProjectInput) (*bcs.Project, error) {
	f.createCalled = true
	f.lastCreate = in
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.created, nil
}

var _ = Describe("resolveCMDBInfo", func() {
	var originCfg *config.Config

	BeforeEach(func() {
		originCfg = config.G
		config.G = &config.Config{
			FeatureCommunity: config.FeatureCommunityConfig{CreateBCSProject: true},
		}
	})

	AfterEach(func() {
		config.G = originCfg
	})

	It("keeps request biz id and leaves org fields empty", func() {
		info, err := resolveCMDBInfo(context.Background(), "some-bkci-project", 398)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.BizID).To(Equal("398"))
		Expect(info.Level2BizID).To(BeEmpty())
		Expect(info.ObsProductID).To(BeEmpty())
		Expect(info.ObsProductName).To(BeEmpty())
	})

	It("returns empty biz id when request has none", func() {
		info, err := resolveCMDBInfo(context.Background(), "some-bkci-project", 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.BizID).To(BeEmpty())
	})
})

var _ = Describe("ensureCommunityBCSProject", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("reuses an existing project and does not create", func() {
		client := &fakeBCSProjectClient{
			StubApiClient: &bcs.StubApiClient{},
			existing:      &bcs.Project{ID: "exist-uid", Code: "bkms-ws", Name: "existing"},
		}

		proj, err := ensureCommunityBCSProject(ctx, client, "workspace-name", "bkms-ws", 100)
		Expect(err).NotTo(HaveOccurred())
		Expect(proj.ID).To(Equal("exist-uid"))
		Expect(client.createCalled).To(BeFalse())
	})

	It("creates a k8s project with display name and business id", func() {
		client := &fakeBCSProjectClient{
			StubApiClient: &bcs.StubApiClient{},
			getErr:        errors.New("not found"),
			created: &bcs.Project{
				ID:   "new-uid",
				Code: "bkms-ws",
				Name: "workspace-name",
				Kind: "k8s",
			},
		}

		proj, err := ensureCommunityBCSProject(ctx, client, "workspace-name", "bkms-ws", 100)
		Expect(err).NotTo(HaveOccurred())
		Expect(proj.ID).To(Equal("new-uid"))
		Expect(client.lastCreate).To(Equal(bcs.CreateProjectInput{
			Name:        "workspace-name",
			ProjectCode: "bkms-ws",
			Kind:        "k8s",
			BusinessID:  "100",
		}))
	})

	It("falls back to project code as name and omits business id", func() {
		client := &fakeBCSProjectClient{
			StubApiClient: &bcs.StubApiClient{},
			getErr:        errors.New("not found"),
			created:       &bcs.Project{ID: "new-uid", Code: "bkms-ws"},
		}

		_, err := ensureCommunityBCSProject(ctx, client, "", "bkms-ws", 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(client.lastCreate.Name).To(Equal("bkms-ws"))
		Expect(client.lastCreate.BusinessID).To(BeEmpty())
		Expect(client.lastCreate.Kind).To(Equal("k8s"))
	})
})
