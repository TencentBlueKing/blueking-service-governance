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

package batchrelease_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"k8s.io/client-go/rest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

const rolloutsPath = "/apis/tkex.tencent.com/v1alpha1/namespaces/demo-ns/rollouts"

type recordedRequest struct {
	Method      string
	Path        string
	ContentType string
	Body        map[string]any
}

// fakeAPIServer 模拟 apiserver 中 Rollout 资源的行为：
// 未安装 CRD 时所有 rollouts 路径返回纯文本 404，安装后按对象是否存在返回 Status 404 或对象本身
type fakeAPIServer struct {
	mu        sync.Mutex
	installed bool
	objects   map[string]map[string]any
	requests  []recordedRequest
}

func (s *fakeAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req := recordedRequest{Method: r.Method, Path: r.URL.Path, ContentType: r.Header.Get("Content-Type")}
	if data, _ := io.ReadAll(r.Body); len(data) > 0 {
		Expect(json.Unmarshal(data, &req.Body)).To(Succeed())
	}
	s.requests = append(s.requests, req)

	if !s.installed || !strings.HasPrefix(r.URL.Path, rolloutsPath) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 page not found"))
		return
	}

	subPath := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, rolloutsPath), "/")
	if subPath == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"apiVersion": "tkex.tencent.com/v1alpha1",
			"kind":       "RolloutList",
			"metadata":   map[string]any{"resourceVersion": "1"},
			"items":      []any{},
		})
		return
	}

	name := strings.TrimSuffix(subPath, "/status")
	obj, exists := s.objects[name]
	switch {
	case r.Method == http.MethodPatch && req.ContentType == "application/apply-patch+yaml":
		s.objects[name] = req.Body
		writeJSON(w, http.StatusOK, req.Body)
	case !exists:
		writeJSON(w, http.StatusNotFound, map[string]any{
			"kind":       "Status",
			"apiVersion": "v1",
			"status":     "Failure",
			"message":    fmt.Sprintf("rollouts.tkex.tencent.com %q not found", name),
			"reason":     "NotFound",
			"details":    map[string]any{"name": name, "group": "tkex.tencent.com", "kind": "rollouts"},
			"code":       http.StatusNotFound,
		})
	case r.Method == http.MethodDelete:
		delete(s.objects, name)
		writeJSON(w, http.StatusOK, map[string]any{"kind": "Status", "apiVersion": "v1", "status": "Success"})
	default:
		writeJSON(w, http.StatusOK, obj)
	}
}

func (s *fakeAPIServer) lastRequest() recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[len(s.requests)-1]
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	Expect(json.NewEncoder(w).Encode(body)).To(Succeed())
}

