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

package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/mockey"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
)

// Build a local repository name using the local registry URL
func localRepoName(repo string) string {
	// Remove the scheme from the URL
	url := testutil.ContainerRegistryURL()
	url = strings.SplitN(url, "://", 2)[1]
	return url + "/" + repo
}

var _ = Describe("Registry Client", func() {
	var regCli *Client
	var sampleRepo string
	var ctx context.Context

	BeforeEach(func() {
		// Local test registry access (insecure)
		regCli = New("", "", true)
		sampleRepo = localRepoName("fixture/sample")
		ctx = context.Background()
	})

	AfterEach(func() {
		mockey.UnPatchAll()
	})

	Describe("Client Creation", func() {
		Context("when creating a client without authentication", func() {
			It("should create a client successfully", func() {
				c := New("", "", false)
				Expect(c).NotTo(BeNil())
				Expect(c.remoteOpts).To(HaveLen(1))
			})
		})

		Context("when creating a client with authentication", func() {
			It("should create a client with auth options", func() {
				c := New("admin", "passwd", false)
				Expect(c).NotTo(BeNil())
				Expect(c.remoteOpts).To(HaveLen(2))
			})
		})

		Context("when creating a client with insecure option", func() {
			It("should create a client with transport options", func() {
				c := New("", "", true)
				Expect(c).NotTo(BeNil())
				Expect(c.nameOpts).To(HaveLen(1))
				Expect(c.remoteOpts).To(HaveLen(1))
			})
		})

		Context("when creating a client with both auth and insecure", func() {
			It("should create a client with both options", func() {
				c := New("admin", "passwd", true)
				Expect(c).NotTo(BeNil())
				Expect(c.nameOpts).To(HaveLen(1))
				Expect(c.remoteOpts).To(HaveLen(2))
			})
		})
	})

	Describe("ListTags", func() {
		Context("when listing tags for nginx repository", func() {
			It("should return a list of tags", func() {
				tags, total, err := regCli.ListTags(ctx, sampleRepo, "", 1, 500)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(tags)).To(Equal(total))
				Expect(tags).NotTo(BeEmpty())
				Expect(tags).To(ContainElement("latest"))
			})
		})

		Context("when repository name is invalid", func() {
			It("should return an error", func() {
				_, _, err := regCli.ListTags(ctx, localRepoName("invalid-repo"), "", 1, 5)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetTagDetail", func() {
		Context("when getting details for nginx:latest", func() {
			It("should return image details", func() {
				detail, err := regCli.GetTagDetail(ctx, sampleRepo, "latest")
				Expect(err).NotTo(HaveOccurred())
				Expect(detail.Tag).To(Equal("latest"))
				Expect(detail.Digest).NotTo(BeEmpty())
				Expect(detail.MediaType).To(Equal("application/vnd.docker.distribution.manifest.v2+json"))
				Expect(detail.Size).To(BeNumerically(">", 0))
			})
		})

		Context("when tag does not exist", func() {
			It("should return an error", func() {
				_, err := regCli.GetTagDetail(ctx, sampleRepo, "nonexistent-tag-12345")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("HeadManifest", func() {
		It("should succeed when the tag manifest exists", func() {
			const (
				repoName = "fixture/sample"
				tagName  = "latest"
				digest   = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
			)
			parsedDigest, err := v1.NewHash(digest)
			Expect(err).NotTo(HaveOccurred())

			mockey.Mock(remote.Head).To(func(ref name.Reference, _ ...remote.Option) (*v1.Descriptor, error) {
				Expect(ref.String()).To(Equal("registry.example.com/" + repoName + ":" + tagName))
				return &v1.Descriptor{Digest: parsedDigest}, nil
			}).Build()

			client := New("", "", true)
			Expect(client.HeadManifest(ctx, "registry.example.com/"+repoName, tagName)).To(Succeed())
		})

		It("should append the caller context to the base remote options", func() {
			const (
				repoName = "fixture/sample"
				tagName  = "latest"
			)

			var gotOptCount int
			mockey.Mock(remote.Head).To(func(_ name.Reference, opts ...remote.Option) (*v1.Descriptor, error) {
				gotOptCount = len(opts)
				return nil, fmt.Errorf("stop here")
			}).Build()

			client := New("", "", true)
			baseOptCount := len(client.remoteOpts)
			_ = client.HeadManifest(ctx, "registry.example.com/"+repoName, tagName)

			// 多出的一项即 remote.WithContext；同时确认基础选项没有被就地追加污染
			Expect(gotOptCount).To(Equal(baseOptCount + 1))
			Expect(client.remoteOpts).To(HaveLen(baseOptCount))
		})

		It("should classify tag not found when head returns 404", func() {
			const (
				repoName = "fixture/sample"
				tagName  = "missing-tag"
			)
			mockey.Mock(remote.Head).To(func(ref name.Reference, _ ...remote.Option) (*v1.Descriptor, error) {
				return nil, fmt.Errorf("lookup %s: %w", tagName, &transport.Error{StatusCode: http.StatusNotFound})
			}).Build()

			client := New("", "", true)
			err := client.HeadManifest(ctx, "registry.example.com/"+repoName, tagName)
			Expect(err).To(HaveOccurred())
			Expect(IsTagNotFound(err)).To(BeTrue())
			Expect(IsAuthRequired(err)).To(BeFalse())
		})

		It("should classify auth required when head returns 401", func() {
			const (
				repoName = "fixture/sample"
				tagName  = "latest"
			)
			mockey.Mock(remote.Head).To(func(ref name.Reference, _ ...remote.Option) (*v1.Descriptor, error) {
				return nil, fmt.Errorf("lookup %s: %w", tagName, &transport.Error{StatusCode: http.StatusUnauthorized})
			}).Build()

			client := New("", "", true)
			err := client.HeadManifest(ctx, "registry.example.com/"+repoName, tagName)
			Expect(err).To(HaveOccurred())
			Expect(IsAuthRequired(err)).To(BeTrue())
			Expect(IsTagNotFound(err)).To(BeFalse())
		})
	})

	Describe("DeleteTag", func() {
		It("should delete manifest by digest resolved from tag", func() {
			const (
				repoName = "fixture/sample"
				tagName  = "latest"
				digest   = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
			)

			// 直接 mock 三方 remote 包，验证 client 会先按 tag 查询 digest，
			// 再按 digest 引用发起删除。
			parsedDigest, err := v1.NewHash(digest)
			Expect(err).NotTo(HaveOccurred())

			var deleteRef string
			mockey.Mock(remote.Head).To(func(ref name.Reference, _ ...remote.Option) (*v1.Descriptor, error) {
				Expect(ref.String()).To(Equal("registry.example.com/" + repoName + ":" + tagName))
				return &v1.Descriptor{Digest: parsedDigest}, nil
			}).Build()
			mockey.Mock(remote.Delete).To(func(ref name.Reference, _ ...remote.Option) error {
				deleteRef = ref.String()
				return nil
			}).Build()

			client := New("", "", true)
			err = client.DeleteTag(ctx, "registry.example.com/"+repoName, tagName)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteRef).To(Equal("registry.example.com/" + repoName + "@" + digest))
		})

		It("should classify tag not found when head manifest returns 404", func() {
			const (
				repoName = "fixture/sample"
				tagName  = "missing-tag"
			)

			// 模拟 registry 返回 404，验证包装后的错误仍可被 IsTagNotFound 识别。
			mockey.Mock(remote.Head).To(func(ref name.Reference, _ ...remote.Option) (*v1.Descriptor, error) {
				Expect(ref.String()).To(Equal("registry.example.com/" + repoName + ":" + tagName))
				return nil, fmt.Errorf("lookup %s: %w", tagName, &transport.Error{StatusCode: http.StatusNotFound})
			}).Build()

			client := New("", "", true)
			err := client.DeleteTag(ctx, "registry.example.com/"+repoName, tagName)
			Expect(err).To(HaveOccurred())
			Expect(IsTagNotFound(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("head manifest"))
			Expect(err.Error()).To(ContainSubstring(tagName))
		})
	})

	Describe("ListTagsWithDetail", func() {
		Context("when getting detail for all tags", func() {
			It("should return detail for all tags", func() {
				details, total, err := regCli.ListTagsWithDetail(ctx, sampleRepo, "1.0.0", 1, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(BeNumerically(">=", 1))
				Expect(details).NotTo(BeEmpty())
				Expect(len(details)).To(Equal(1))
				for _, detail := range details {
					Expect(detail.Tag).NotTo(BeEmpty())
					Expect(detail.Tag).To(Equal("1.0.0"))
					Expect(detail.Digest).NotTo(BeEmpty())
					Expect(detail.Size).To(BeNumerically(">", 0))
				}
			})
		})

		Context("when repository does not exist", func() {
			It("should return an error", func() {
				_, _, err := regCli.ListTagsWithDetail(ctx, localRepoName("nonexistent-repo-12345"), "", 1, 5)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ImageDetail", func() {
		DescribeTable("HumanizeSize",
			func(size int64, expected string) {
				detail := ImageDetail{Size: size}
				Expect(detail.HumanizeSize()).To(Equal(expected))
			},
			Entry("when size is zero", int64(0), "0 B"),
			Entry("when size is in bytes", int64(512), "512 B"),
			Entry("when size is exactly 1 KB", int64(1024), "1.0 KB"),
			Entry("when size is in KB", int64(1536), "1.5 KB"),
			Entry("when size is exactly 1 MB", int64(1048576), "1.0 MB"),
			Entry("when size is in MB", int64(1572864), "1.5 MB"),
			Entry("when size is in GB", int64(1610612736), "1.5 GB"),
		)
	})

	// Integration tests that can be run manually
	Describe("Integration Tests", func() {
		Context("when testing with local registry sample repository", func() {
			It("should perform end-to-end operations", func() {
				By("listing tags for sample repository")
				tags, total, err := regCli.ListTags(ctx, sampleRepo, "", 1, 10)
				Expect(total).To(BeNumerically(">=", 1))
				Expect(err).NotTo(HaveOccurred())
				Expect(tags).NotTo(BeEmpty())

				By("getting details for the latest tag")
				detail, err := regCli.GetTagDetail(ctx, sampleRepo, "latest")
				Expect(err).NotTo(HaveOccurred())
				Expect(detail.Tag).To(Equal("latest"))
				Expect(detail.Size).To(BeNumerically(">", 0))

				By("verifying humanized size format")
				humanSize := detail.HumanizeSize()
				Expect(humanSize).To(MatchRegexp(`^\d+(\.\d+)?\s+B$`))

				By("getting all repository details")
				details, total, err := regCli.ListTagsWithDetail(ctx, sampleRepo, "", 1, 15)
				Expect(total).To(BeNumerically(">=", 1))
				Expect(err).NotTo(HaveOccurred())
				Expect(details).NotTo(BeEmpty())
				Expect(len(details)).To(Equal(2))
			})
		})
	})
})
