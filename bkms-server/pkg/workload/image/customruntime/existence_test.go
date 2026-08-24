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

package customruntime

import (
	"context"
	"net/http"

	"github.com/bytedance/mockey"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	infrasreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

var _ = Describe("ExistenceChecker", func() {
	const (
		workspaceID  = "ws-demo"
		imageName    = "docker.bkrepo.example.com/demo/repo/my-golang"
		imageRef     = imageName + ":1.0"
		registryAddr = "docker.bkrepo.example.com/demo/repo"
	)

	var checker *ExistenceChecker

	BeforeEach(func() {
		checker = NewExistenceChecker(&snapshot.Service{})
	})

	AfterEach(func() {
		mockey.UnPatchAll()
	})

	mockBoundRegistry := func() {
		mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(
			&bkmsreg.ImageRegistry{Registry: registryAddr}, nil,
		).Build()
	}

	Describe("MatchesWorkspaceRegistry", func() {
		It("returns true when the image name falls under the workspace registry path", func() {
			mockBoundRegistry()

			matches, err := checker.MatchesWorkspaceRegistry(context.Background(), workspaceID, imageName)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})

		It("returns false when the workspace has no image registry", func() {
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(
				nil, bkmsreg.ErrImageRegistryNotFound,
			).Build()

			matches, err := checker.MatchesWorkspaceRegistry(context.Background(), workspaceID, imageName)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
		})
	})

	Describe("ValidateTaggedReference", func() {
		var (
			headCalls int
			listCalls int
			headErr   error
			listTags  []string
			listErr   error
			// 记录两次远端调用拿到的 context 是否带 deadline，用于确认超时预算传导到底
			headCtxHasDeadline bool
			listCtxHasDeadline bool
		)

		BeforeEach(func() {
			headCalls, listCalls = 0, 0
			headErr, listErr = nil, nil
			listTags = nil
			headCtxHasDeadline, listCtxHasDeadline = false, false

			mockey.Mock((*snapshot.Service).ResolveRepoKeyForWorkspace).Return(
				&snapshot.RepoKeyInfo{
					RepoName: imageName,
					Username: "user",
					Password: "pass",
				},
				nil,
			).Build()
			mockey.Mock(infrasreg.New).Return(&infrasreg.Client{}).Build()
			mockey.Mock((*infrasreg.Client).HeadManifest).To(
				func(_ *infrasreg.Client, gotCtx context.Context, _, _ string) error {
					headCalls++
					_, headCtxHasDeadline = gotCtx.Deadline()
					return headErr
				},
			).Build()
			mockey.Mock((*infrasreg.Client).ListAllTags).To(
				func(_ *infrasreg.Client, gotCtx context.Context, _ string) ([]string, error) {
					listCalls++
					_, listCtxHasDeadline = gotCtx.Deadline()
					return listTags, listErr
				},
			).Build()
		})

		It("only sends HEAD when the tag exists", func() {
			Expect(checker.ValidateTaggedReference(context.Background(), workspaceID, imageRef)).To(Succeed())
			Expect(headCalls).To(Equal(1))
			Expect(listCalls).To(Equal(0))
		})

		It("classifies a 404 after a successful tag list as tag not found", func() {
			headErr = &transport.Error{StatusCode: http.StatusNotFound}
			listTags = []string{"2.0"}

			err := checker.ValidateTaggedReference(context.Background(), workspaceID, imageRef)
			Expect(err).To(MatchError(ErrImageTagNotFound))
			Expect(listCalls).To(Equal(1))
		})

		It("classifies a 404 after a 404 tag list as image name not found", func() {
			headErr = &transport.Error{StatusCode: http.StatusNotFound}
			listErr = &transport.Error{StatusCode: http.StatusNotFound}

			err := checker.ValidateTaggedReference(context.Background(), workspaceID, imageRef)
			Expect(err).To(MatchError(ErrImageNameNotFound))
		})

		It("classifies an unauthorized tag list after a 404 head as registry access denied", func() {
			headErr = &transport.Error{StatusCode: http.StatusNotFound}
			listErr = &transport.Error{StatusCode: http.StatusUnauthorized}

			err := checker.ValidateTaggedReference(context.Background(), workspaceID, imageRef)
			Expect(err).To(MatchError(ErrRegistryAccessDenied))
			Expect(listCalls).To(Equal(1))
		})

		It("classifies an unexpected tag list failure after a 404 head as registry access failed", func() {
			headErr = &transport.Error{StatusCode: http.StatusNotFound}
			listErr = errors.New("connection reset")

			err := checker.ValidateTaggedReference(context.Background(), workspaceID, imageRef)
			Expect(err).To(MatchError(ErrRegistryAccessFailed))
			Expect(listCalls).To(Equal(1))
		})

		It("classifies unauthorized HEAD as registry access denied", func() {
			headErr = &transport.Error{StatusCode: http.StatusUnauthorized}

			err := checker.ValidateTaggedReference(context.Background(), workspaceID, imageRef)
			Expect(err).To(MatchError(ErrRegistryAccessDenied))
			Expect(listCalls).To(Equal(0))
		})

		It("classifies unexpected HEAD errors as registry access failed", func() {
			headErr = errors.New("timeout")

			err := checker.ValidateTaggedReference(context.Background(), workspaceID, imageRef)
			Expect(err).To(MatchError(ErrRegistryAccessFailed))
			Expect(listCalls).To(Equal(0))
		})

		It("passes a context with the check timeout down to both registry calls", func() {
			// 同步路径上的探测必须可中断，HEAD 与回退的 tag 列表都要拿到带 deadline 的 context
			headErr = &transport.Error{StatusCode: http.StatusNotFound}
			listTags = []string{"2.0"}

			err := checker.ValidateTaggedReference(context.Background(), workspaceID, imageRef)
			Expect(err).To(MatchError(ErrImageTagNotFound))
			Expect(headCtxHasDeadline).To(BeTrue())
			Expect(listCtxHasDeadline).To(BeTrue())
		})
	})
})

var _ = Describe("nameBelongsToRegistry", func() {
	const registryAddr = "docker.bkrepo.example.com/demo/repo"

	DescribeTable("path boundary",
		func(imageName string, expected bool) {
			Expect(nameBelongsToRegistry(imageName, registryAddr)).To(Equal(expected))
		},
		Entry("matches a repository under the registry path",
			"docker.bkrepo.example.com/demo/repo/my-golang", true),
		Entry("matches a nested repository under the registry path",
			"docker.bkrepo.example.com/demo/repo/team/my-golang", true),
		Entry("does not match a sibling path that only shares a prefix substring",
			"docker.bkrepo.example.com/demo/repo-evil/my-golang", false),
		Entry("does not match the registry address itself without a repository name",
			"docker.bkrepo.example.com/demo/repo", false),
		Entry("does not match an official image name",
			"golang", false),
		Entry("does not match a blank image name",
			"", false),
		Entry("matches regardless of registry host letter case",
			"DOCKER.BKRepo.example.com/demo/repo/my-golang", true),
	)

	It("trims trailing slash on the registry address before matching", func() {
		Expect(nameBelongsToRegistry(
			"docker.bkrepo.example.com/demo/repo/my-golang",
			registryAddr+"/",
		)).To(BeTrue())
	})

	DescribeTable("registry address with a scheme",
		func(addr string) {
			// 镜像源地址由用户填写且未强制格式，带 scheme 时不能把自定义镜像误判成官方镜像
			Expect(nameBelongsToRegistry(
				"docker.bkrepo.example.com/demo/repo/my-golang", addr,
			)).To(BeTrue())
		},
		Entry("https", "https://"+registryAddr),
		Entry("http", "http://"+registryAddr),
		Entry("mixed case https", "HTTPS://DOCKER.BKRepo.example.com/demo/repo"),
	)
})