var _ = Describe("RolloutClient", func() {
	var (
		ctx     context.Context
		fake    *fakeAPIServer
		server  *httptest.Server
		client  *batchrelease.RolloutClient
		rollout map[string]any
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake = &fakeAPIServer{installed: true, objects: map[string]map[string]any{}}
		server = httptest.NewServer(fake)
		client = batchrelease.NewRolloutClient(&cluster.Config{
			Rest:      &rest.Config{Host: server.URL},
			ClusterID: "BCS-K8S-00000",
		})

		var err error
		rollout, err = batchrelease.BuildManifest("GameDeployment", "demo", []releasestrategy.Batch{
			{Target: "1", PauseMode: releasestrategy.PauseModePermanent},
			{Target: "50%", PauseMode: releasestrategy.PauseModeNone},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Context("when the rollout CRD is installed", func() {
		It("applies, reads and deletes the rollout", func() {
			applied, err := client.Apply(ctx, "demo-ns", rollout)
			Expect(err).NotTo(HaveOccurred())
			Expect(applied.GetName()).To(Equal("demo"))
			applyReq := fake.lastRequest()
			Expect(applyReq.Method).To(Equal(http.MethodPatch))
			Expect(applyReq.Path).To(Equal(rolloutsPath + "/demo"))
			Expect(applyReq.Body).To(HaveKeyWithValue("spec", HaveKeyWithValue("type", "Canary")))

			obj, err := client.Get(ctx, "demo-ns", "demo")
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.GetName()).To(Equal("demo"))

			Expect(client.Delete(ctx, "demo-ns", "demo")).To(Succeed())
			_, err = client.Get(ctx, "demo-ns", "demo")
			Expect(errors.Is(err, batchrelease.ErrRolloutNotFound)).To(BeTrue())
		})

		It("parses the status of an existing rollout", func() {
			fake.objects["demo"] = map[string]any{
				"apiVersion": "tkex.tencent.com/v1alpha1",
				"kind":       "Rollout",
				"metadata":   map[string]any{"name": "demo", "namespace": "demo-ns"},
				"status": map[string]any{
					"phase":      "Running",
					"paused":     "false",
					"stepStatus": map[string]any{"currentStepIndex": 0, "currentStepPhase": "PausedTemporarily"},
				},
			}

			status, err := client.GetStatus(ctx, "demo-ns", "demo")
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Phase).To(Equal("Running"))
			Expect(status.CurrentStepIndex).To(Equal(lo.ToPtr(int64(0))))
			Expect(status.BatchPhase(batchrelease.WorkloadSnapshot{})).To(Equal(batchrelease.BatchPhaseTimedPaused))
		})

		It("patches the status subresource", func() {
			fake.objects["demo"] = rollout

			Expect(client.UpdateStatus(ctx, "demo-ns", "demo", batchrelease.StatusPatch{
				Paused: lo.ToPtr(batchrelease.PausedFalse),
			})).To(Succeed())
			Expect(fake.lastRequest()).To(Equal(recordedRequest{
				Method:      http.MethodPatch,
				Path:        rolloutsPath + "/demo/status",
				ContentType: "application/merge-patch+json",
				Body:        map[string]any{"status": map[string]any{"paused": "false"}},
			}))

			Expect(client.UpdateStatus(ctx, "demo-ns", "demo", batchrelease.StatusPatch{
				Paused:           lo.ToPtr(batchrelease.PausedFalse),
				CurrentStepIndex: lo.ToPtr(int64(-1)),
			})).To(Succeed())
			Expect(fake.lastRequest().Body).To(Equal(map[string]any{
				"status": map[string]any{
					"paused": "false",
					"stepStatus": map[string]any{
						"currentStepIndex": float64(-1),
						"pauseStartTime":   nil,
					},
				},
			}))
		})

		It("rejects an empty status patch", func() {
			Expect(client.UpdateStatus(ctx, "demo-ns", "demo", batchrelease.StatusPatch{})).NotTo(Succeed())
		})

		It("reports missing rollouts as not found", func() {
			_, err := client.Get(ctx, "demo-ns", "missing")
			Expect(errors.Is(err, batchrelease.ErrRolloutNotFound)).To(BeTrue())

			err = client.UpdateStatus(ctx, "demo-ns", "missing", batchrelease.StatusPatch{
				Paused: lo.ToPtr(batchrelease.PausedTrue),
			})
			Expect(errors.Is(err, batchrelease.ErrRolloutNotFound)).To(BeTrue())

			Expect(client.Delete(ctx, "demo-ns", "missing")).To(Succeed())
		})
	})

	Context("when the rollout CRD is not installed", func() {
		BeforeEach(func() {
			fake.installed = false
		})

		It("reports rollout not installed with the cluster message", func() {
			_, err := client.Get(ctx, "demo-ns", "demo")
			Expect(errors.Is(err, batchrelease.ErrRolloutNotInstalled)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("BCS-K8S-00000"))
			Expect(err.Error()).To(ContainSubstring("the server could not find the requested resource"))

			_, err = client.Apply(ctx, "demo-ns", rollout)
			Expect(errors.Is(err, batchrelease.ErrRolloutNotInstalled)).To(BeTrue())

			err = client.UpdateStatus(ctx, "demo-ns", "demo", batchrelease.StatusPatch{
				Paused: lo.ToPtr(batchrelease.PausedTrue),
			})
			Expect(errors.Is(err, batchrelease.ErrRolloutNotInstalled)).To(BeTrue())
		})

		It("treats deletion as a no-op", func() {
			Expect(client.Delete(ctx, "demo-ns", "demo")).To(Succeed())
		})
	})
})
