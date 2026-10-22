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

package appmodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const (
	fakeRolloutsPath    = "/apis/tkex.tencent.com/v1alpha1/namespaces/demo-ns/rollouts"
	fakeGameDeploysPath = "/apis/tkex.tencent.com/v1alpha1/namespaces/demo-ns/gamedeployments"
	fakePodsPath        = "/api/v1/namespaces/demo-ns/pods"
)

// fakeCluster 模拟 apiserver 中 Rollout、GameDeployment 与 Pod 资源：
//   - 未安装 Rollout CRD 时 rollouts 路径返回纯文本 404
//   - Rollout 的 SSA 仅在 spec 变化时递增 generation，并保留已有 status
//   - reconcile 为 true 时模拟控制器在 Rollout 下发后写入 workloadRevision 与 rolloutRevision；
//     observeSpecAfterGets 大于 0 时，rolloutRevision 在之后第 observeSpecAfterGets 次 GET 该 Rollout 时才写入，
//     模拟控制器滞后观察到新 spec
//   - staleWebhook 为 true 时模拟 webhook 缓存未同步，JSON Patch 修改 Pod 模板时仍注入 partition = replicas
type fakeCluster struct {
	mu                   sync.Mutex
	rolloutInstalled     bool
	reconcile            bool
	observeSpecAfterGets int
	failRolloutDelete    bool
	staleWebhook         bool
	rollouts             map[string]map[string]any
	gameDeploys          map[string]map[string]any
	pods                 map[string]map[string]any
	requests             []string

	pendingRolloutRevision string
	pendingGets            int
}

func newFakeCluster() *fakeCluster {
	return &fakeCluster{
		rolloutInstalled: true,
		reconcile:        true,
		rollouts:         map[string]map[string]any{},
		gameDeploys:      map[string]map[string]any{},
		pods:             map[string]map[string]any{},
	}
}

func (s *fakeCluster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	contentType := r.Header.Get("Content-Type")
	s.requests = append(s.requests, r.Method+" "+r.URL.Path+" "+contentType)
	data, _ := io.ReadAll(r.Body)

	var store map[string]map[string]any
	var prefix string
	switch {
	case strings.HasPrefix(r.URL.Path, fakeRolloutsPath) && s.rolloutInstalled:
		store, prefix = s.rollouts, fakeRolloutsPath
	case strings.HasPrefix(r.URL.Path, fakeGameDeploysPath):
		store, prefix = s.gameDeploys, fakeGameDeploysPath
	case strings.HasPrefix(r.URL.Path, fakePodsPath):
		store, prefix = s.pods, fakePodsPath
	default:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 page not found"))
		return
	}

	name := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	if name == "" {
		items := make([]any, 0, len(store))
		for _, obj := range store {
			items = append(items, obj)
		}
		fakeWriteJSON(w, http.StatusOK, map[string]any{
			"apiVersion": "v1", "kind": "List", "metadata": map[string]any{}, "items": items,
		})
		return
	}

	obj, exists := store[name]
	switch {
	case r.Method == http.MethodPatch && contentType == "application/apply-patch+yaml":
		var body map[string]any
		Expect(json.Unmarshal(data, &body)).To(Succeed())
		if prefix == fakeRolloutsPath {
			s.applyRollout(name, body)
		} else {
			store[name] = body
		}
		fakeWriteJSON(w, http.StatusOK, store[name])
	case !exists:
		fakeWriteJSON(w, http.StatusNotFound, map[string]any{
			"kind": "Status", "apiVersion": "v1", "status": "Failure",
			"reason": "NotFound", "code": http.StatusNotFound,
		})
	case r.Method == http.MethodDelete && s.failRolloutDelete:
		fakeWriteJSON(w, http.StatusInternalServerError, map[string]any{
			"kind": "Status", "apiVersion": "v1", "status": "Failure",
			"message": "etcd unavailable", "reason": "InternalError", "code": http.StatusInternalServerError,
		})
	case r.Method == http.MethodDelete:
		delete(store, name)
		fakeWriteJSON(w, http.StatusOK, map[string]any{"kind": "Status", "apiVersion": "v1", "status": "Success"})
	case r.Method == http.MethodPatch && contentType == "application/json-patch+json":
		var ops []map[string]any
		Expect(json.Unmarshal(data, &ops)).To(Succeed())
		applyFakeJSONPatch(obj, ops)
		if prefix == fakeGameDeploysPath && s.staleWebhook && strings.Contains(string(data), "/spec/template") {
			spec := obj["spec"].(map[string]any)
			spec["updateStrategy"].(map[string]any)["partition"] = spec["replicas"]
		}
		fakeWriteJSON(w, http.StatusOK, obj)
	default:
		if prefix == fakeRolloutsPath && s.pendingGets > 0 {
			s.pendingGets--
			if s.pendingGets == 0 {
				obj["status"].(map[string]any)["rolloutRevision"] = s.pendingRolloutRevision
			}
		}
		fakeWriteJSON(w, http.StatusOK, obj)
	}
}

