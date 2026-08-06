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

package bscpcfg_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	extmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

// mockStore 是一个最小化的 Store mock，仅实现 GetSnapshot 方法。
type mockStore struct {
	extmodel.Store
	snapshot *extmodel.Snapshot
	err      error
}

func (m *mockStore) GetSnapshot(_ context.Context, _, _ string) (*extmodel.Snapshot, error) {
	return m.snapshot, m.err
}

var _ = Describe("MergePodSpec", func() {
	Describe("when fragment is nil", func() {
		It("should not modify the PodSpec and return nil", func() {
			podSpec := &corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "existing-init"}},
				Containers: []corev1.Container{
					{Name: "main", VolumeMounts: []corev1.VolumeMount{{Name: "app-vol", MountPath: "/app"}}},
				},
				Volumes: []corev1.Volume{{Name: "existing-vol"}},
			}

			err := bscpcfg.MergePodSpec(podSpec, nil, "main")

			Expect(err).NotTo(HaveOccurred())
			Expect(podSpec.InitContainers).To(HaveLen(1))
			Expect(podSpec.InitContainers[0].Name).To(Equal("existing-init"))
			Expect(podSpec.Containers).To(HaveLen(1))
			Expect(podSpec.Volumes).To(HaveLen(1))
		})
	})

	Describe("when podSpec is nil", func() {
		It("should return ErrPodSpecNil", func() {
			fragment := &bscpcfg.PodFragment{
				InitContainers: []corev1.Container{{Name: bscpcfg.InitContainerName}},
			}

			err := bscpcfg.MergePodSpec(nil, fragment, "main")

			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(bscpcfg.ErrPodSpecNil))
		})
	})

	Describe("when main container is not found in PodSpec", func() {
		It("should return ErrMainContainerNotFound with container name", func() {
			podSpec := &corev1.PodSpec{
				Containers: []corev1.Container{{Name: "sidecar-only"}},
			}
			fragment := &bscpcfg.PodFragment{
				InitContainers: []corev1.Container{{Name: bscpcfg.InitContainerName}},
			}

			err := bscpcfg.MergePodSpec(podSpec, fragment, "main")

			Expect(err).To(HaveOccurred())
			Expect(errors.Cause(err)).To(Equal(bscpcfg.ErrMainContainerNotFound))
			Expect(err.Error()).To(ContainSubstring("main"))
		})
	})

	Describe("idempotency - already injected", func() {
		It("should skip all injections when already present (including volume)", func() {
			podSpec := &corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: bscpcfg.InitContainerName}},
				Containers: []corev1.Container{
					{
						Name: "main",
						VolumeMounts: []corev1.VolumeMount{
							{Name: bscpcfg.ShareVolumeName, MountPath: "/data/bscp"},
						},
					},
					{Name: bscpcfg.SidecarContainerName},
				},
				Volumes: []corev1.Volume{
					{Name: bscpcfg.VolumeName},
					{Name: bscpcfg.ShareVolumeName},
				},
			}
			fragment := &bscpcfg.PodFragment{
				InitContainers: []corev1.Container{{Name: bscpcfg.InitContainerName}},
				Containers:     []corev1.Container{{Name: bscpcfg.SidecarContainerName}},
				Volumes: []corev1.Volume{
					{Name: bscpcfg.VolumeName},
					{Name: bscpcfg.ShareVolumeName},
				},
				MainContainerVolumeMounts: []corev1.VolumeMount{
					{Name: bscpcfg.ShareVolumeName, MountPath: "/data/bscp"},
				},
			}

			err := bscpcfg.MergePodSpec(podSpec, fragment, "main")

			Expect(err).NotTo(HaveOccurred())
			Expect(podSpec.InitContainers).To(HaveLen(1))
			Expect(podSpec.Containers).To(HaveLen(2))
			Expect(podSpec.Volumes).To(HaveLen(2))
			Expect(podSpec.Containers[0].VolumeMounts).To(HaveLen(1))
		})
	})

	Describe("normal injection with existing fields", func() {
		It("should merge all fragment fields into the PodSpec without overwriting existing ones", func() {
			podSpec := &corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "existing-init"}},
				Containers: []corev1.Container{
					{Name: "main", VolumeMounts: []corev1.VolumeMount{{Name: "app-vol", MountPath: "/app"}}},
				},
				Volumes: []corev1.Volume{{Name: "existing-vol"}},
			}

			fragment := &bscpcfg.PodFragment{
				InitContainers: []corev1.Container{{Name: bscpcfg.InitContainerName}},
				Containers:     []corev1.Container{{Name: bscpcfg.SidecarContainerName}},
				Volumes: []corev1.Volume{
					{Name: bscpcfg.VolumeName},
					{Name: bscpcfg.ShareVolumeName},
				},
				MainContainerVolumeMounts: []corev1.VolumeMount{
					{Name: bscpcfg.ShareVolumeName, MountPath: "/data/bscp"},
				},
			}

			err := bscpcfg.MergePodSpec(podSpec, fragment, "main")

			Expect(err).NotTo(HaveOccurred())

			By("InitContainers are appended")
			Expect(podSpec.InitContainers).To(HaveLen(2))
			Expect(podSpec.InitContainers[0].Name).To(Equal("existing-init"))
			Expect(podSpec.InitContainers[1].Name).To(Equal(bscpcfg.InitContainerName))

			By("Sidecar containers are appended after main container")
			Expect(podSpec.Containers).To(HaveLen(2))
			Expect(podSpec.Containers[0].Name).To(Equal("main"))
			Expect(podSpec.Containers[1].Name).To(Equal(bscpcfg.SidecarContainerName))

			By("Volumes are appended (bscp-temp + bscp-share)")
			Expect(podSpec.Volumes).To(HaveLen(3))
			Expect(podSpec.Volumes[0].Name).To(Equal("existing-vol"))
			Expect(podSpec.Volumes[1].Name).To(Equal(bscpcfg.VolumeName))
			Expect(podSpec.Volumes[2].Name).To(Equal(bscpcfg.ShareVolumeName))

			By("Main container VolumeMounts are appended (bscp-share -> user MountPath)")
			Expect(podSpec.Containers[0].VolumeMounts).To(HaveLen(2))
			Expect(podSpec.Containers[0].VolumeMounts[0].Name).To(Equal("app-vol"))
			Expect(podSpec.Containers[0].VolumeMounts[1].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(podSpec.Containers[0].VolumeMounts[1].MountPath).To(Equal("/data/bscp"))
		})
	})
})

