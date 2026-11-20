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

package gitremote

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

// pktLine 按 pkt-line 格式编码一行
func pktLine(s string) string {
	return fmt.Sprintf("%04x%s", len(s)+4, s)
}

const (
	shaMaster  = "1111111111111111111111111111111111111111"
	shaDevelop = "2222222222222222222222222222222222222222"
	shaTagObj  = "3333333333333333333333333333333333333333"
	shaTagPeel = "4444444444444444444444444444444444444444"
	shaLight   = "5555555555555555555555555555555555555555"
)

func sampleAdvertisement() string {
	return pktLine("# service=git-upload-pack\n") + "0000" +
		pktLine(shaMaster+" HEAD\x00multi_ack side-band-64k symref=HEAD:refs/heads/main agent=git/2.40\n") +
		pktLine(shaDevelop+" refs/heads/develop\n") +
		pktLine(shaMaster+" refs/heads/main\n") +
		pktLine(shaMaster+" refs/merge-requests/1/head\n") +
		pktLine(shaTagObj+" refs/tags/v1.0.0\n") +
		pktLine(shaTagPeel+" refs/tags/v1.0.0^{}\n") +
		pktLine(shaLight+" refs/tags/v1.1.0\n") +
		"0000"
}

var _ = Describe("ListRefs", func() {
	var (
		server  *httptest.Server
		handler http.HandlerFunc
	)

	BeforeEach(func() {
		handler = nil
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler(w, r)
		}))
	})

	AfterEach(func() {
		server.Close()
	})

	It("lists branches and tags with basic auth", func() {
		handler = func(w http.ResponseWriter, r *http.Request) {
			Expect(r.URL.Path).To(Equal("/group/app.git/info/refs"))
			Expect(r.URL.Query().Get("service")).To(Equal("git-upload-pack"))
			username, password, ok := r.BasicAuth()
			if !ok || username != "git" || password != "secret" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
			_, _ = w.Write([]byte(sampleAdvertisement()))
		}

		refs, err := ListRefs(context.Background(), server.URL+"/group/app.git/", Credential{"git", "secret"})
		Expect(err).NotTo(HaveOccurred())
		Expect(refs.DefaultBranch).To(Equal("main"))
		Expect(refs.Branches).To(Equal([]Ref{
			{Name: "develop", Path: "refs/heads/develop", SHA: shaDevelop},
			{Name: "main", Path: "refs/heads/main", SHA: shaMaster},
		}))
		// 附注 Tag 使用解引用后的提交 ID
		Expect(refs.Tags).To(Equal([]Ref{
			{Name: "v1.0.0", Path: "refs/tags/v1.0.0", SHA: shaTagPeel},
			{Name: "v1.1.0", Path: "refs/tags/v1.1.0", SHA: shaLight},
		}))
	})

	It("returns empty refs for an empty repository", func() {
		handler = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
			_, _ = w.Write([]byte(pktLine("# service=git-upload-pack\n") + "0000" +
				pktLine(strings.Repeat("0", 40)+" capabilities^{}\x00multi_ack\n") + "0000"))
		}

		refs, err := ListRefs(context.Background(), server.URL+"/empty.git", Credential{})
		Expect(err).NotTo(HaveOccurred())
		Expect(refs.Branches).To(BeEmpty())
		Expect(refs.Tags).To(BeEmpty())
	})

	DescribeTable("maps http status to errors",
		func(status int, expected error) {
			handler = func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}
			_, err := ListRefs(context.Background(), server.URL+"/app.git", Credential{"git", "wrong"})
			Expect(errors.Is(err, expected)).To(BeTrue(), "unexpected error: %v", err)
		},
		Entry("unauthorized", http.StatusUnauthorized, ErrAuthFailed),
		Entry("forbidden", http.StatusForbidden, ErrAuthFailed),
		Entry("not found", http.StatusNotFound, ErrRepoNotFound),
	)

	It("rejects responses that are not smart http advertisements", func() {
		handler = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>login</html>"))
		}
		_, err := ListRefs(context.Background(), server.URL+"/app.git", Credential{"git", "secret"})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, ErrAuthFailed)).To(BeFalse())
	})

	It("does not follow redirects to another host", func() {
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			Fail("redirect target should not be requested")
		}))
		defer other.Close()
		handler = func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, other.URL+"/login", http.StatusFound)
		}
		_, err := ListRefs(context.Background(), server.URL+"/app.git", Credential{"git", "secret"})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("ValidateRepoURL", func() {
	DescribeTable("validates repository url",
		func(repoURL string, valid bool) {
			err := ValidateRepoURL(repoURL)
			if valid {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(errors.Is(err, ErrInvalidRepoURL)).To(BeTrue())
		},
		Entry("https", "https://git.example.com/group/app.git", true),
		Entry("http", "http://git.example.com/group/app.git", true),
		Entry("ssh scheme", "ssh://git@git.example.com/group/app.git", false),
		Entry("scp style", "git@git.example.com:group/app.git", false),
		Entry("embedded credentials", "https://user:pass@git.example.com/group/app.git", false),
		Entry("missing host", "https:///group/app.git", false),
	)
})
