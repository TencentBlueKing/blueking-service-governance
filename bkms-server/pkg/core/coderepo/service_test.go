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

package coderepo_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const testWorkspaceID = "coderepo-ws"

var _ = Describe("Save", func() {
	var (
		ctx  context.Context
		db   *mongo.Database
		repo *testutil.GitRepo
	)

	gitSpec := func(username, password string) coderepo.SaveSpec {
		return coderepo.SaveSpec{
			WorkspaceID: testWorkspaceID, Type: testutil.BasicRepoSourceTypeName, URL: repo.URL, Username: username, Password: password,
		}
	}
	rawDoc := func(alias string) bson.Raw {
		GinkgoHelper()
		var doc bson.Raw
		Expect(db.Collection(coderepo.CodeRepositoryMeta.Collection).
			FindOne(ctx, bson.M{"workspaceID": testWorkspaceID, "alias": alias}).Decode(&doc)).To(Succeed())
		return doc
	}

	BeforeEach(func() {
		useSourceTypes(tgitSourceType, gitSourceType)
		ctx = auth.WithUser(testutil.NewDefaultTenantContext(), auth.User{ID: "creator"})
		db = testutil.NewDB(GinkgoT())
		Expect(testutil.CleanupMetaColl(coderepo.CodeRepositoryMeta)).To(Succeed())
		repo = testutil.NewGitRepo(GinkgoT())
	})

	Context("basic source type", func() {
		It("verifies the credential and stores the password encrypted", func() {
			saved, err := coderepo.Save(ctx, db, gitSpec(testutil.GitServerUsername(), testutil.GitServerPassword()))
			Expect(err).NotTo(HaveOccurred())
			Expect(
				saved.Alias,
			).To(Equal(strings.ToLower(testutil.BasicRepoSourceTypeName) + ":" + testutil.GitServerUsername() + "/" + repo.Name))
			Expect(saved.SourceType).To(Equal(testutil.BasicRepoSourceTypeName))
			Expect(saved.Creator).To(Equal("creator"))
			Expect(saved.BasicAuth).To(Equal(&coderepo.BasicAuth{
				Username: testutil.GitServerUsername(), Password: testutil.GitServerPassword(),
			}))

			stored := rawDoc(saved.Alias).Lookup("basicAuth", "password").StringValue()
			Expect(stored).NotTo(BeEmpty())
			Expect(stored).NotTo(Equal(testutil.GitServerPassword()))

			got, err := coderepo.Get(ctx, db, testWorkspaceID, saved.Alias)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.BasicAuth.Password).To(Equal(testutil.GitServerPassword()))
		})

		It("rejects a wrong password without saving", func() {
			_, err := coderepo.Save(ctx, db, gitSpec(testutil.GitServerUsername(), "wrong"))
			Expect(err).To(MatchError(coderepo.ErrCredentialInvalid))
			n, err := store.Of(db, coderepo.CodeRepositoryMeta).Count(ctx, coderepo.ByWorkspace(testWorkspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeZero())
		})

		It("requires the credential when associating for the first time", func() {
			_, err := coderepo.Save(ctx, db, gitSpec(testutil.GitServerUsername(), ""))
			Expect(err).To(MatchError(coderepo.ErrCredentialRequired))
		})

		Context("when the repository is associated", func() {
			var alias string

			BeforeEach(func() {
				saved, err := coderepo.Save(
					ctx,
					db,
					gitSpec(testutil.GitServerUsername(), testutil.GitServerPassword()),
				)
				Expect(err).NotTo(HaveOccurred())
				alias = saved.Alias
			})

			It("keeps the saved password when the password is omitted", func() {
				saved, err := coderepo.Save(ctx, db, gitSpec("", ""))
				Expect(err).NotTo(HaveOccurred())
				Expect(saved.BasicAuth.Password).To(Equal(testutil.GitServerPassword()))
			})

			It("requires the password when the username changes", func() {
				_, err := coderepo.Save(ctx, db, gitSpec("another", ""))
				Expect(err).To(MatchError(coderepo.ErrPasswordRequired))
			})

			It("verifies a new password before updating", func() {
				_, err := coderepo.Save(ctx, db, gitSpec(testutil.GitServerUsername(), "wrong"))
				Expect(err).To(MatchError(coderepo.ErrCredentialInvalid))
				got, err := coderepo.Get(ctx, db, testWorkspaceID, alias)
				Expect(err).NotTo(HaveOccurred())
				Expect(got.BasicAuth.Password).To(Equal(testutil.GitServerPassword()))
			})

			It("lets Probe reuse the saved password", func() {
				controller, err := coderepo.Probe(ctx, db, gitSpec("", ""))
				Expect(err).NotTo(HaveOccurred())
				Expect(controller.Touch(ctx)).To(Succeed())
			})
		})
	})

	Context("bkci managed source type", func() {
		It("saves the repository without a credential", func() {
			saved, err := coderepo.Save(ctx, db, coderepo.SaveSpec{
				WorkspaceID: testWorkspaceID, Type: config.DefaultRepoSourceTypeName, URL: "https://git.example.com/group/app.git",
				Alias: "group/app", Username: "ignored", Password: "ignored",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.SourceType).To(Equal(config.DefaultRepoSourceTypeName))
			Expect(saved.BasicAuth).To(BeNil())
			_, lookupErr := rawDoc("group/app").LookupErr("basicAuth")
			Expect(lookupErr).To(HaveOccurred())
		})

		It("treats the legacy GitHub type as the default source type", func() {
			saved, err := coderepo.Save(ctx, db, coderepo.SaveSpec{
				WorkspaceID: testWorkspaceID, Type: coderepo.LegacyTypeGitHub, URL: "https://github.com/group/app.git", Alias: "group/app",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.SourceType).To(Equal(config.DefaultRepoSourceTypeName))
		})

		It("updates the url of an associated repository", func() {
			spec := coderepo.SaveSpec{
				WorkspaceID: testWorkspaceID, Type: config.DefaultRepoSourceTypeName, URL: "https://git.example.com/group/app.git", Alias: "group/app",
			}
			_, err := coderepo.Save(ctx, db, spec)
			Expect(err).NotTo(HaveOccurred())

			spec.URL = "https://git.example.com/group/app-renamed.git"
			_, err = coderepo.Save(ctx, db, spec)
			Expect(err).NotTo(HaveOccurred())
			got, err := coderepo.Get(ctx, db, testWorkspaceID, "group/app")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.URL).To(Equal(spec.URL))
			Expect(got.Creator).To(Equal("creator"))
		})
	})
})