var _ = Describe("InjectFromStore", func() {
	var (
		ctx     context.Context
		podSpec *corev1.PodSpec
	)

	BeforeEach(func() {
		ctx = context.Background()
		podSpec = &corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main"}},
		}
	})

	Describe("when store returns nil snapshot (not configured)", func() {
		It("should not modify the PodSpec", func() {
			store := &mockStore{snapshot: nil, err: nil}

			err := bscpcfg.InjectFromStore(ctx, store, "app-1", "dev", "main", podSpec)

			Expect(err).NotTo(HaveOccurred())
			Expect(podSpec.InitContainers).To(BeEmpty())
			Expect(podSpec.Containers).To(HaveLen(1))
			Expect(podSpec.Volumes).To(BeEmpty())
		})
	})

	Describe("when store returns an error", func() {
		It("should propagate the error", func() {
			store := &mockStore{snapshot: nil, err: errors.New("db connection failed")}

			err := bscpcfg.InjectFromStore(ctx, store, "app-1", "dev", "main", podSpec)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db connection failed"))
		})
	})

	Describe("when store returns a valid snapshot", func() {
		It("should merge bscp config artifacts into the PodSpec", func() {
			store := &mockStore{
				snapshot: &extmodel.Snapshot{
					Metadata: &extmodel.Metadata{
						AppID:        "app-1",
						BscpBizID:    "100",
						MountPath:    "/data/bscp",
						FeedAddr:     "feed.example.com:9510",
						Token:        "test-token",
						WorkloadName: "main",
					},
					EnvBinding: &extmodel.EnvBinding{
						AppID:   "app-1",
						EnvName: "dev",
						Services: []extmodel.ServiceRef{
							{ID: "svc-1", Name: "order-svc"},
							{ID: "svc-2", Name: "user-svc"},
						},
					},
				},
			}

			err := bscpcfg.InjectFromStore(ctx, store, "app-1", "dev", "main", podSpec)

			Expect(err).NotTo(HaveOccurred())
			Expect(podSpec.InitContainers).To(HaveLen(1))
			Expect(podSpec.InitContainers[0].Name).To(Equal(bscpcfg.InitContainerName))
			Expect(podSpec.Containers).To(HaveLen(2))
			Expect(podSpec.Containers[1].Name).To(Equal(bscpcfg.SidecarContainerName))
			Expect(podSpec.Volumes).To(HaveLen(2))
			Expect(podSpec.Volumes[0].Name).To(Equal(bscpcfg.VolumeName))
			Expect(podSpec.Volumes[1].Name).To(Equal(bscpcfg.ShareVolumeName))
			// 主容器挂载 bscp-share 到 MountPath
			Expect(podSpec.Containers[0].VolumeMounts).To(HaveLen(1))
			Expect(podSpec.Containers[0].VolumeMounts[0].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(podSpec.Containers[0].VolumeMounts[0].MountPath).To(Equal("/data/bscp"))
		})
	})

	Describe("when store returns a valid snapshot - full field verification for debug", func() {
		It("should inject all fields correctly (image, args, env, volumeMounts, volumes)", func() {
			store := &mockStore{
				snapshot: &extmodel.Snapshot{
					Metadata: &extmodel.Metadata{
						AppID:        "debug-app",
						BscpBizID:    "200",
						MountPath:    "/data/app/config",
						FeedAddr:     "feed.bscp.svc:9510",
						Token:        "my-secret-token",
						WorkloadName: "main",
					},
					EnvBinding: &extmodel.EnvBinding{
						AppID:   "debug-app",
						EnvName: "prod",
						Services: []extmodel.ServiceRef{
							{ID: "svc-a", Name: "config-svc"},
							{ID: "svc-b", Name: "secret-svc"},
						},
					},
				},
			}

			err := bscpcfg.InjectFromStore(ctx, store, "debug-app", "prod", "main", podSpec)

			Expect(err).NotTo(HaveOccurred())

			By("init container count and name")
			Expect(podSpec.InitContainers).To(HaveLen(1))
			initC := podSpec.InitContainers[0]
			Expect(initC.Name).To(Equal(bscpcfg.InitContainerName))

			By("init container image")
			Expect(initC.Image).To(Equal(bscpcfg.InitImage))

			By("init container args")
			Expect(initC.Args).To(ConsistOf("--file-cache-enabled=false"))

			By("init container env vars")
			Expect(initC.Env).To(HaveLen(5))
			Expect(initC.Env).To(ContainElements(
				corev1.EnvVar{Name: "biz", Value: "200"},
				corev1.EnvVar{Name: "app", Value: "config-svc,secret-svc"},
				corev1.EnvVar{Name: "feed_addrs", Value: "feed.bscp.svc:9510"},
				corev1.EnvVar{Name: "token", Value: "my-secret-token"},
				corev1.EnvVar{Name: "temp_dir", Value: bscpcfg.BscpDownloadPath},
			))

			By("init container volumeMounts")
			Expect(initC.VolumeMounts).To(HaveLen(1))
			Expect(initC.VolumeMounts[0].Name).To(Equal(bscpcfg.VolumeName))
			Expect(initC.VolumeMounts[0].MountPath).To(Equal(bscpcfg.BscpDownloadPath))

			By("sidecar container")
			Expect(podSpec.Containers).To(HaveLen(2))
			sidecar := podSpec.Containers[1]
			Expect(sidecar.Name).To(Equal(bscpcfg.SidecarContainerName))
			Expect(sidecar.Image).To(Equal(bscpcfg.SidecarImage))
			Expect(sidecar.Args).To(ConsistOf("--file-cache-enabled=false"))

			By("sidecar env vars should match init container")
			Expect(sidecar.Env).To(HaveLen(5))
			Expect(sidecar.Env).To(ContainElements(
				corev1.EnvVar{Name: "biz", Value: "200"},
				corev1.EnvVar{Name: "app", Value: "config-svc,secret-svc"},
				corev1.EnvVar{Name: "feed_addrs", Value: "feed.bscp.svc:9510"},
				corev1.EnvVar{Name: "token", Value: "my-secret-token"},
				corev1.EnvVar{Name: "temp_dir", Value: bscpcfg.BscpDownloadPath},
			))

			By("sidecar volumeMounts: bscp-temp + bscp-share")
			Expect(sidecar.VolumeMounts).To(HaveLen(2))
			Expect(sidecar.VolumeMounts[0].Name).To(Equal(bscpcfg.VolumeName))
			Expect(sidecar.VolumeMounts[0].MountPath).To(Equal(bscpcfg.BscpDownloadPath))
			Expect(sidecar.VolumeMounts[1].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(sidecar.VolumeMounts[1].MountPath).To(Equal(bscpcfg.BscpShareBasePath))

			By("volumes: bscp-temp + bscp-share both emptyDir")
			Expect(podSpec.Volumes).To(HaveLen(2))
			Expect(podSpec.Volumes[0].Name).To(Equal(bscpcfg.VolumeName))
			Expect(podSpec.Volumes[0].VolumeSource.EmptyDir).NotTo(BeNil())
			Expect(podSpec.Volumes[1].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(podSpec.Volumes[1].VolumeSource.EmptyDir).NotTo(BeNil())

			By("main container volumeMount: bscp-share mounted to user-specified MountPath")
			mainC := podSpec.Containers[0]
			Expect(mainC.VolumeMounts).To(HaveLen(1))
			Expect(mainC.VolumeMounts[0].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(mainC.VolumeMounts[0].MountPath).To(Equal("/data/app/config"))
		})
	})

	Describe("when snapshot has invalid metadata (validation fails)", func() {
		It("should return a validation error", func() {
			store := &mockStore{
				snapshot: &extmodel.Snapshot{
					Metadata: &extmodel.Metadata{
						AppID:     "app-1",
						BscpBizID: "100",
						MountPath: "",
						FeedAddr:  "",
						Token:     "",
					},
					EnvBinding: &extmodel.EnvBinding{
						AppID:   "app-1",
						EnvName: "dev",
						Services: []extmodel.ServiceRef{
							{ID: "svc-1", Name: "order-svc"},
						},
					},
				},
			}

			err := bscpcfg.InjectFromStore(ctx, store, "app-1", "dev", "main", podSpec)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validating bscp config snapshot"))
		})
	})
})