func (s *fakeCluster) applyRollout(name string, body map[string]any) {
	generation := int64(1)
	status := map[string]any{}
	if prev, exists := s.rollouts[name]; exists {
		generation, _, _ = unstructured.NestedInt64(prev, "metadata", "generation")
		if !reflect.DeepEqual(prev["spec"], body["spec"]) {
			generation++
		}
		if prevStatus, ok := prev["status"].(map[string]any); ok {
			status = prevStatus
		}
	}
	body["metadata"].(map[string]any)["generation"] = generation
	body["status"] = status
	s.rollouts[name] = body

	if !s.reconcile {
		return
	}
	if status["workloadRevision"] == nil {
		status["workloadRevision"] = name + "-1"
	}
	revision := fmt.Sprintf("%s-rev-%d", name, generation)
	if s.observeSpecAfterGets == 0 {
		status["rolloutRevision"] = revision
		return
	}
	s.pendingRolloutRevision, s.pendingGets = revision, s.observeSpecAfterGets
}

func (s *fakeCluster) requestsWith(substr string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ret []string
	for _, req := range s.requests {
		if strings.Contains(req, substr) {
			ret = append(ret, req)
		}
	}
	return ret
}

// applyFakeJSONPatch 仅支持测试用到的 remove / replace，数组元素只支持 replace
func applyFakeJSONPatch(obj map[string]any, ops []map[string]any) {
	for _, op := range ops {
		fields := strings.Split(strings.TrimPrefix(op["path"].(string), "/"), "/")
		var parent any = obj
		for _, field := range fields[:len(fields)-1] {
			if list, ok := parent.([]any); ok {
				parent = list[fakeIndex(field)]
			} else {
				parent = parent.(map[string]any)[field]
			}
		}
		last := fields[len(fields)-1]
		switch p := parent.(type) {
		case []any:
			p[fakeIndex(last)] = op["value"]
		case map[string]any:
			if op["op"] == "remove" {
				delete(p, last)
			} else {
				p[last] = op["value"]
			}
		}
	}
}

func fakeIndex(field string) int {
	idx, err := strconv.Atoi(field)
	Expect(err).NotTo(HaveOccurred())
	return idx
}

func fakeWriteJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	Expect(json.NewEncoder(w).Encode(body)).To(Succeed())
}

func fakeGameDeploy(labels map[string]any, replicas, partition int64) map[string]any {
	return map[string]any{
		"apiVersion": "tkex.tencent.com/v1alpha1",
		"kind":       "GameDeployment",
		"metadata":   map[string]any{"name": "demo", "namespace": "demo-ns", "labels": labels},
		"spec": map[string]any{
			"replicas":       replicas,
			"updateStrategy": map[string]any{"partition": partition},
		},
	}
}

