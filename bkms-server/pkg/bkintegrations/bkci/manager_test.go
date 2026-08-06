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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/tof"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
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

func setupManagerTestStores() (*ProjectStoreMongo, *PipelineStoreMongo, *DBPipelineTemplateStore) {
	Expect(testutil.CleanupCollection(projectCollName)).To(Succeed())
	Expect(testutil.CleanupCollection(pipelineCollName)).To(Succeed())
	Expect(testutil.CleanupCollection(pipelineTemplateCollName)).To(Succeed())

	projectStore, err := NewProjectStoreMongo(database.Client(), database.Name())
	Expect(err).NotTo(HaveOccurred())
	pipelineStore, err := NewPipelineStoreMongo(database.Client(), database.Name())
	Expect(err).NotTo(HaveOccurred())
	templateStore, err := NewDBPipelineTemplateStore(database.Client(), database.Name())
	Expect(err).NotTo(HaveOccurred())

	return projectStore, pipelineStore, templateStore
}

func createManagerTestPipeline(ctx context.Context, store *PipelineStoreMongo, templateVersion string) *Pipeline {
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
	Expect(store.Create(ctx, pipeline)).To(Succeed())
	return pipeline
}

func createManagerTestTemplate(ctx context.Context, store *DBPipelineTemplateStore, version string) *PipelineTemplate {
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
	Expect(store.Upsert(ctx, tmpl)).To(Succeed())
	return tmpl
}

var _ = Describe("NewProjectManager", func() {
	It("should create valid ProjectManager instance", func() {
		workspaceID := "test-workspace"
		manager := NewProjectManager(workspaceID)
		Expect(manager).NotTo(BeNil())
		Expect(manager.workspaceID).To(Equal(workspaceID))
	})
})

var _ = Describe("genDefaultProjectCode", func() {
	It("should return code in format 'bkms-{workspaceID}'", func() {
		manager := NewProjectManager("test-workspace")
		code := manager.genDefaultProjectCode("test-workspace")
		Expect(code).To(Equal("bkms-test-workspace"))
	})
})

var _ = Describe("ProjectManager Initialize", func() {
	var (
		ctx          context.Context
		manager      *ProjectManager
		projectStore *ProjectStoreMongo
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = NewProjectManager(managerTestWorkspaceID)
		projectStore, _, _ = setupManagerTestStores()
	})

	Context("Success Scenarios", func() {
		It("should return existing project when workspace already has it", func() {
			existingProj := mockProject()
			Expect(projectStore.Create(ctx, existingProj)).To(Succeed())

			proj, err := manager.Initialize(ctx, managerTestProjectCode, "test-obs-id", "test-obs-name")

			Expect(err).NotTo(HaveOccurred())
			Expect(proj.Code).To(Equal(managerTestProjectCode))
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
				Expect(proj.Code).To(Equal(managerTestProjectCode))
				Expect(proj.WorkspaceID).To(Equal(managerTestWorkspaceID))

				stored, err := projectStore.GetByWorkspace(ctx, managerTestWorkspaceID)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.Code).To(Equal(managerTestProjectCode))
			})
		})

		It("should use default project code when not specified", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
				mockey.Mock(tof.GetUserOrganization).Return(&tof.Organization{}, nil).Build()

				proj, err := manager.Initialize(ctx, "", "test-obs-id", "test-obs-name")

				Expect(err).NotTo(HaveOccurred())
				Expect(proj.Code).To(Equal("bkms-test-workspace"))
			})
		})
	})

	Context("Error Scenarios", func() {
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
			existingProj.WorkspaceID = "another-workspace"
			Expect(projectStore.Create(ctx, existingProj)).To(Succeed())

			proj, err := manager.Initialize(ctx, managerTestProjectCode, "test-obs-id", "test-obs-name")

			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrProjectCodeAlreadyUsed)).To(BeTrue())
			Expect(proj).To(BeNil())
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
		pipelineStore *PipelineStoreMongo
		templateStore *DBPipelineTemplateStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = NewPipelineManager(managerTestWorkspaceID)
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

			stored, err := pipelineStore.GetByWorkspaceAndType(
				ctx,
				managerTestWorkspaceID,
				string(PipelineTypeDockerfile),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.TemplateVersion).To(Equal("1.1.0"))
			Expect(stored.Name).To(Equal("updated pipeline"))
		})
	})
})
