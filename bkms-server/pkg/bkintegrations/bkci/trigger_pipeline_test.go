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

package bkci

import (
	"context"
	"errors"
	"strings"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

var _ = Describe("build-trigger type helpers", func() {
	DescribeTable("ParseBuildTriggerPipelineType",
		func(pipelineType, wantAppID string, wantOK bool) {
			appID, ok := ParseBuildTriggerPipelineType(pipelineType)
			Expect(ok).To(Equal(wantOK))
			Expect(appID).To(Equal(wantAppID))
		},
		Entry("composite type", "build-trigger-demo-app", "demo-app", true),
		Entry("empty appID after prefix", "build-trigger-", "", false),
	)

	DescribeTable("ResolveBuiltinTemplateType",
		func(pipelineType string, expectedType PipelineType, expectedOK bool) {
			got, ok := ResolveBuiltinTemplateType(pipelineType)
			Expect(ok).To(Equal(expectedOK))
			Expect(got).To(Equal(expectedType))
		},
		Entry("shared builtin", "dockerfile", PipelineTypeDockerfile, true),
		Entry("build-trigger composite", "build-trigger-demo", PipelineTypeBuildTrigger, true),
		Entry("custom pipeline", "p-0123456789abcdef0123456789abcdef", PipelineType(""), false),
	)

	It("should render name and stages without mutating input", func() {
		renderCtx := map[string]any{
			pipelineTmplCtxKeyAppID:        "demo-app",
			pipelineTmplCtxKeyCallbackURL:  "https://bkms.example.com/cb",
			pipelineTmplCtxKeyCredentialID: "bkms_bt_demo",
		}
		name, err := renderBuildTriggerText("[bkms] 自动构建触发（[[ .appID ]]）", renderCtx)
		Expect(err).NotTo(HaveOccurred())
		Expect(name).To(Equal("[bkms] 自动构建触发（demo-app）"))

		stages := []map[string]any{
			{"script": "url=[[ .callbackURL ]] cred=[[ .credentialID ]]"},
		}
		rendered, err := renderBuildTriggerStages(stages, renderCtx)
		Expect(err).NotTo(HaveOccurred())
		Expect(rendered[0]["script"]).To(Equal("url=https://bkms.example.com/cb cred=bkms_bt_demo"))
		Expect(stages[0]["script"]).To(Equal("url=[[ .callbackURL ]] cred=[[ .credentialID ]]"))
	})

	DescribeTable("buildCredentialID",
		func(appID, want string) {
			got := NewTriggerPipelineManager("ws").buildCredentialID(appID)
			Expect(got).To(Equal(want))
			Expect(len(got)).To(BeNumerically("<=", buildTriggerCredentialIDMaxLen))
		},
		Entry("short appID stays readable", "demo-app", "bkms_bt_demo_app"),
		Entry("empty after sanitize falls back to prefix", "---", "bkms_bt"),
	)

	It("should not collide when truncating appIDs that share a long prefix", func() {
		manager := NewTriggerPipelineManager("ws")
		prefix := strings.Repeat("a", buildTriggerCredentialIDMaxLen)
		first := manager.buildCredentialID(prefix + "-one")
		second := manager.buildCredentialID(prefix + "-two")
		Expect(first).NotTo(Equal(second))
		Expect(len(first)).To(Equal(buildTriggerCredentialIDMaxLen))
		Expect(len(second)).To(Equal(buildTriggerCredentialIDMaxLen))
		Expect(first).To(HavePrefix(buildTriggerCredentialIDPrefix))
		Expect(second).To(HavePrefix(buildTriggerCredentialIDPrefix))
	})
})

var _ = Describe("PipelineManager Initialize build-trigger guard", func() {
	var (
		ctx           context.Context
		manager       *PipelineManager
		pipelineStore *PipelineStoreMongo
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = NewPipelineManager(managerTestWorkspaceID)
		_, pipelineStore, _ = setupManagerTestStores()
	})

	It("should return existing build-trigger pipeline and reject creating missing ones", func() {
		pipelineType := string(BuildTriggerPipelineType("demo-app"))
		Expect(pipelineStore.Create(ctx, &Pipeline{
			ID:              "p-bt-existing",
			Type:            pipelineType,
			WorkspaceID:     managerTestWorkspaceID,
			ProjectCode:     managerTestProjectCode,
			Name:            "[bkms] 自动构建触发（demo-app）",
			TemplateVersion: "1.0.0",
			Creator:         "test-user",
		})).To(Succeed())

		got, err := manager.Initialize(ctx, pipelineType)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID).To(Equal("p-bt-existing"))

		_, err = manager.Initialize(ctx, string(BuildTriggerPipelineType("missing-app")))
		Expect(errors.Is(err, ErrBuildTriggerPipelineRequireEnsure)).To(BeTrue())
	})
})

