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

package build

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/semver"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/revision"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

var _ = Describe("ChartBuildService revision check", func() {
	It("rejects empty revision before touching any store", func() {
		// 零值服务的 db 是 nil：只要校验晚于准备或版本号生成就会 panic，
		// 因此能返回哨兵错误即证明语义化版本号没有被消耗
		svc := &ChartBuildService{}
		app := &bkmsapp.Application{ID: "helm-app", WorkspaceID: "ws-1", Type: bkmsapp.AppTypeHelm}

		_, err := svc.ExecuteChartBuild(
			context.Background(), app, "patch", revision.CodeRevision{Type: revision.TypeTag},
		)

		Expect(errors.Is(err, revision.ErrRequired)).To(BeTrue())
	})
})

var _ = Describe("buildPipelineParams", func() {
	app := &bkmsapp.Application{
		ID:   "helm-app-" + stringx.Random(8),
		Name: "demo-chart",
		HelmSpec: &bkmsapp.HelmSpec{
			HelmSource: &bkmsapp.HelmSource{
				GitRepoConfig: &bkmsapp.GitRepoConfig{
					RepoAlias: "demo/repo",
					SourceDir: "charts/demo",
				},
			},
		},
	}
	var svc *ChartBuildService

	BeforeEach(func() {
		svc = &ChartBuildService{db: testutil.NewDB(GinkgoT())}
	})
	AfterEach(func() {
		_, err := svc.db.Collection(semver.CounterMeta.Collection).
			DeleteMany(context.Background(), bson.M{"_id": app.ID})
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should map revision type to checkout params",
		func(rev revision.CodeRevision, checkoutBy string) {
			params, chartVersion, err := svc.buildPipelineParams(
				context.Background(), app, "patch", rev, "https://helm.example.com",
			)

			Expect(err).NotTo(HaveOccurred())
			// 每个用例结束后清理计数器，版本号总是从 0.0.1 开始
			Expect(chartVersion).To(Equal("0.0.1"))
			Expect(params[pipelineparam.RepoCheckoutBy]).To(Equal(checkoutBy))
			Expect(params[pipelineparam.RepoRevision]).To(Equal(rev.Value))
			Expect(params[pipelineparam.RepoAlias]).To(Equal("demo/repo"))
			Expect(params[pipelineparam.HelmChartBuildDir]).To(Equal("charts/demo"))
		},
		Entry("branch", revision.CodeRevision{Type: revision.TypeBranch, Value: "main"}, "BRANCH"),
		Entry("tag", revision.CodeRevision{Type: revision.TypeTag, Value: "v1.0.0"}, "TAG"),
		Entry("commit", revision.CodeRevision{Type: revision.TypeCommit, Value: "a1b2c3d"}, "COMMIT_ID"),
	)
})
