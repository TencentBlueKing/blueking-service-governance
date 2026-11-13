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

package frameworkcfgtmpl_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/frameworkcfgtmpl"
)

const (
	trpcGoBuiltinID  = "builtin-trpc-go"
	trpcCppBuiltinID = "builtin-trpc-cpp"
	tafBuiltinID     = "builtin-taf-cpp"
)

var _ = Describe("Framework config template service", func() {
	var (
		ctx         context.Context
		db          *mongo.Database
		workspaceID string
	)

	create := func(framework, language, name string, isDefault bool) *frameworkcfgtmpl.View {
		content := "global:\n  namespace: Development\n"
		fileName := "trpc_" + language + ".yaml"
		if framework == "taf" {
			content = "<taf>\n  <application>\n    timeout=3000\n  </application>\n</taf>\n"
			fileName = "config.conf"
		}
		view, err := frameworkcfgtmpl.Create(ctx, db, workspaceID, frameworkcfgtmpl.CreateParams{
			Framework: framework,
			Language:  language,
			Definition: frameworkcfgtmpl.Definition{
				Name: name, FileName: fileName, Content: content,
			},
			IsDefault: isDefault,
			Operator:  "admin",
		})
		Expect(err).NotTo(HaveOccurred())
		return view
	}

	defaultIDs := func(group frameworkcfgtmpl.Group) []string {
		views, err := frameworkcfgtmpl.List(ctx, db, workspaceID, group)
		Expect(err).NotTo(HaveOccurred())
		return lo.FilterMap(views, func(v *frameworkcfgtmpl.View, _ int) (string, bool) { return v.ID, v.IsDefault })
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		workspaceID = "ws-frameworkcfgtmpl"
	})

	AfterEach(func() {
		// Clear documents without Drop so migration-managed unique indexes survive.
		Expect(testutil.CleanupMetaColl(frameworkcfgtmpl.TemplateMeta)).To(Succeed())
	})

	Context("List", func() {
		It("returns only builtin templates as defaults for an empty workspace", func() {
			views, err := frameworkcfgtmpl.List(ctx, db, workspaceID, frameworkcfgtmpl.Group{})
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(views, func(v *frameworkcfgtmpl.View, _ int) string { return v.ID })).
				To(Equal([]string{trpcGoBuiltinID, trpcCppBuiltinID, tafBuiltinID}))
			for _, v := range views {
				Expect(v.Builtin).To(BeTrue())
				Expect(v.IsDefault).To(BeTrue())
				Expect(v.Content).To(BeEmpty())
				Expect(v.Name).To(Equal(frameworkcfgtmpl.BuiltinName))
			}
		})

		It("places custom templates after the builtin one of their group and filters by group", func() {
			goTpl := create("trpc", "go", "go-a", false)
			cppTpl := create("trpc", "cpp", "cpp-a", false)
			tafTpl := create("taf", "cpp", "taf-a", false)
			create("trpc", "go", "go-other-ws", false)
			Expect(store.Of(db, frameworkcfgtmpl.TemplateMeta).Update(
				ctx, map[string]any{"workspaceID": "ws-other"}, store.Where("name", "go-other-ws"),
			)).To(Succeed())

			views, err := frameworkcfgtmpl.List(ctx, db, workspaceID, frameworkcfgtmpl.Group{})
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(views, func(v *frameworkcfgtmpl.View, _ int) string { return v.ID })).To(Equal([]string{
				trpcGoBuiltinID, goTpl.ID, trpcCppBuiltinID, cppTpl.ID, tafBuiltinID, tafTpl.ID,
			}))
			Expect(views[1].Content).To(BeEmpty())

			trpcOnly, err := frameworkcfgtmpl.List(ctx, db, workspaceID, frameworkcfgtmpl.Group{Framework: "trpc"})
			Expect(err).NotTo(HaveOccurred())
			Expect(trpcOnly).To(HaveLen(4))

			cppOnly, err := frameworkcfgtmpl.List(ctx, db, workspaceID, frameworkcfgtmpl.Group{Language: "cpp"})
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(cppOnly, func(v *frameworkcfgtmpl.View, _ int) string { return v.ID })).
				To(Equal([]string{trpcCppBuiltinID, cppTpl.ID, tafBuiltinID, tafTpl.ID}))
		})
	})

	Context("Get", func() {
		It("returns builtin content and whether it is the default", func() {
			view, err := frameworkcfgtmpl.Get(ctx, db, workspaceID, trpcGoBuiltinID)
			Expect(err).NotTo(HaveOccurred())
			Expect(view.Content).NotTo(BeEmpty())
			Expect(view.IsDefault).To(BeTrue())

			create("trpc", "go", "go-default", true)
			view, err = frameworkcfgtmpl.Get(ctx, db, workspaceID, trpcGoBuiltinID)
			Expect(err).NotTo(HaveOccurred())
			Expect(view.IsDefault).To(BeFalse())
		})

		It("returns custom templates of the same workspace only", func() {
			created := create("trpc", "go", "go-a", false)
			view, err := frameworkcfgtmpl.Get(ctx, db, workspaceID, created.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(view.Content).To(ContainSubstring("Development"))

			_, err = frameworkcfgtmpl.Get(ctx, db, "ws-other", created.ID)
			Expect(errors.Is(err, frameworkcfgtmpl.ErrTemplateNotFound)).To(BeTrue())
		})

		DescribeTable("treats unknown IDs as not found",
			func(id string) {
				_, err := frameworkcfgtmpl.Get(ctx, db, workspaceID, id)
				Expect(errors.Is(err, frameworkcfgtmpl.ErrTemplateNotFound)).To(BeTrue())
			},
			Entry("unknown builtin", "builtin-trpc-python"),
			Entry("malformed ID", "not-an-object-id"),
			Entry("missing custom", "000000000000000000000000"),
		)
	})

	Context("Create", func() {
		It("creates a custom template in its group", func() {
			view := create("taf", "cpp", "taf-a", false)
			Expect(view.Framework).To(Equal("taf"))
			Expect(view.Language).To(Equal("cpp"))
			Expect(view.Builtin).To(BeFalse())
			Expect(view.Creator).To(Equal("admin"))
		})

		It("rejects duplicate names in the same group but allows them across groups", func() {
			create("trpc", "go", "same", false)
			_, err := frameworkcfgtmpl.Create(ctx, db, workspaceID, frameworkcfgtmpl.CreateParams{
				Framework:  "trpc",
				Language:   "go",
				Definition: frameworkcfgtmpl.Definition{Name: " same ", FileName: "a.yaml", Content: "a: 1"},
			})
			Expect(errors.Is(err, frameworkcfgtmpl.ErrTemplateConflict)).To(BeTrue())

			create("trpc", "cpp", "same", false)
		})

		It("rejects groups without a framework config file and invalid content", func() {
			_, err := frameworkcfgtmpl.Create(ctx, db, workspaceID, frameworkcfgtmpl.CreateParams{
				Framework:  "blank",
				Language:   "go",
				Definition: frameworkcfgtmpl.Definition{Name: "t", FileName: "a.yaml", Content: "a: 1"},
			})
			Expect(errors.Is(err, frameworkcfgtmpl.ErrInvalidTemplate)).To(BeTrue())

			_, err = frameworkcfgtmpl.Create(ctx, db, workspaceID, frameworkcfgtmpl.CreateParams{
				Framework:  "trpc",
				Language:   "go",
				Definition: frameworkcfgtmpl.Definition{Name: "t", FileName: "a.yaml", Content: "a: [1"},
			})
			Expect(errors.Is(err, frameworkcfgtmpl.ErrInvalidTemplate)).To(BeTrue())
		})

		It("moves the default to the new template when requested", func() {
			first := create("trpc", "go", "first", true)
			Expect(defaultIDs(frameworkcfgtmpl.Group{Framework: "trpc", Language: "go"})).To(Equal([]string{first.ID}))

			second := create("trpc", "go", "second", true)
			Expect(second.IsDefault).To(BeTrue())
			Expect(
				defaultIDs(frameworkcfgtmpl.Group{Framework: "trpc", Language: "go"}),
			).To(Equal([]string{second.ID}))
			// 其它分组不受影响
			Expect(defaultIDs(frameworkcfgtmpl.Group{Framework: "trpc", Language: "cpp"})).
				To(Equal([]string{trpcCppBuiltinID}))
		})

		It("keeps the existing default and writes nothing when a new default has a duplicate name", func() {
			group := frameworkcfgtmpl.Group{Framework: "trpc", Language: "go"}
			current := create("trpc", "go", "same", true)

			_, err := frameworkcfgtmpl.Create(ctx, db, workspaceID, frameworkcfgtmpl.CreateParams{
				Framework:  "trpc",
				Language:   "go",
				Definition: frameworkcfgtmpl.Definition{Name: "same", FileName: "a.yaml", Content: "a: 1"},
				IsDefault:  true,
			})
			Expect(errors.Is(err, frameworkcfgtmpl.ErrTemplateConflict)).To(BeTrue())
			Expect(defaultIDs(group)).To(Equal([]string{current.ID}))
			views, err := frameworkcfgtmpl.List(ctx, db, workspaceID, group)
			Expect(err).NotTo(HaveOccurred())
			Expect(views).To(HaveLen(2))
		})
	})

	Context("Update", func() {
		It("updates editable fields and keeps the group", func() {
			created := create("trpc", "go", "old", false)
			before, after, err := frameworkcfgtmpl.Update(
				ctx,
				db,
				workspaceID,
				created.ID,
				frameworkcfgtmpl.UpdateParams{
					Definition: frameworkcfgtmpl.Definition{Name: "new", FileName: "custom.yaml", Content: "b: 2\n"},
					Operator:   "editor",
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(before.Name).To(Equal("old"))
			Expect(after.Name).To(Equal("new"))
			Expect(after.Group).To(Equal(created.Group))
			Expect(after.Updater).To(Equal("editor"))

			stored, err := frameworkcfgtmpl.Get(ctx, db, workspaceID, created.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.FileName).To(Equal("custom.yaml"))
			Expect(stored.Content).To(Equal("b: 2\n"))
			Expect(stored.Creator).To(Equal("admin"))
		})

		It("sets and clears the default through isDefault", func() {
			created := create("trpc", "go", "tpl", false)
			group := frameworkcfgtmpl.Group{Framework: "trpc", Language: "go"}
			def := frameworkcfgtmpl.Definition{Name: "tpl", FileName: "trpc_go.yaml", Content: "a: 1\n"}

			_, after, err := frameworkcfgtmpl.Update(ctx, db, workspaceID, created.ID, frameworkcfgtmpl.UpdateParams{
				Definition: def, IsDefault: lo.ToPtr(true),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(after.IsDefault).To(BeTrue())
			Expect(defaultIDs(group)).To(Equal([]string{created.ID}))

			// 不传 isDefault 时保持默认状态
			_, _, err = frameworkcfgtmpl.Update(
				ctx,
				db,
				workspaceID,
				created.ID,
				frameworkcfgtmpl.UpdateParams{Definition: def},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultIDs(group)).To(Equal([]string{created.ID}))

			_, after, err = frameworkcfgtmpl.Update(ctx, db, workspaceID, created.ID, frameworkcfgtmpl.UpdateParams{
				Definition: def, IsDefault: lo.ToPtr(false),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(after.IsDefault).To(BeFalse())
			Expect(defaultIDs(group)).To(Equal([]string{trpcGoBuiltinID}))
		})

		It("rejects builtin templates and duplicate names", func() {
			_, _, err := frameworkcfgtmpl.Update(ctx, db, workspaceID, trpcGoBuiltinID, frameworkcfgtmpl.UpdateParams{
				Definition: frameworkcfgtmpl.Definition{Name: "x", FileName: "a.yaml", Content: "a: 1"},
			})
			Expect(errors.Is(err, frameworkcfgtmpl.ErrBuiltinReadOnly)).To(BeTrue())

			create("trpc", "go", "taken", false)
			other := create("trpc", "go", "other", false)
			_, _, err = frameworkcfgtmpl.Update(ctx, db, workspaceID, other.ID, frameworkcfgtmpl.UpdateParams{
				Definition: frameworkcfgtmpl.Definition{Name: "taken", FileName: "a.yaml", Content: "a: 1"},
			})
			Expect(errors.Is(err, frameworkcfgtmpl.ErrTemplateConflict)).To(BeTrue())
		})

		It("clears isDefault on the edited template only", func() {
			group := frameworkcfgtmpl.Group{Framework: "trpc", Language: "go"}
			edited := create("trpc", "go", "edited", false)
			other := create("trpc", "go", "other", true)

			_, after, err := frameworkcfgtmpl.Update(ctx, db, workspaceID, edited.ID, frameworkcfgtmpl.UpdateParams{
				Definition: frameworkcfgtmpl.Definition{Name: "edited", FileName: "a.yaml", Content: "a: 1\n"},
				IsDefault:  lo.ToPtr(false),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(after.IsDefault).To(BeFalse())
			Expect(defaultIDs(group)).To(Equal([]string{other.ID}))
		})

		It("writes neither content nor default when setting a default with a duplicate name", func() {
			group := frameworkcfgtmpl.Group{Framework: "trpc", Language: "go"}
			current := create("trpc", "go", "taken", true)
			edited := create("trpc", "go", "edited", false)

			_, _, err := frameworkcfgtmpl.Update(ctx, db, workspaceID, edited.ID, frameworkcfgtmpl.UpdateParams{
				Definition: frameworkcfgtmpl.Definition{Name: "taken", FileName: "a.yaml", Content: "changed: 1\n"},
				IsDefault:  lo.ToPtr(true),
			})
			Expect(errors.Is(err, frameworkcfgtmpl.ErrTemplateConflict)).To(BeTrue())
			Expect(defaultIDs(group)).To(Equal([]string{current.ID}))
			stored, err := frameworkcfgtmpl.Get(ctx, db, workspaceID, edited.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.Name).To(Equal("edited"))
			Expect(stored.Content).NotTo(ContainSubstring("changed"))
		})
	})

	Context("Delete", func() {
		It("falls back to the builtin default after deleting the default template", func() {
			created := create("taf", "cpp", "taf-default", true)
			group := frameworkcfgtmpl.Group{Framework: "taf", Language: "cpp"}
			Expect(defaultIDs(group)).To(Equal([]string{created.ID}))

			deleted, err := frameworkcfgtmpl.Delete(ctx, db, workspaceID, created.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.ID).To(Equal(created.ID))
			Expect(defaultIDs(group)).To(Equal([]string{tafBuiltinID}))
		})

		It("rejects builtin templates and templates of other workspaces", func() {
			_, err := frameworkcfgtmpl.Delete(ctx, db, workspaceID, tafBuiltinID)
			Expect(errors.Is(err, frameworkcfgtmpl.ErrBuiltinReadOnly)).To(BeTrue())

			created := create("trpc", "go", "tpl", false)
			_, err = frameworkcfgtmpl.Delete(ctx, db, "ws-other", created.ID)
			Expect(errors.Is(err, frameworkcfgtmpl.ErrTemplateNotFound)).To(BeTrue())
		})
	})

	Context("SetDefault", func() {
		It("switches the default between custom and builtin templates", func() {
			group := frameworkcfgtmpl.Group{Framework: "trpc", Language: "go"}
			first := create("trpc", "go", "first", false)
			second := create("trpc", "go", "second", false)

			before, after, err := frameworkcfgtmpl.SetDefault(ctx, db, workspaceID, first.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(before.IsDefault).To(BeFalse())
			Expect(after.IsDefault).To(BeTrue())
			Expect(defaultIDs(group)).To(Equal([]string{first.ID}))

			_, _, err = frameworkcfgtmpl.SetDefault(ctx, db, workspaceID, second.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultIDs(group)).To(Equal([]string{second.ID}))

			_, after, err = frameworkcfgtmpl.SetDefault(ctx, db, workspaceID, trpcGoBuiltinID)
			Expect(err).NotTo(HaveOccurred())
			Expect(after.Builtin).To(BeTrue())
			Expect(defaultIDs(group)).To(Equal([]string{trpcGoBuiltinID}))
		})

		It("does not touch updatedAt and is idempotent for the current default", func() {
			created := create("trpc", "go", "tpl", false)
			stored, err := frameworkcfgtmpl.Get(ctx, db, workspaceID, created.ID)
			Expect(err).NotTo(HaveOccurred())

			_, _, err = frameworkcfgtmpl.SetDefault(ctx, db, workspaceID, created.ID)
			Expect(err).NotTo(HaveOccurred())

			reloaded, err := frameworkcfgtmpl.Get(ctx, db, workspaceID, created.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(reloaded.IsDefault).To(BeTrue())
			Expect(reloaded.UpdatedAt).To(BeTemporally("==", stored.UpdatedAt))

			before, after, err := frameworkcfgtmpl.SetDefault(ctx, db, workspaceID, created.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(before.IsDefault).To(BeTrue())
			Expect(after.IsDefault).To(BeTrue())
		})

		It("rejects a second default in the same group at the database level", func() {
			create("trpc", "go", "first", true)
			templates := store.Of(db, frameworkcfgtmpl.TemplateMeta)
			err := templates.Create(ctx, &frameworkcfgtmpl.Template{
				WorkspaceID: workspaceID,
				Group:       frameworkcfgtmpl.Group{Framework: "trpc", Language: "go"},
				Name:        "raw",
				FileName:    "a.yaml",
				Content:     "a: 1",
				IsDefault:   true,
			})
			Expect(errors.Is(err, frameworkcfgtmpl.ErrTemplateConflict)).To(BeTrue())
		})
	})
})
