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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/tof"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const (
	managerTestWorkspaceID = "test-workspace"
	managerTestProjectCode = "test-project"
)

// mockUser creates a mock user for testing
func mockUser() auth.User {
	return auth.User{
		ID:   "test-user",
		Cred: auth.UserCredential{AccessToken: "test-token"},
	}
}

// mockProject creates a mock bkci project
func mockProject() *Project {
	return &Project{
		ID:          "test-project-id",
		Code:        managerTestProjectCode,
		WorkspaceID: managerTestWorkspaceID,
		Creator:     "test-user",
	}
}

func setupManagerTestStores() (*ProjectStore, *PipelineStore, *PipelineTemplateStore) {
	Expect(testutil.CleanupMetaColl(ProjectMeta)).To(Succeed())
	Expect(testutil.CleanupMetaColl(PipelineMeta)).To(Succeed())
	Expect(testutil.CleanupMetaColl(PipelineTemplateMeta)).To(Succeed())

	return testutil.NewStore(GinkgoT(), ProjectMeta),
		testutil.NewStore(GinkgoT(), PipelineMeta),
		testutil.NewStore(GinkgoT(), PipelineTemplateMeta)
}

func createManagerTestPipeline(ctx context.Context, pipelineStore *PipelineStore, templateVersion string) *Pipeline {
	pipeline := &Pipeline{
		ID:              "p-builtin-pipeline-id",
		Type:            string(PipelineTypeDockerfile),
		WorkspaceID:     managerTestWorkspaceID,
		ProjectCode:     managerTestProjectCode,
		Name:            "old pipeline",
		Description:     "old description",
		TemplateVersion: templateVersion,
		Creator:         "test-user",
	}
	Expect(pipelineStore.Create(ctx, pipeline)).To(Succeed())
	return pipeline
}

func createManagerTestTemplate(
	ctx context.Context,
	tmplStore *PipelineTemplateStore,
	version string,
) *PipelineTemplate {
	tmpl := &PipelineTemplate{
		ID:          "0348a1df-44de-29b2-1c94-2d42841c009d",
		Type:        string(PipelineTypeDockerfile),
		Version:     version,
		Name:        "updated pipeline",
		Description: "updated description",
		Stages: []map[string]any{
			{
				"id":   "stage-1",
				"name": "stage-1",
			},
		},
	}
	_, err := tmplStore.Upsert(ctx, tmpl, ByType(tmpl.Type))
	Expect(err).NotTo(HaveOccurred())
	return tmpl
}

var _ = Describe("NewProjectManager", func() {
	It("should create valid ProjectManager instance", func() {
		workspaceID := "test-workspace"
		manager := NewProjectManager(testutil.NewDB(GinkgoT()), workspaceID)
		Expect(manager).NotTo(BeNil())
		Expect(manager.workspaceID).To(Equal(workspaceID))
	})
})

var _ = Describe("DefaultProjectCode", func() {
	It("should return code in format 'bkms-{workspaceID}'", func() {
		Expect(DefaultProjectCode("test-workspace")).To(Equal("bkms-test-workspace"))
	})
})