var _ = Describe("deleteRollout", func() {
	var (
		ctx    context.Context
		fake   *fakeCluster
		server *httptest.Server
		cfg    *cluster.Config
		record *Record
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake = newFakeCluster()
		server = httptest.NewServer(fake)
		cfg = &cluster.Config{Rest: &rest.Config{Host: server.URL}, ClusterID: "BCS-K8S-00000"}
		record = &Record{
			Namespace:    "demo-ns",
			WorkloadKind: k8skind.GameDeploy,
			ResourceKeys: ResourceKeys{{Kind: k8skind.GameDeploy, Name: "demo"}, {Kind: "Service", Name: "demo"}},
		}
	})

	AfterEach(func() {
		server.Close()
	})

	It("deletes the rollout named after the main workload", func() {
		fake.rollouts["demo"] = map[string]any{
			"apiVersion": batchrelease.APIVersion, "kind": batchrelease.Kind,
			"metadata": map[string]any{"name": "demo"},
		}

		Expect(deleteRollout(ctx, cfg, record)).To(Succeed())
		Expect(fake.rollouts).NotTo(HaveKey("demo"))
	})

	It("succeeds when the rollout or the rollout CRD does not exist", func() {
		Expect(deleteRollout(ctx, cfg, record)).To(Succeed())

		fake.rolloutInstalled = false
		Expect(deleteRollout(ctx, cfg, record)).To(Succeed())
	})

	It("skips a Deployment main workload", func() {
		record.WorkloadKind = k8skind.Deploy
		record.ResourceKeys = ResourceKeys{{Kind: k8skind.Deploy, Name: "demo"}}

		Expect(deleteRollout(ctx, cfg, record)).To(Succeed())
		Expect(fake.requestsWith("/")).To(BeEmpty())
	})
})

