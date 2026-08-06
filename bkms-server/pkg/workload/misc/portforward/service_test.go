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

package portforward

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Service", func() {
	var (
		ctx   context.Context
		store appmodeldeploy.RecordStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		store, err = appmodeldeploy.NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	It("validates a running pod from latest deploy record", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:         "myapp",
			EnvName:       "test",
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-12345",
			Namespace:     "default",
			LabelSelector: map[string]string{"app": "myapp"},
		})
		Expect(err).NotTo(HaveOccurred())

		pod := runningPodWithLabels("pod-1", "10.0.0.1", map[string]string{"app": "myapp"})
		podClient := &MockPodClient{}
		podClient.On("Get", mock.Anything, "default", "pod-1", mock.Anything).Return(&pod, nil)
		svc := NewService(store, WithPodClientFactory(func(clusterID string) PodClient {
			Expect(clusterID).To(Equal("BCS-K8S-12345"))
			return podClient
		}))

		target, err := svc.ResolveTarget(ctx, "myapp", "test", "pod-1", 8080)

		Expect(err).NotTo(HaveOccurred())
		Expect(target.ClusterID).To(Equal("BCS-K8S-12345"))
		Expect(target.Namespace).To(Equal("default"))
		Expect(target.PodName).To(Equal("pod-1"))
		Expect(target.PodIP).To(Equal("10.0.0.1"))
		Expect(target.RemotePort).To(Equal(int32(8080)))
		podClient.AssertCalled(GinkgoT(), "Get", mock.Anything, "default", "pod-1", mock.Anything)
	})

	It("rejects a running pod without pod IP", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:         "myapp",
			EnvName:       "test",
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-12345",
			Namespace:     "default",
			LabelSelector: map[string]string{"app": "myapp"},
		})
		Expect(err).NotTo(HaveOccurred())

		pod := podWithPhaseIPAndLabels("pod-1", "Running", "", map[string]string{"app": "myapp"})
		svc := NewService(store, WithPodClientFactory(func(clusterID string) PodClient {
			m := &MockPodClient{}
			m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&pod, nil)
			return m
		}))

		_, err = svc.ResolveTarget(ctx, "myapp", "test", "pod-1", 8080)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pod IP is empty"))
	})

	It("rejects a pod with invalid pod IP", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:         "myapp",
			EnvName:       "test",
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-12345",
			Namespace:     "default",
			LabelSelector: map[string]string{"app": "myapp"},
		})
		Expect(err).NotTo(HaveOccurred())

		pod := podWithPhaseIPAndLabels("pod-1", "Running", "not-an-ip", map[string]string{"app": "myapp"})
		svc := NewService(store, WithPodClientFactory(func(clusterID string) PodClient {
			m := &MockPodClient{}
			m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&pod, nil)
			return m
		}))

		_, err = svc.ResolveTarget(ctx, "myapp", "test", "pod-1", 8080)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pod IP is invalid"))
	})

	It("rejects a pod with forbidden pod IP", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:         "myapp",
			EnvName:       "test",
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-12345",
			Namespace:     "default",
			LabelSelector: map[string]string{"app": "myapp"},
		})
		Expect(err).NotTo(HaveOccurred())

		pod := podWithPhaseIPAndLabels("pod-1", "Running", "127.0.0.1", map[string]string{"app": "myapp"})
		svc := NewService(store, WithPodClientFactory(func(clusterID string) PodClient {
			m := &MockPodClient{}
			m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&pod, nil)
			return m
		}))

		_, err = svc.ResolveTarget(ctx, "myapp", "test", "pod-1", 8080)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pod IP is forbidden"))
	})

	It("returns a controlled error when deploy record lookup fails", func() {
		// 不插入任何记录，GetLatest 应返回 not found 错误
		svc := NewService(store)

		_, err := svc.ResolveTarget(ctx, "nonexistent-app", "test", "pod-1", 8080)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("deploy record not found"))
	})

	It("returns a controlled error when pod get fails", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:         "myapp",
			EnvName:       "test",
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-12345",
			Namespace:     "default",
			LabelSelector: map[string]string{"app": "myapp"},
		})
		Expect(err).NotTo(HaveOccurred())

		svc := NewService(store, WithPodClientFactory(func(clusterID string) PodClient {
			m := &MockPodClient{}
			m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, errors.New("kube internal error"))
			return m
		}))

		_, err = svc.ResolveTarget(ctx, "myapp", "test", "pod-1", 8080)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("get pod 'pod-1' in namespace 'default'"))
	})

	It("rejects a pod that does not belong to current application environment", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:         "myapp",
			EnvName:       "test",
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-12345",
			Namespace:     "default",
			LabelSelector: map[string]string{"app": "myapp"},
		})
		Expect(err).NotTo(HaveOccurred())

		// Pod 存在但 labels 不匹配部署记录的 LabelSelector
		pod := runningPodWithLabels("pod-1", "127.0.0.1", map[string]string{"app": "other-app"})
		svc := NewService(store, WithPodClientFactory(func(clusterID string) PodClient {
			m := &MockPodClient{}
			m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&pod, nil)
			return m
		}))

		_, err = svc.ResolveTarget(ctx, "myapp", "test", "pod-1", 8080)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pod does not belong to current application environment"))
	})

	It("rejects a pod that is not running", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:         "myapp",
			EnvName:       "test",
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-12345",
			Namespace:     "default",
			LabelSelector: map[string]string{"app": "myapp"},
		})
		Expect(err).NotTo(HaveOccurred())

		pod := podWithPhaseIPAndLabels("pod-1", "Pending", "127.0.0.1", map[string]string{"app": "myapp"})
		svc := NewService(store, WithPodClientFactory(func(clusterID string) PodClient {
			m := &MockPodClient{}
			m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&pod, nil)
			return m
		}))

		_, err = svc.ResolveTarget(ctx, "myapp", "test", "pod-1", 8080)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pod is not running"))
	})
})

func runningPodWithLabels(name, podIP string, podLabels map[string]string) unstructured.Unstructured {
	return podWithPhaseIPAndLabels(name, "Running", podIP, podLabels)
}

func podWithPhaseIPAndLabels(name, phase, podIP string, podLabels map[string]string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"name":   name,
			"labels": toAnyMap(podLabels),
		},
		"spec": map[string]any{"containers": []any{
			map[string]any{"name": "main", "ports": []any{
				map[string]any{"containerPort": int64(8080)},
			}},
		}},
		"status": map[string]any{
			"phase": phase,
			"podIP": podIP,
		},
	}}
}

func toAnyMap(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