var _ = Describe("ProjectManager Initialize", func() {
	var (
		ctx          context.Context
		manager      *ProjectManager
		projectStore *ProjectStore
		// 多租户模式（测试可通过环境变量开启）下入库的是带租户前缀的蓝盾项目 Code
		expectedCode string
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		manager = NewProjectManager(testutil.NewDB(GinkgoT()), managerTestWorkspaceID)
		projectStore, _, _ = setupManagerTestStores()
		expectedCode = bkciProjectCode(ctx, managerTestProjectCode)
	})

	Context("Success Scenarios", func() {
		It("should return existing project when workspace already has it", func() {
			existingProj := mockProject()
			existingProj.Code = expectedCode
			Expect(projectStore.Create(ctx, existingProj)).To(Succeed())

			proj, err := manager.Initialize(ctx, managerTestProjectCode, "test-obs-id", "test-obs-name")

			Expect(err).NotTo(HaveOccurred())
			Expect(proj.Code).To(Equal(expectedCode))
		})

		It("should create new project successfully", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
				mockey.Mock(tof.GetUserOrganization).Return(&tof.Organization{
					BgID:      "1",
					BgName:    "Test BG",
					DeptID:    "100",
					DeptName:  "Test Dept",
					GroupID:   "10",
					GroupName: "Test Group",
				}, nil).Build()

				proj, err := manager.Initialize(ctx, managerTestProjectCode, "test-obs-id", "test-obs-name")

				Expect(err).NotTo(HaveOccurred())
				Expect(proj.Code).To(Equal(expectedCode))
				Expect(proj.WorkspaceID).To(Equal(managerTestWorkspaceID))

				stored, err := projectStore.Get(ctx, ByWorkspace(managerTestWorkspaceID))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.Code).To(Equal(expectedCode))
			})
		})

		It("should use default project code when not specified", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
				mockey.Mock(tof.GetUserOrganization).Return(&tof.Organization{}, nil).Build()

				proj, err := manager.Initialize(ctx, "", "test-obs-id", "test-obs-name")

				Expect(err).NotTo(HaveOccurred())
				Expect(proj.Code).To(Equal(bkciProjectCode(ctx, "bkms-test-workspace")))
			})
		})

		// stub 客户端的 GetProject 总是成功，Initialize 走不到创建分支，因此直接测 createBKCIProject
		It("should create bkci project without querying tof when user organization is skipped", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
				orgMocker := mockey.Mock(tof.GetUserOrganization).
					Return(nil, errors.New("tof api not found")).Build()

				proj, err := manager.SkipUserOrganization().createBKCIProject(ctx, expectedCode, "", "")

				Expect(err).NotTo(HaveOccurred())
				Expect(proj.Code).To(Equal(expectedCode))
				Expect(orgMocker.Times()).To(BeZero())
			})
		})
	})

	Context("Error Scenarios", func() {
		It("should fail creating bkci project when querying user organization fails by default", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
				mockey.Mock(tof.GetUserOrganization).Return(nil, errors.New("tof api not found")).Build()

				proj, err := manager.createBKCIProject(ctx, expectedCode, "", "")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("organization"))
				Expect(proj).To(BeNil())
			})
		})

		It("should fail when workspace has different project", func() {
			existingProj := mockProject()
			existingProj.Code = "another-project"
			Expect(projectStore.Create(ctx, existingProj)).To(Succeed())

			proj, err := manager.Initialize(ctx, managerTestProjectCode, "test-obs-id", "test-obs-name")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already has project"))
			Expect(proj).To(BeNil())
		})

		It("should fail when project code used by another workspace", func() {
			existingProj := mockProject()
			existingProj.Code = expectedCode
			existingProj.WorkspaceID = "another-workspace"
			Expect(projectStore.Create(ctx, existingProj)).To(Succeed())

			proj, err := manager.Initialize(ctx, managerTestProjectCode, "test-obs-id", "test-obs-name")

			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrProjectCodeAlreadyUsed)).To(BeTrue())
			Expect(proj).To(BeNil())
		})

		// 测的是：蓝盾项目 Code 全局唯一，其他租户已绑定的项目也算占用，在写库前就以
		// ErrProjectCodeAlreadyUsed 拒绝（而不是撞唯一索引），且报错里不暴露对方的工作空间
		It("should fail when project code used by a workspace of another tenant", func() {
			existingProj := mockProject()
			existingProj.Code = expectedCode
			existingProj.WorkspaceID = "another-tenant-workspace"
			Expect(projectStore.Create(ctx, existingProj, store.TenantOf("another-tenant"))).To(Succeed())

			proj, err := manager.Initialize(ctx, managerTestProjectCode, "test-obs-id", "test-obs-name")

			Expect(errors.Is(err, ErrProjectCodeAlreadyUsed)).To(BeTrue())
			Expect(err.Error()).NotTo(ContainSubstring("another-tenant-workspace"))
			Expect(proj).To(BeNil())
		})
	})
})