var _ = Describe("workloadRollout", func() {
	var (
		ctx    context.Context
		fake   *fakeCluster
		server *httptest.Server
		cfg    *cluster.Config
		meta   *metav1.ObjectMeta

		origTimeout, origInterval time.Duration
	)

	strategy := &releasestrategy.ReleaseStrategy{Batches: []releasestrategy.Batch{
		{Target: "1", PauseMode: releasestrategy.PauseModePermanent},
		{Target: "50%", PauseMode: releasestrategy.PauseModeNone},
	}}

	newRollout := func(kind string, strategy *releasestrategy.ReleaseStrategy) *workloadRollout {
		rollout, err := newWorkloadRollout(cfg, "demo-ns", kind, "demo", strategy)
		Expect(err).NotTo(HaveOccurred())
		return rollout
	}

	BeforeEach(func() {
		ctx = context.Background()
		fake = newFakeCluster()
		server = httptest.NewServer(fake)
		cfg = &cluster.Config{Rest: &rest.Config{Host: server.URL}, ClusterID: "BCS-K8S-00000"}
		meta = &metav1.ObjectMeta{Name: "demo"}

		origTimeout, origInterval = rolloutWaitTimeout, rolloutWaitInterval
		rolloutWaitTimeout, rolloutWaitInterval = 300*time.Millisecond, 10*time.Millisecond
	})

	AfterEach(func() {
		server.Close()
		rolloutWaitTimeout, rolloutWaitInterval = origTimeout, origInterval
	})

	It("rejects a Deployment main workload with a strategy without touching the cluster", func() {
		_, err := newWorkloadRollout(cfg, "demo-ns", k8skind.Deploy, "demo", strategy)
		Expect(errors.Is(err, batchrelease.ErrWorkloadNotSupported)).To(BeTrue())
		Expect(fake.requestsWith("/")).To(BeEmpty())
	})

	Context("when releasing in batches", func() {
		It("applies the rollout, labels the workload and verifies the injected partition", func() {
			fake.gameDeploys["demo"] = fakeGameDeploy(nil, 4, 0)

			rollout := newRollout(k8skind.GameDeploy, strategy)
			Expect(rollout.EnsureRollout(ctx, meta)).To(Succeed())

			Expect(fake.rollouts["demo"]).To(HaveKeyWithValue("spec", And(
				HaveKeyWithValue("type", "Canary"),
				HaveKeyWithValue("failurePolicy", "Present"),
				HaveKeyWithValue("steps", Equal([]any{
					map[string]any{"replicas": float64(1), "pause": map[string]any{}},
					map[string]any{"replicas": "50%"},
				})),
			)))
			Expect(meta.Labels).To(HaveKeyWithValue("rollout", "true"))

			injected := &unstructured.Unstructured{Object: fakeGameDeploy(nil, 4, 4)}
			Expect(rollout.EnsurePartition(ctx, injected)).To(Succeed())
			notInjected := &unstructured.Unstructured{Object: fakeGameDeploy(nil, 4, 0)}
			Expect(rollout.EnsurePartition(ctx, notInjected)).To(
				MatchError(ContainSubstring("rollout webhook did not take effect")),
			)
		})

		It("skips waiting and verifying when the workload is created for the first time", func() {
			fake.reconcile = false

			rollout := newRollout(k8skind.GameDeploy, strategy)
			Expect(rollout.EnsureRollout(ctx, meta)).To(Succeed())
			Expect(fake.rollouts).To(HaveKey("demo"))
			Expect(meta.Labels).To(HaveKeyWithValue("rollout", "true"))
			Expect(
				rollout.EnsurePartition(ctx, &unstructured.Unstructured{Object: fakeGameDeploy(nil, 4, 0)}),
			).To(Succeed())
		})

		It("fails when the controller does not reconcile the rollout in time", func() {
			fake.reconcile = false
			fake.gameDeploys["demo"] = fakeGameDeploy(nil, 4, 0)

			err := newRollout(k8skind.GameDeploy, strategy).EnsureRollout(ctx, meta)
			Expect(err).To(MatchError(ContainSubstring("wait rollout demo-ns/demo to be reconciled")))
		})

		Context("with a leftover rollout reconciled by a previous release", func() {
			BeforeEach(func() {
				fake.gameDeploys["demo"] = fakeGameDeploy(map[string]any{"rollout": "true"}, 4, 0)
				previous := &releasestrategy.ReleaseStrategy{Batches: []releasestrategy.Batch{
					{Target: "3", PauseMode: releasestrategy.PauseModePermanent},
				}}
				Expect(newRollout(k8skind.GameDeploy, previous).EnsureRollout(ctx, meta)).To(Succeed())
				Expect(fake.rollouts["demo"]["status"]).To(HaveKeyWithValue("rolloutRevision", "demo-rev-1"))
			})

			It("waits for the controller to observe the changed spec", func() {
				fake.observeSpecAfterGets = 3

				Expect(newRollout(k8skind.GameDeploy, strategy).EnsureRollout(ctx, meta)).To(Succeed())
				// workloadRevision 早已写入，只有等到 rolloutRevision 更新才能返回
				Expect(fake.pendingGets).To(BeZero())
				Expect(fake.rollouts["demo"]["status"]).To(HaveKeyWithValue("rolloutRevision", "demo-rev-2"))
			})

			It("fails when the controller does not observe the changed spec in time", func() {
				fake.observeSpecAfterGets = 1000

				err := newRollout(k8skind.GameDeploy, strategy).EnsureRollout(ctx, meta)
				Expect(err).To(MatchError(ContainSubstring("wait rollout demo-ns/demo to be reconciled")))
			})

			It("does not wait for the rollout revision when the spec is unchanged", func() {
				Expect(newRollout(k8skind.GameDeploy, strategy).EnsureRollout(ctx, meta)).To(Succeed())
				fake.observeSpecAfterGets = 1000

				Expect(newRollout(k8skind.GameDeploy, strategy).EnsureRollout(ctx, meta)).To(Succeed())
				Expect(fake.rollouts["demo"]["metadata"]).To(HaveKeyWithValue("generation", BeNumerically("==", 2)))
			})
		})

		It("fails with the cluster reason when the rollout CRD is not installed", func() {
			fake.rolloutInstalled = false
			fake.gameDeploys["demo"] = fakeGameDeploy(nil, 4, 0)

			err := newRollout(k8skind.GameDeploy, strategy).EnsureRollout(ctx, meta)
			Expect(errors.Is(err, batchrelease.ErrRolloutNotInstalled)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("the server could not find the requested resource"))
			Expect(meta.Labels).NotTo(HaveKey("rollout"))
		})

		It("appends the rollout to the snapshot objects without modifying the extra objects", func() {
			extraObjs := make([]unstructured.Unstructured, 1, 2)
			extraObjs[0] = unstructured.Unstructured{Object: map[string]any{"kind": "Service"}}

			objs := newRollout(k8skind.GameDeploy, strategy).SnapshotObjects(extraObjs)
			Expect(objs).To(HaveLen(2))
			Expect(objs[1].GetKind()).To(Equal(batchrelease.Kind))
			Expect(extraObjs[:2][1].Object).To(BeNil())
		})
	})

	Context("when updating all instances at once", func() {
		It("deletes the leftover rollout, removes the label and resets the partition", func() {
			fake.rollouts["demo"] = map[string]any{
				"apiVersion": batchrelease.APIVersion, "kind": batchrelease.Kind,
				"metadata": map[string]any{"name": "demo"},
			}
			fake.gameDeploys["demo"] = fakeGameDeploy(map[string]any{"rollout": "true", "app": "demo"}, 4, 0)

			rollout := newRollout(k8skind.GameDeploy, nil)
			Expect(rollout.EnsureRollout(ctx, meta)).To(Succeed())
			Expect(fake.rollouts).NotTo(HaveKey("demo"))
			labels := fake.gameDeploys["demo"]["metadata"].(map[string]any)["labels"]
			Expect(labels).To(Equal(map[string]any{"app": "demo"}))

			// webhook 缓存未同步时仍注入了 partition，更新后需放开
			fake.gameDeploys["demo"] = fakeGameDeploy(nil, 4, 4)
			Expect(
				rollout.EnsurePartition(ctx, &unstructured.Unstructured{Object: fakeGameDeploy(nil, 4, 4)}),
			).To(Succeed())
			updateStrategy := fake.gameDeploys["demo"]["spec"].(map[string]any)["updateStrategy"]
			Expect(updateStrategy).To(HaveKeyWithValue("partition", BeNumerically("==", 0)))
		})

		It("does nothing when there is no leftover rollout, label or partition", func() {
			fake.gameDeploys["demo"] = fakeGameDeploy(map[string]any{"app": "demo"}, 4, 0)

			rollout := newRollout(k8skind.GameDeploy, nil)
			Expect(rollout.EnsureRollout(ctx, meta)).To(Succeed())
			Expect(
				rollout.EnsurePartition(ctx, &unstructured.Unstructured{Object: fakeGameDeploy(nil, 4, 0)}),
			).To(Succeed())
			Expect(fake.requestsWith(http.MethodDelete)).To(BeEmpty())
			Expect(fake.requestsWith(http.MethodPatch)).To(BeEmpty())
		})

		It("resets the partition even if the rollout was deleted by a previous attempt", func() {
			fake.gameDeploys["demo"] = fakeGameDeploy(map[string]any{"app": "demo"}, 4, 4)

			rollout := newRollout(k8skind.GameDeploy, nil)
			Expect(rollout.EnsureRollout(ctx, meta)).To(Succeed())
			Expect(
				rollout.EnsurePartition(ctx, &unstructured.Unstructured{Object: fakeGameDeploy(nil, 4, 4)}),
			).To(Succeed())
			Expect(fake.requestsWith(http.MethodDelete)).To(BeEmpty())
			updateStrategy := fake.gameDeploys["demo"]["spec"].(map[string]any)["updateStrategy"]
			Expect(updateStrategy).To(HaveKeyWithValue("partition", BeNumerically("==", 0)))
		})

		It("treats a cluster without the rollout CRD as having nothing to clean up", func() {
			fake.rolloutInstalled = false

			Expect(newRollout(k8skind.GameDeploy, nil).EnsureRollout(ctx, meta)).To(Succeed())
			Expect(fake.requestsWith(http.MethodDelete)).To(BeEmpty())
		})

		It("fails without touching the workload when deleting the rollout fails", func() {
			fake.failRolloutDelete = true
			fake.rollouts["demo"] = map[string]any{
				"apiVersion": batchrelease.APIVersion, "kind": batchrelease.Kind,
				"metadata": map[string]any{"name": "demo"},
			}
			fake.gameDeploys["demo"] = fakeGameDeploy(map[string]any{"rollout": "true"}, 4, 0)

			err := newRollout(k8skind.GameDeploy, nil).EnsureRollout(ctx, meta)
			Expect(err).To(MatchError(ContainSubstring("etcd unavailable")))
			Expect(fake.requestsWith(fakeGameDeploysPath)).To(BeEmpty())
		})

		It("keeps the extra objects as the snapshot objects", func() {
			extraObjs := []unstructured.Unstructured{{Object: map[string]any{"kind": "Service"}}}
			Expect(newRollout(k8skind.GameDeploy, nil).SnapshotObjects(extraObjs)).To(Equal(extraObjs))
		})
	})

	It("does nothing for a Deployment main workload", func() {
		rollout := newRollout(k8skind.Deploy, nil)
		Expect(rollout.EnsureRollout(ctx, meta)).To(Succeed())
		Expect(rollout.EnsurePartition(ctx, nil)).To(Succeed())
		Expect(fake.requestsWith("/")).To(BeEmpty())
	})
})