var _ = Describe("TriggerPipelineManager", func() {
	const testAppID = "demo-app"

	var (
		ctx           context.Context
		manager       *TriggerPipelineManager
		projectStore  *ProjectStoreMongo
		pipelineStore *PipelineStoreMongo
		templateStore *DBPipelineTemplateStore
		prevConfig    *config.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = NewTriggerPipelineManager(managerTestWorkspaceID)
		projectStore, pipelineStore, templateStore = setupManagerTestStores()
		Expect(projectStore.Create(ctx, mockProject())).To(Succeed())
		Expect(templateStore.Upsert(ctx, &PipelineTemplate{
			ID:          "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Type:        string(PipelineTypeBuildTrigger),
			Version:     "1.0.0",
			Name:        "[bkms] 自动构建触发（[[ .appID ]]）",
			Description: "template",
			Stages: []map[string]any{
				{"script": "curl [[ .callbackURL ]] token=[[ .credentialID ]]"},
			},
		})).To(Succeed())

		prevConfig = config.G
		config.G = &config.Config{
			HTTPServer: config.HTTPServerConfig{
				PublicBaseURL: "https://bkms.example.com",
			},
		}
	})

	AfterEach(func() {
		config.G = prevConfig
	})

	expectNoLocal := func() {
		GinkgoHelper()
		_, err := pipelineStore.GetByWorkspaceAndType(
			ctx, managerTestWorkspaceID, string(BuildTriggerPipelineType(testAppID)),
		)
		Expect(errors.Is(err, ErrPipelineNotFound)).To(BeTrue())
	}

	It("Ensure: require publicBaseURL, create-if-missing, rollback on create fail", func() {
		config.G.HTTPServer.PublicBaseURL = ""
		_, err := manager.Ensure(ctx, testAppID)
		Expect(err).To(MatchError(ContainSubstring("publicBaseURL")))
		config.G.HTTPServer.PublicBaseURL = "https://bkms.example.com/foo"
		_, err = manager.Ensure(ctx, testAppID)
		Expect(err).To(MatchError(ContainSubstring("must not include path")))
		config.G.HTTPServer.PublicBaseURL = "https://bkms.example.com"

		mockey.PatchConvey("ensure", GinkgoT(), func() {
			user := mockUser()
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbkci.New).Return(cloudbkci.NewStub(user), nil).Build()

			first, err := manager.Ensure(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(first.CallbackCredentialID).To(Equal("bkms_bt_demo_app"))
			second, err := manager.Ensure(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(second.ID).To(Equal(first.ID))

			Expect(pipelineStore.Delete(ctx, managerTestWorkspaceID, first.Type)).To(Succeed())
			mockey.Mock((*cloudbkci.StubApiClient).CreatePipeline).Return("", errors.New("create failed")).Build()
			_, err = manager.Ensure(ctx, testAppID)
			Expect(err).To(MatchError(ContainSubstring("create failed")))
			expectNoLocal()
		})
	})

	It("Ensure: rollback remote pipeline when local insert fails after CreatePipeline", func() {
		mockey.PatchConvey("rollback remote pipeline", GinkgoT(), func() {
			user := mockUser()
			const remotePipelineID = "p-bt-rollback-after-db-fail"
			var deletedPipelineID string

			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbkci.New).Return(cloudbkci.NewStub(user), nil).Build()
			mockey.Mock((*cloudbkci.StubApiClient).CreatePipeline).
				Return(remotePipelineID, nil).
				Build()
			mockey.Mock((*PipelineStoreMongo).Create).Return(errors.New("mongo insert failed")).Build()
			mockey.Mock((*cloudbkci.StubApiClient).DeletePipeline).To(
				func(_ context.Context, _, pipelineID string) error {
					deletedPipelineID = pipelineID
					return nil
				},
			).Build()

			_, err := manager.Ensure(ctx, testAppID)
			Expect(err).To(MatchError(ContainSubstring("mongo insert failed")))
			Expect(deletedPipelineID).To(Equal(remotePipelineID))
			expectNoLocal()
		})
	})

	It("Cleanup: clear local when credential or remote pipeline delete is soft-failed", func() {
		mockey.PatchConvey("cleanup", GinkgoT(), func() {
			user := mockUser()
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbkci.New).Return(cloudbkci.NewStub(user), nil).Build()

			_, err := manager.Ensure(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			mockey.Mock((*cloudbkci.StubApiClient).DeleteCredential).
				Return(errors.New("credential delete failed")).
				Build()
			Expect(manager.Cleanup(ctx, testAppID)).To(Succeed())
			expectNoLocal()

			Expect(pipelineStore.Create(ctx, &Pipeline{
				ID:                   "p-bt-orphan",
				Type:                 string(BuildTriggerPipelineType(testAppID)),
				WorkspaceID:          managerTestWorkspaceID,
				ProjectCode:          managerTestProjectCode,
				CallbackCredentialID: "bkms_bt_demo_app",
				Creator:              "test-user",
			})).To(Succeed())
			mockey.Mock((*cloudbkci.StubApiClient).DeletePipeline).Return(cloudbkci.ObjectNotFound).Build()
			Expect(manager.Cleanup(ctx, testAppID)).To(Succeed())
			expectNoLocal()
		})
	})
})
