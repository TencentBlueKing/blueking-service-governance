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
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Local Provider", func() {
	var store *AppConfigFileStoreMongo
	var ctx context.Context

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
	})

	Context("GetBaseContentInfo", func() {
		It("should return nil for normal file", func() {
			info, err := newLocalBaseContentProvider(store, normalAcf).GetInfo(ctx)
			Expect(errors.Is(err, ErrBaseContentEmpty)).To(BeTrue())
			Expect(info).To(BeNil())
		})

		It("should return base content info for overlay file", func() {
			// Create and save a base app config file
			baseContent := "base: content"
			baseAcf := &AppConfigFile{
				AppConfigFileContentSpec: AppConfigFileContentSpec{
					AppID:             appID,
					Name:              "base-values",
					Type:              AppConfigFileTypeNormal,
					ContentSourceType: ContentSourceTypeLocal,
					Content:           &baseContent,
				},
			}
			baseID, err := store.Add(ctx, *baseAcf)
			Expect(err).ToNot(HaveOccurred())

			// Set the base app config file ID for overlay file
			overlayAcf.BaseAppConfigFileID = &baseID

			baseInfo, err := newLocalBaseContentProvider(store, overlayAcf).GetInfo(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(baseInfo).To(Equal(&BaseContentInfo{
				HolderID:                baseID,
				HolderName:              baseAcf.Name,
				HolderContentSourceType: "local",
				IsFromAnotherFile:       true,
				Content:                 baseContent,
			}))
		})
	})
})

var _ = Describe("BSCP Provider", func() {
	var store *AppConfigFileStoreMongo
	var ctx context.Context

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

	Context("GetBaseContentInfo", func() {
		It("should return base content info for normal file with overlay content", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(
					auth.User{ID: "test-user"},
				).Build()
				mockey.Mock(bscp.New).Return(&bscp.ApiClient{}, nil).Build()
				mockey.Mock((*bscp.ApiClient).ListServiceVersions).Return(
					bscp.Versions{{ID: "91011", IsFullyReleased: true}}, nil,
				).Build()
				mockey.Mock((*bscp.ApiClient).GetServiceConfig).Return(
					bscp.NewKeyValue("stag", "foo: bar\nbar: baz\n", ""), nil,
				).Build()

				overlayContent := "bar: bax"
				normalAcf.BSCPConfig = &BSCPConfig{BizID: "123", ServiceID: "456", VersionID: "789", ConfigID: "666"}
				normalAcf.OverlayContent = &overlayContent

				baseInfo, err := newBSCPBaseContentProvider(store, normalAcf).GetInfo(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(baseInfo).To(Equal(&BaseContentInfo{
					HolderID:                normalAcf.ID,
					HolderName:              normalAcf.Name,
					HolderContentSourceType: "bscp",
					IsFromAnotherFile:       false,
					Content:                 "foo: bar\nbar: bax\n",
				}))
			})
		})

		It("should return base content info for overlay file", func() {
			// Create and save a base app config file
			baseContent := "base: content"
			baseAcf := &AppConfigFile{
				AppConfigFileContentSpec: AppConfigFileContentSpec{
					AppID: appID,
					Name:  "base-values",
					Type:  AppConfigFileTypeNormal,
					// Use the Local type as base for testing because reading BSCP content is not implemented yet
					ContentSourceType: ContentSourceTypeLocal,
					Content:           &baseContent,
				},
			}
			baseID, err := store.Add(ctx, *baseAcf)
			Expect(err).ToNot(HaveOccurred())

			// Set the base app config file ID for overlay file
			overlayAcf.BaseAppConfigFileID = &baseID

			baseInfo, err := newBSCPBaseContentProvider(store, overlayAcf).GetInfo(ctx)
			Expect(err).To(Not(HaveOccurred()))
			Expect(baseInfo).To(Equal(&BaseContentInfo{
				HolderID:                baseID,
				HolderName:              baseAcf.Name,
				HolderContentSourceType: "local",
				IsFromAnotherFile:       true,
				Content:                 baseContent,
			}))
		})
	})
})
