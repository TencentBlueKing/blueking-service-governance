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

package appcfg

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Local Editor", func() {
	var ctx context.Context
	var store *AppConfigFileStoreMongo

	var appID string
	var normalAcf, overlayAcf *AppConfigFile

	BeforeEach(func() {
		ctx = context.Background()

		var err error
		store, err = NewAppConfigFileStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		appID = "test-app-" + stringx.Random(6)

		normalAcf = &AppConfigFile{
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:             appID,
				Name:              "test-values",
				Type:              AppConfigFileTypeNormal,
				ContentSourceType: ContentSourceTypeLocal,
			},
		}
		overlayAcf = &AppConfigFile{
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:             appID,
				Name:              "test-values",
				Type:              AppConfigFileTypeOverlay,
				ContentSourceType: ContentSourceTypeLocal,
			},
		}
		normalAcf.ID, _ = store.Add(ctx, *normalAcf)
		overlayAcf.ID, _ = store.Add(ctx, *overlayAcf)
	})

	Context("GetEditableContentField", func() {
		It("should return Content for normal file", func() {
			field := newLocalAppConfigFileEditor(store, normalAcf).GetEditableContentField()
			Expect(field).To(Equal(EditableContentFieldContent))
		})
		It("should return OverlayContent for overlay file", func() {
			field := newLocalAppConfigFileEditor(store, overlayAcf).GetEditableContentField()
			Expect(field).To(Equal(EditableContentFieldOverlayContent))
		})
	})

	Context("SetContent", func() {
		It("should success for normal file", func() {
			err := newLocalAppConfigFileEditor(store, normalAcf).SetContent("test: content")
			Expect(err).ToNot(HaveOccurred())
			Expect(normalAcf.Content).To(Equal(lo.ToPtr("test: content")))
		})
		It("should fail for overlay file", func() {
			err := newLocalAppConfigFileEditor(store, overlayAcf).SetContent("test: content")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only normal app config file can set"))
		})
	})

	Context("SetOverlayContent", func() {
		It("should fail for normal file", func() {
			err := newLocalAppConfigFileEditor(store, normalAcf).SetOverlayContent("test: content")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only overlay app config file can set"))
		})
		It("should success for overlay file", func() {
			err := newLocalAppConfigFileEditor(store, overlayAcf).SetOverlayContent("overlay: true")
			Expect(err).ToNot(HaveOccurred())
			Expect(overlayAcf.OverlayContent).To(Equal(lo.ToPtr("overlay: true")))
		})
	})

	setUpAppConfigFixture := func() (base, overlay AppConfigFile) {
		baseContent := `global:
  image: myapp:latest
  replicas: 3`
		baseAcf := AppConfigFile{
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:             appID,
				Name:              "base-values",
				Type:              AppConfigFileTypeNormal,
				ContentSourceType: ContentSourceTypeLocal,
				Content:           &baseContent,
			},
		}
		oid, err := store.Add(ctx, baseAcf)
		Expect(err).NotTo(HaveOccurred())

		overlayContent := `overlayVersion: "2"
patches:
- global:
    image: myapp:v2.0`
		overlayAcfNew := AppConfigFile{
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:               appID,
				Name:                "overlay-values",
				Type:                AppConfigFileTypeOverlay,
				ContentSourceType:   ContentSourceTypeLocal,
				BaseAppConfigFileID: &oid,
				OverlayContent:      &overlayContent,
			},
		}
		_, err = store.Add(ctx, overlayAcfNew)
		Expect(err).NotTo(HaveOccurred())

		return baseAcf, overlayAcfNew
	}

	Context("GetCompiledContent", func() {
		It("overlay type with base should work", func() {
			_, overlayAcfNew := setUpAppConfigFixture()

			result, err := newLocalAppConfigFileEditor(store, &overlayAcfNew).GetCompiledContent(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Compare the result with the expected merged content
			expectedYAML := `global:
  image: myapp:v2.0
  replicas: 3`

			equal, err := testutil.YAMLEqual(result, expectedYAML)
			Expect(err).ToNot(HaveOccurred())
			Expect(equal).To(BeTrue())
		})

		It("should return Content when OverlayContent is nil", func() {
			content := `global:
  image: myapp:latest
  replicas: 3`

			normalAcf.Content = &content
			normalAcf.OverlayContent = nil

			result, err := newLocalAppConfigFileEditor(nil, normalAcf).GetCompiledContent(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(content))
		})

		It("should merge YAML content correctly", func() {
			baseContent := `global:
  image: myapp:latest
  replicas: 3
  env: development
database:
  host: db.example.com
  port: 5432
  ssl: false`

			overlayContent := `overlayVersion: "2"
patches:
- global:
   image: myapp:v2.0
   env: production
  database:
    ssl: true
  cache:
    enabled: true`

			normalAcf.Content = &baseContent
			overlayAcf.OverlayContent = &overlayContent
			overlayAcf.BaseAppConfigFileID = &normalAcf.ID
			_, _ = store.Update(ctx, *normalAcf)
			_, _ = store.Update(ctx, *overlayAcf)

			result, err := newLocalAppConfigFileEditor(store, overlayAcf).GetCompiledContent(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Compare the result with the expected merged content
			expectedYAML := `global:
  image: myapp:v2.0
  replicas: 3
  env: production
database:
  host: db.example.com
  port: 5432
  ssl: true
cache:
  enabled: true`

			equal, err := testutil.YAMLEqual(result, expectedYAML)
			Expect(err).ToNot(HaveOccurred())
			Expect(equal).To(BeTrue())
		})

		It("should handle multiple patches correctly", func() {
			baseContent := `global:
  image: myapp:latest
  replicas: 3
  env: development`

			overlayContent := `overlayVersion: "2"
patches:
- global:
    image: myapp:v2.0
- global:
    env: production
- global:
    replicas: null`

			normalAcf.Content = &baseContent
			overlayAcf.OverlayContent = &overlayContent
			overlayAcf.BaseAppConfigFileID = &normalAcf.ID
			_, _ = store.Update(ctx, *normalAcf)
			_, _ = store.Update(ctx, *overlayAcf)

			result, err := newLocalAppConfigFileEditor(store, overlayAcf).GetCompiledContent(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Compare the result with the expected merged content
			expectedYAML := `global:
  image: myapp:v2.0
  env: production`

			equal, err := testutil.YAMLEqual(result, expectedYAML)
			Expect(err).ToNot(HaveOccurred())
			Expect(equal).To(BeTrue())
		})

		It("should return error when Content is nil", func() {
			normalAcf.Content = nil

			_, err := newLocalAppConfigFileEditor(store, normalAcf).GetCompiledContent(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("app config file content is empty"))
		})
	})
})

var _ = Describe("BSCP Editor", func() {
	var store *AppConfigFileStoreMongo

	var appID string
	var normalAcf, overlayAcf *AppConfigFile

	BeforeEach(func() {
		var err error

		store, err = NewAppConfigFileStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		appID = "test-app-" + stringx.Random(6)

		normalAcf = &AppConfigFile{
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:             appID,
				Name:              "test-values",
				Type:              AppConfigFileTypeNormal,
				ContentSourceType: ContentSourceTypeBSCP,
			},
		}
		overlayAcf = &AppConfigFile{
			AppConfigFileContentSpec: AppConfigFileContentSpec{
				AppID:             appID,
				Name:              "test-values",
				Type:              AppConfigFileTypeOverlay,
				ContentSourceType: ContentSourceTypeBSCP,
			},
		}
	})

	Context("GetEditableContentField", func() {
		It("should return OverlayContent for normal file", func() {
			field := newBSCPAppConfigFileEditor(store, normalAcf).GetEditableContentField()
			Expect(field).To(Equal(EditableContentFieldOverlayContent))
		})
		It("should return None for overlay file", func() {
			field := newBSCPAppConfigFileEditor(store, overlayAcf).GetEditableContentField()
			Expect(field).To(Equal(EditableContentFieldNone))
		})
	})

	Context("SetContent", func() {
		It("should fail for normal file", func() {
			err := newBSCPAppConfigFileEditor(store, normalAcf).SetContent("test: content")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to set content"))
		})
		It("should fail for overlay file", func() {
			err := newBSCPAppConfigFileEditor(store, overlayAcf).SetContent("test: content")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to set content"))
		})
	})

	Context("SetOverlayContent", func() {
		It("should success for normal file", func() {
			err := newBSCPAppConfigFileEditor(store, normalAcf).SetOverlayContent("overlay: true")
			Expect(err).To(Not(HaveOccurred()))
			Expect(normalAcf.OverlayContent).To(Equal(lo.ToPtr("overlay: true")))
		})
		It("should success for overlay file", func() {
			err := newBSCPAppConfigFileEditor(store, overlayAcf).SetOverlayContent("overlay: true")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only normal app config file can set overlay content"))
		})
	})
})
