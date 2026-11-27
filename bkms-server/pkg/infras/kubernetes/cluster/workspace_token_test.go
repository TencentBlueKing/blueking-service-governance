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

package cluster_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

// fixedResolver 返回固定 Token 的解析函数，并记录收到的空间与集群
func fixedResolver(token string, gotWorkspaceID, gotClusterID *string) cluster.TokenResolver {
	return func(_ context.Context, workspaceID, clusterID string) (string, error) {
		*gotWorkspaceID, *gotClusterID = workspaceID, clusterID
		return token, nil
	}
}

// get 用集群配置请求 BCS 网关上的任意路径
func get(cfg *cluster.Config) (*http.Response, error) {
	httpClient, err := rest.HTTPClientFor(cfg.Rest)
	Expect(err).NotTo(HaveOccurred())
	return httpClient.Get(cfg.Rest.Host + "/version")
}

var _ = Describe("NewConfigForWorkspace", func() {
	ctx := context.Background()

	It("uses the platform token when workspace cluster tokens are disabled", func() {
		var gotWorkspaceID, gotClusterID string
		useBCSConfig("https://bcs.example.com", false, fixedResolver("ws-token", &gotWorkspaceID, &gotClusterID))

		cfg, err := cluster.NewConfigForWorkspace(ctx, "ws-1", "BCS-K8S-00001")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Rest.BearerToken).To(Equal(platformToken))
		Expect(cfg.Rest.Host).To(Equal("https://bcs.example.com/clusters/BCS-K8S-00001"))
		Expect(gotWorkspaceID).To(BeEmpty())
	})

	It("uses the token resolved for the workspace when enabled", func() {
		var gotWorkspaceID, gotClusterID string
		useBCSConfig("https://bcs.example.com", true, fixedResolver("ws-token", &gotWorkspaceID, &gotClusterID))

		cfg, err := cluster.NewConfigForWorkspace(ctx, "ws-1", "BCS-K8S-00001")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Rest.BearerToken).To(Equal("ws-token"))
		Expect(cfg.ClusterID).To(Equal("BCS-K8S-00001"))
		Expect(gotWorkspaceID).To(Equal("ws-1"))
		Expect(gotClusterID).To(Equal("BCS-K8S-00001"))
	})

	It("returns the resolver error when the token cannot be resolved", func() {
		errNotAccessed := errors.New("cluster not accessed")
		useBCSConfig("https://bcs.example.com", true, func(context.Context, string, string) (string, error) {
			return "", errNotAccessed
		})

		_, err := cluster.NewConfigForWorkspace(ctx, "ws-1", "BCS-K8S-00001")
		Expect(err).To(MatchError(errNotAccessed))
	})

	It("fails when enabled but no resolver is set", func() {
		useBCSConfig("https://bcs.example.com", true, nil)

		_, err := cluster.NewConfigForWorkspace(ctx, "ws-1", "BCS-K8S-00001")
		Expect(err).To(MatchError(ContainSubstring("resolver is not set")))
	})

	Context("when the BCS gateway responds", func() {
		var status int

		BeforeEach(func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			DeferCleanup(server.Close)
			var gotWorkspaceID, gotClusterID string
			useBCSConfig(server.URL, true, fixedResolver("ws-token", &gotWorkspaceID, &gotClusterID))
		})

		It("converts 401 to ErrUnauthorized", func() {
			status = http.StatusUnauthorized
			cfg, err := cluster.NewConfigForWorkspace(ctx, "ws-1", "BCS-K8S-00001")
			Expect(err).NotTo(HaveOccurred())

			_, err = get(cfg)
			Expect(errors.Is(err, cluster.ErrUnauthorized)).To(BeTrue())
		})

		It("returns CLUSTER_TOKEN_UNAUTHORIZED instead of an internal error", func() {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(bkerrs.ErrorHandler())
			router.GET("/ping", func(c *gin.Context) {
				cause := errors.Wrapf(cluster.ErrUnauthorized, "cluster %s, GET /version", "BCS-K8S-16060")
				bkerrs.AbortWithErr(c, bkerrs.Wrap(cause, bkerrs.ErrCodeInternalServerError, "list helm releases"))
			})

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

			Expect(rec.Code).To(Equal(http.StatusForbidden))
			var out struct {
				Error struct {
					Code    string           `json:"code"`
					Details []map[string]any `json:"details"`
				} `json:"error"`
			}
			Expect(json.Unmarshal(rec.Body.Bytes(), &out)).To(Succeed())
			Expect(out.Error.Code).To(Equal(string(bkerrs.ErrCodeNoPermission)))
			Expect(out.Error.Details).To(HaveLen(1))
			Expect(out.Error.Details[0]["code"]).To(Equal(string(bkerrs.ErrDetailCodeClusterTokenUnauthorized)))
		})

		It("passes other responses through", func() {
			status = http.StatusForbidden
			cfg, err := cluster.NewConfigForWorkspace(ctx, "ws-1", "BCS-K8S-00001")
			Expect(err).NotTo(HaveOccurred())

			resp, err := get(cfg)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})
	})
})

var _ = Describe("ConfigCache", func() {
	ctx := context.Background()
	var calls int

	BeforeEach(func() {
		calls = 0
		useBCSConfig("https://bcs.example.com", true, func(context.Context, string, string) (string, error) {
			calls++
			return "ws-token", nil
		})
	})

	It("resolves the token once for the same workspace and cluster", func() {
		var cache cluster.ConfigCache

		first, err := cache.Get(ctx, "ws-1", "BCS-K8S-00001")
		Expect(err).NotTo(HaveOccurred())
		second, err := cache.Get(ctx, "ws-1", "BCS-K8S-00001")
		Expect(err).NotTo(HaveOccurred())
		_, err = cache.Get(ctx, "ws-1", "BCS-K8S-00002")
		Expect(err).NotTo(HaveOccurred())

		Expect(second).To(BeIdenticalTo(first))
		Expect(calls).To(Equal(2))
	})

	It("does not cache when the cache is nil", func() {
		var cache *cluster.ConfigCache

		_, err := cache.Get(ctx, "ws-1", "BCS-K8S-00001")
		Expect(err).NotTo(HaveOccurred())
		_, err = cache.Get(ctx, "ws-1", "BCS-K8S-00001")
		Expect(err).NotTo(HaveOccurred())

		Expect(calls).To(Equal(2))
	})
})
