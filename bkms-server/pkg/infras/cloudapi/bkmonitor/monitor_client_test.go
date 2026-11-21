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

package bkmonitor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

var _ = Describe("MonitorGatewayClient", func() {
	newClientWithServer := func(handler http.HandlerFunc) *MonitorGatewayClient {
		server := httptest.NewServer(handler)
		DeferCleanup(server.Close)

		original := config.G
		config.G = &config.Config{}
		DeferCleanup(func() { config.G = original })

		client, err := newAPIClient("tester", server.URL)
		Expect(err).NotTo(HaveOccurred())
		return &MonitorGatewayClient{ApiClient: client}
	}

	Describe("SetBkmsSpaceStatus", func() {
		It("posts bkci space uid and status", func() {
			gotReq := make(chan map[string]any, 1)
			gotPath := make(chan string, 1)
			client := newClientWithServer(func(w http.ResponseWriter, r *http.Request) {
				gotPath <- r.Method + " " + r.URL.Path
				body, _ := io.ReadAll(r.Body)
				payload := map[string]any{}
				_ = json.Unmarshal(body, &payload)
				gotReq <- payload
				_, _ = w.Write([]byte(`{"result":true,"code":200,"data":{"status":"disabled"}}`))
			})

			Expect(client.SetBkmsSpaceStatus(context.Background(), "bkms-demo", SpaceStatusDisabled)).To(Succeed())
			Expect(<-gotPath).To(Equal("POST /app/metadata/set_bkms_space_status/"))
			Expect(<-gotReq).To(Equal(map[string]any{"space_uid": "bkci__bkms-demo", "status": "disabled"}))
		})

		It("returns error when returned status mismatches", func() {
			client := newClientWithServer(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"result":true,"code":200,"data":{"status":"normal"}}`))
			})

			Expect(client.SetBkmsSpaceStatus(context.Background(), "bkms-demo", SpaceStatusDisabled)).
				To(MatchError(ContainSubstring("normal")))
		})

		It("returns APIError with request id on business failure", func() {
			client := newClientWithServer(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"result":false,"code":400,"message":"invalid space","request_id":"req-1"}`))
			})

			err := client.SetBkmsSpaceStatus(context.Background(), "demo", SpaceStatusDisabled)

			var apiErr *APIError
			Expect(errors.As(err, &apiErr)).To(BeTrue())
			Expect(apiErr.Message).To(Equal("invalid space"))
			Expect(apiErr.RequestID).To(Equal("req-1"))
		})
	})

	Describe("GetMetadataSpaceDetail", func() {
		It("returns ErrSpaceNotFound when space is not found", func() {
			client := newClientWithServer(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"result":false,"code":404,"message":"space not found"}`))
			})

			_, err := client.GetMetadataSpaceDetail(context.Background(), "bkms-demo")
			Expect(err).To(MatchError(ErrSpaceNotFound))
		})

		It("does not treat other failures as ErrSpaceNotFound", func() {
			client := newClientWithServer(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"result":false,"code":500,"message":"internal"}`))
			})

			_, err := client.GetMetadataSpaceDetail(context.Background(), "bkms-demo")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrSpaceNotFound)).To(BeFalse())
		})
	})
})