var _ = Describe("bkci project code tenant prefix", func() {
	var ctx context.Context

	setMultiTenantMode := func(enabled bool) {
		origin := config.G.Tenant.EnableMultiTenantMode
		config.G.Tenant.EnableMultiTenantMode = enabled
		DeferCleanup(func() { config.G.Tenant.EnableMultiTenantMode = origin })
	}

	BeforeEach(func() {
		ctx = tenant.WithTenantID(context.Background(), "system")
	})

	It("adds tenant prefix in multi-tenant mode", func() {
		setMultiTenantMode(true)
		Expect(bkciProjectCode(ctx, "bkms-nicole-1")).To(Equal("system.bkms-nicole-1"))
		Expect(trimBKCIProjectTenantPrefix(ctx, "system.bkms-nicole-1")).To(Equal("bkms-nicole-1"))
	})

	It("keeps code that already has tenant prefix", func() {
		setMultiTenantMode(true)
		Expect(bkciProjectCode(ctx, "system.bkms-nicole-1")).To(Equal("system.bkms-nicole-1"))
	})

	It("keeps code unchanged in single-tenant mode", func() {
		setMultiTenantMode(false)
		Expect(bkciProjectCode(ctx, "bkms-nicole-1")).To(Equal("bkms-nicole-1"))
		Expect(trimBKCIProjectTenantPrefix(ctx, "bkms-nicole-1")).To(Equal("bkms-nicole-1"))
	})

	It("creates bkci project with code without tenant prefix", func() {
		setMultiTenantMode(true)
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
			var createdName string
			mockey.Mock((*cloudbkci.StubApiClient).CreateProject).To(
				func(_ *cloudbkci.StubApiClient, _ context.Context, code, _, _ string, _ *tof.Organization) error {
					createdName = code
					return nil
				},
			).Build()
			manager := NewProjectManager(testutil.NewDB(GinkgoT()), managerTestWorkspaceID).SkipUserOrganization()

			proj, err := manager.createBKCIProject(ctx, "system.bkms-nicole-1", "", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(createdName).To(Equal("bkms-nicole-1"))
			Expect(proj.Code).To(Equal("system.bkms-nicole-1"))
		})
	})
})

var _ = Describe("shouldUpdateBuiltinPipelineFromTemplate", func() {
	DescribeTable("should decide whether to update by semver versions",
		func(currentVersion, templateVersion string, expected, expectErr bool) {
			actual, err := shouldUpdateBuiltinPipelineFromTemplate(currentVersion, templateVersion)

			Expect(actual).To(Equal(expected))
			Expect(err != nil).To(Equal(expectErr))
		},
		Entry("current version is lower than template version", "1.0.0", "1.1.0", true, false),
		Entry("current version equals template version", "1.1.0", "1.1.0", false, false),
		Entry("current version is greater than template version", "1.2.0", "1.1.0", false, false),
		Entry("current version is invalid", "not-a-semver", "1.0.0", true, false),
		Entry("current version is empty", "", "1.0.0", true, false),
		Entry("template version is invalid", "1.0.0", "not-a-semver", false, true),
	)
})

var _ = Describe("PipelineManager Initialize", func() {
	var (
		ctx           context.Context
		manager       *PipelineManager
		pipelineStore *PipelineStore
		templateStore *PipelineTemplateStore
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		manager = NewPipelineManager(testutil.NewDB(GinkgoT()), managerTestWorkspaceID)
		_, pipelineStore, templateStore = setupManagerTestStores()
	})

	It("should return custom pipeline directly without template version check", func() {
		pipeline := &Pipeline{
			ID:          "p-custom-pipeline-id",
			Type:        "p-custom-pipeline-id",
			WorkspaceID: managerTestWorkspaceID,
			ProjectCode: managerTestProjectCode,
			Creator:     "test-user",
		}
		Expect(pipelineStore.Create(ctx, pipeline)).To(Succeed())

		result, err := manager.Initialize(ctx, pipeline.Type)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.ID).To(Equal(pipeline.ID))
		Expect(result.Type).To(Equal(pipeline.Type))
	})

	It("should update builtin pipeline when template version changed", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
			createManagerTestPipeline(ctx, pipelineStore, "1.0.0")
			createManagerTestTemplate(ctx, templateStore, "1.1.0")

			result, err := manager.Initialize(ctx, string(PipelineTypeDockerfile))

			Expect(err).NotTo(HaveOccurred())
			Expect(result.TemplateVersion).To(Equal("1.1.0"))
			Expect(result.Name).To(Equal("updated pipeline"))
			Expect(result.Description).To(Equal("updated description"))

			stored, err := pipelineStore.Get(
				ctx, ByWorkspaceAndType(managerTestWorkspaceID, string(PipelineTypeDockerfile)),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.TemplateVersion).To(Equal("1.1.0"))
			Expect(stored.Name).To(Equal("updated pipeline"))
		})
	})
})
