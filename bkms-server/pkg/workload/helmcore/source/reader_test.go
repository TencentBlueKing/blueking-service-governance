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

package repo

import (
	"github.com/Masterminds/semver/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

var _ = Describe("Test Reader", func() {
	var repoReader *Reader

	BeforeEach(func() {
		// TODO: Replace with a local helm repository for testing
		config := &bkmsapp.HelmRepoConfig{
			RepoURL:   testutil.HelmRegistryURL(),
			ChartName: "sample-app",
		}
		repoReader = NewReader(config)
	})

	Context("ReadFile", func() {
		It("should read Chart.yaml successfully", func() {
			content, err := repoReader.ReadFile(Version{Name: "0.1.0"}, "Chart.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("A minimal chart used for helm client tests"))
		})
		It("should return file not found error", func() {
			_, err := repoReader.ReadFile(Version{Name: "0.1.0"}, "non-existent-file.yaml")
			Expect(err).To(MatchError(ErrFileNotFound))
		})
	})

	Context("ListVersions", func() {
		It("should list versions successfully", func() {
			versions, err := repoReader.ListVersions()
			Expect(err).NotTo(HaveOccurred())
			_, err = semver.StrictNewVersion(versions[0].Name)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