var _ = Describe("UpdateInstances with a leftover rollout", func() {
	var (
		ctx      context.Context
		fake     *fakeCluster
		server   *httptest.Server
		mocker   *mockey.Mocker
		deployer *Deployer
	)

	leftoverRollout := func() map[string]any {
		return map[string]any{
			"apiVersion": batchrelease.APIVersion, "kind": batchrelease.Kind,
			"metadata": map[string]any{"name": "demo"},
		}
	}
	// gameDeployWithImage 返回包含主容器镜像的 GameDeployment
	gameDeployWithImage := func(labels map[string]any) map[string]any {
		obj := fakeGameDeploy(labels, 4, 0)
		obj["spec"].(map[string]any)["template"] = map[string]any{"spec": map[string]any{
			"containers": []any{map[string]any{"name": "main", "image": "repo/demo:v1"}},
		}}
		return obj
	}
	gameDeploySpec := func() map[string]any {
		return fake.gameDeploys["demo"]["spec"].(map[string]any)
	}
	mainImage := func() any {
		containers := gameDeploySpec()["template"].(map[string]any)["spec"].(map[string]any)["containers"]
		return containers.([]any)[0].(map[string]any)["image"]
	}

	BeforeEach(func() {
		ctx = auth.WithUser(context.Background(), auth.User{ID: "tester"})
		fake = newFakeCluster()
		server = httptest.NewServer(fake)
		cfg := &cluster.Config{Rest: &rest.Config{Host: server.URL}, ClusterID: "BCS-K8S-00000"}
		mocker = mockey.Mock(cluster.NewConfig).Return(cfg).Build()

		db := testutil.NewDB(GinkgoT())
		recordStore := store.Of(db, RecordMeta)
		appID := "rollout-inst-update-" + stringx.Random(6)
		DeferCleanup(func() {
			_, _ = recordStore.DeleteMany(context.Background(), store.Where("appID", appID))
			_, _ = store.Of(db, ResourceSnapshotMeta).DeleteMany(context.Background(), store.Where("appID", appID))
		})
		err := recordStore.Create(ctx, &Record{
			AppID:         appID,
			EnvName:       "prod",
			ClusterID:     "BCS-K8S-00000",
			Namespace:     "demo-ns",
			ImageTag:      "v1",
			Replicas:      4,
			LabelSelector: map[string]string{"app": "demo"},
			WorkloadKind:  k8skind.GameDeploy,
			ResourceKeys:  ResourceKeys{{Kind: k8skind.GameDeploy, Name: "demo"}},
			Status:        StatusDeployed,
		})
		Expect(err).NotTo(HaveOccurred())
		deployer = &Deployer{db: db, app: &bkmsapp.Application{ID: appID}, podManager: &podManager{}}
	})

	AfterEach(func() {
		mocker.UnPatch()
		server.Close()
	})

	It("cleans up the rollout before updating all instances and releases the injected partition", func() {
		fake.staleWebhook = true
		fake.rollouts["demo"] = leftoverRollout()
		fake.gameDeploys["demo"] = gameDeployWithImage(map[string]any{"rollout": "true", "app": "demo"})

		deployID, err := deployer.UpdateInstances(ctx, "prod", "", "v2", "InplaceUpdate", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(deployID).NotTo(BeEmpty())

		Expect(fake.rollouts).NotTo(HaveKey("demo"))
		Expect(
			fake.gameDeploys["demo"]["metadata"].(map[string]any)["labels"],
		).To(Equal(map[string]any{"app": "demo"}))
		Expect(mainImage()).To(Equal("repo/demo:v2"))
		Expect(gameDeploySpec()["updateStrategy"]).To(HaveKeyWithValue("partition", BeNumerically("==", 0)))
	})

	It("restores the grayscale partition after cleaning up the rollout", func() {
		fake.staleWebhook = true
		fake.rollouts["demo"] = leftoverRollout()
		fake.gameDeploys["demo"] = gameDeployWithImage(map[string]any{"rollout": "true"})
		for _, name := range []string{"demo-0", "demo-1", "demo-2", "demo-3"} {
			fake.pods[name] = map[string]any{
				"apiVersion": "v1", "kind": "Pod",
				"metadata": map[string]any{"name": name, "annotations": map[string]any{}},
				"spec": map[string]any{
					"containers": []any{map[string]any{"name": "main", "image": "repo/demo:v1"}},
				},
				"status": map[string]any{"phase": "Running"},
			}
		}

		deployID, err := deployer.UpdateInstances(ctx, "prod", "", "v2", "", []string{"demo-0"})
		Expect(err).NotTo(HaveOccurred())
		Expect(deployID).To(BeEmpty())

		Expect(fake.rollouts).NotTo(HaveKey("demo"))
		Expect(mainImage()).To(Equal("repo/demo:v2"))
		// 4 个实例灰度 1 个，保留 3 个旧版本
		Expect(gameDeploySpec()["updateStrategy"]).To(HaveKeyWithValue("partition", BeNumerically("==", 3)))
	})

	It("releases the injected partition when retrying after a previous attempt deleted the rollout", func() {
		// 上一次尝试已删除 Rollout 与标签，但纠正 partition 前失败；webhook 缓存仍未同步
		fake.staleWebhook = true
		fake.gameDeploys["demo"] = gameDeployWithImage(map[string]any{"app": "demo"})

		_, err := deployer.UpdateInstances(ctx, "prod", "", "v2", "", nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(mainImage()).To(Equal("repo/demo:v2"))
		Expect(gameDeploySpec()["updateStrategy"]).To(HaveKeyWithValue("partition", BeNumerically("==", 0)))
		Expect(fake.requestsWith(http.MethodPatch)).To(HaveLen(2))
	})

	It("resets the partition left by a grayscale update on a rolling full update", func() {
		fake.gameDeploys["demo"] = gameDeployWithImage(map[string]any{"app": "demo"})
		gameDeploySpec()["updateStrategy"].(map[string]any)["partition"] = int64(3)

		_, err := deployer.UpdateInstances(ctx, "prod", "", "v2", "RollingUpdate", nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(gameDeploySpec()["updateStrategy"]).To(And(
			HaveKeyWithValue("type", "RollingUpdate"),
			HaveKeyWithValue("partition", BeNumerically("==", 0)),
		))
		Expect(fake.requestsWith(http.MethodPatch)).To(HaveLen(1))
	})

	It("only patches the image without a leftover rollout", func() {
		fake.gameDeploys["demo"] = gameDeployWithImage(map[string]any{"app": "demo"})

		_, err := deployer.UpdateInstances(ctx, "prod", "", "v2", "", nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(mainImage()).To(Equal("repo/demo:v2"))
		Expect(fake.requestsWith(http.MethodDelete)).To(BeEmpty())
		Expect(fake.requestsWith(http.MethodPatch)).To(HaveLen(1))
	})

	It("fails without patching the workload when deleting the rollout fails", func() {
		fake.failRolloutDelete = true
		fake.rollouts["demo"] = leftoverRollout()
		fake.gameDeploys["demo"] = gameDeployWithImage(map[string]any{"rollout": "true"})

		_, err := deployer.UpdateInstances(ctx, "prod", "", "v2", "", nil)
		Expect(err).To(MatchError(ContainSubstring("clean up rollout")))
		Expect(mainImage()).To(Equal("repo/demo:v1"))
		Expect(fake.requestsWith(http.MethodPatch)).To(BeEmpty())
	})
})
