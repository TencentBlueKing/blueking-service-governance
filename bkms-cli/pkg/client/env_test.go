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

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment APIs", func() {
	var cli *SvcBasedClient
	var server *httptest.Server
	var status int
	var response string
	var requestPath, requestMethod string
	var requestBody CreateFeatureEnvBody

	BeforeEach(func() {
		status = http.StatusOK
		response = `{"data":[]}`
		requestBody = CreateFeatureEnvBody{}
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()
			requestPath, requestMethod = r.URL.Path, r.Method
			if r.Method == http.MethodPost {
				Expect(json.NewDecoder(r.Body).Decode(&requestBody)).To(Succeed())
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(response))
		}))
		cli = &SvcBasedClient{cli: resty.New().SetBaseURL(server.URL)}
	})
	AfterEach(func() { server.Close() })

	It("lists app environments and preserves feature ownership", func() {
		response = `{"data":[{"id":"standard-id","name":"staging","kind":"standard"},
   {"id":"feature-id","name":"feat-1","kind":"feature","ownerAppID":"app-1","sourceEnvID":"standard-id","status":"Ready"}]}`
		envs, err := cli.ListAppEnvs(context.Background(), "app-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(requestPath).To(Equal("/bkms/v1/bkms-server/apps/app-1/envs"))
		Expect(requestMethod).To(Equal(http.MethodGet))
		Expect(envs).To(HaveLen(2))
		Expect(envs[1].OwnerAppID).To(Equal("app-1"))
		Expect(envs[1].SourceEnvID).To(Equal("standard-id"))
		Expect(envs[1].Kind).To(Equal("feature"))
	})

	DescribeTable("creates a feature environment using the backend contract",
		func(code int) {
			status = code
			response = `{"data":{"id":"created-id","name":"feat-1","kind":"feature","cluster":{"namespace":"feat-ns"}}}`
			body := CreateFeatureEnvBody{SourceEnvID: "source-id", DisplayName: "Feature"}
			env, err := cli.CreateFeatureEnv(context.Background(), "app-1", body)
			Expect(err).NotTo(HaveOccurred())
			Expect(requestMethod).To(Equal(http.MethodPost))
			Expect(requestPath).To(Equal("/bkms/v1/bkms-server/apps/app-1/feat-envs"))
			Expect(requestBody).To(Equal(body))
			Expect(env.Name).To(Equal("feat-1"))
			Expect(env.Cluster.Namespace).To(Equal("feat-ns"))
		},
		Entry("OK", http.StatusOK),
		Entry("Created", http.StatusCreated),
	)

	DescribeTable("deletes the resolved environment through the shared environment endpoint",
		func(code int) {
			status = code
			response = "{}"
			err := cli.DeleteEnv(context.Background(), "feature-id")
			Expect(err).NotTo(HaveOccurred())
			Expect(requestMethod).To(Equal(http.MethodDelete))
			Expect(requestPath).To(Equal("/bkms/v1/bkms-server/envs/feature-id"))
		},
		Entry("OK", http.StatusOK),
		Entry("NoContent", http.StatusNoContent),
	)

	It("preserves the server rejection when the environment still has deployed apps", func() {
		status = http.StatusInternalServerError
		response = `{"message":"environment has 1 apps, cannot delete"}`
		err := cli.DeleteEnv(context.Background(), "feature-id")
		Expect(err).To(MatchError(ContainSubstring("environment has 1 apps, cannot delete")))
	})

	DescribeTable("propagates backend failures",
		func(operation string) {
			status = http.StatusForbidden
			response = `{"message":"permission denied"}`
			var err error
			switch operation {
			case "app-list":
				_, err = cli.ListAppEnvs(context.Background(), "app-1")
			case "create":
				_, err = cli.CreateFeatureEnv(context.Background(), "app-1", CreateFeatureEnvBody{})
			case "delete":
				err = cli.DeleteEnv(context.Background(), "feature-id")
			}
			Expect(err).To(MatchError(ContainSubstring("[403]")))
			Expect(err).To(MatchError(ContainSubstring("permission denied")))
		},
		Entry("app environments", "app-list"),
		Entry("create", "create"),
		Entry("delete", "delete"),
	)
})
