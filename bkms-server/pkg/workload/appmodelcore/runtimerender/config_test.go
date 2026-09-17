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

package runtimerender_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/cfgrender"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/runtimerender"
)

var _ = Describe("BuildSedCommand (legacy per-file format)", func() {
	It("should copy template config and replace runtime variable placeholders", func() {
		command := runtimerender.BuildSedCommand("/config-template/app.yaml", "/config-rendered/app.yaml")

		Expect(command).To(Equal(
			"cp '/config-template/app.yaml' '/config-rendered/app.yaml' && " +
				"sed -i 's/__#VAR_PLACEHOLDER#__BKMS_POD_IP__/'\"$BKMS_POD_IP\"'/g' '/config-rendered/app.yaml' && " +
				"sed -i 's/__#VAR_PLACEHOLDER#__BKMS_POD_NAME__/'\"$BKMS_POD_NAME\"'/g' '/config-rendered/app.yaml' && " +
				"sed -i 's/__#VAR_PLACEHOLDER#__BKMS_NODE_IP__/'\"$BKMS_NODE_IP\"'/g' '/config-rendered/app.yaml'",
		))
	})
})

var _ = Describe("Config nil receiver", func() {
	It("should return nil from Storage when Config is nil", func() {
		var cfg *runtimerender.Config
		mounts, volumes, err := cfg.Storage(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(mounts).To(BeNil())
		Expect(volumes).To(BeNil())
	})

	It("should return nil from ExtraResources when Config is nil", func() {
		var cfg *runtimerender.Config
		objs, err := cfg.ExtraResources(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(objs).To(BeNil())
	})

	It("should return nil from InitContainers when Config is nil", func() {
		var cfg *runtimerender.Config
		containers, err := cfg.InitContainers(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(containers).To(BeNil())
	})

	It("should return nil from ExtraResources when ConfigMap name is empty", func() {
		cfg := &runtimerender.Config{}
		objs, err := cfg.ExtraResources(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(objs).To(BeNil())
	})
})

var _ = Describe("BuildConfig", func() {
	It("should build single-file config rendering resources", func() {
		result, err := runtimerender.BuildConfig(runtimerender.ConfigParams{
			WorkloadType:  "trpc",
			ConfigMapName: "demo-app",
			Files: []cfgrender.RenderedMountableFile{
				{FileName: "app.yaml", FilePath: "/etc/app", FileContent: "server:\n  app: demo\n"},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.ConfigMap.Name).To(Equal("demo-app"))
		Expect(result.ConfigMap.Data).To(HaveKeyWithValue("00-app.yaml", "server:\n  app: demo\n"))
		Expect(result.MainContainerMounts).To(HaveLen(1))
		Expect(result.MainContainerMounts[0].Name).To(Equal("trpc-config-rendered"))
		Expect(result.MainContainerMounts[0].MountPath).To(Equal("/etc/app/app.yaml"))
		Expect(result.Volumes).To(HaveLen(2))
		Expect(result.Volumes[0].Name).To(Equal("trpc-config-template"))
		Expect(result.Volumes[1].Name).To(Equal("trpc-config-rendered"))
		Expect(result.InitContainerSpecs).To(HaveLen(1))
		Expect(result.InitContainerSpecs[0].Name).To(Equal("trpc-init"))
		Expect(result.InitContainerSpecs[0].Image).To(Equal("busybox:1.36"))
		Expect(result.InitContainerSpecs[0].VolumeMounts).To(HaveLen(2))
		Expect(result.InitContainerSpecs[0].Command).To(HaveLen(3))

		// 验证 init container 使用 for 循环脚本而非 per-file 命令
		script := result.InitContainerSpecs[0].Command[2]
		Expect(script).To(ContainSubstring("for f in /trpc-config-template/*"))
		Expect(script).To(ContainSubstring(`cp "$f" "/trpc-config-rendered/$name"`))
		Expect(script).To(ContainSubstring("__#VAR_PLACEHOLDER#__BKMS_POD_IP__"))
		Expect(script).To(ContainSubstring("__#VAR_PLACEHOLDER#__BKMS_POD_NAME__"))
		Expect(script).To(ContainSubstring("__#VAR_PLACEHOLDER#__BKMS_NODE_IP__"))
		Expect(script).To(ContainSubstring("done"))
	})

	It("should build multi-file config rendering resources with same loop script", func() {
		result, err := runtimerender.BuildConfig(runtimerender.ConfigParams{
			WorkloadType:  "plain-cfg",
			ConfigMapName: "my-app-plain-cfg",
			Files: []cfgrender.RenderedMountableFile{
				{FileName: "nginx.conf", FilePath: "/etc/nginx", FileContent: "worker_processes 4;\n"},
				{FileName: "redis.conf", FilePath: "/etc/redis", FileContent: "maxmemory 256mb\n"},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.ConfigMap.Data).To(HaveLen(2))
		Expect(result.ConfigMap.Data).To(HaveKeyWithValue("00-nginx.conf", "worker_processes 4;\n"))
		Expect(result.ConfigMap.Data).To(HaveKeyWithValue("01-redis.conf", "maxmemory 256mb\n"))
		Expect(result.MainContainerMounts).To(HaveLen(2))
		Expect(result.MainContainerMounts[0].MountPath).To(Equal("/etc/nginx/nginx.conf"))
		Expect(result.MainContainerMounts[1].MountPath).To(Equal("/etc/redis/redis.conf"))

		// for 循环脚本不包含文件名——它遍历整个目录
		script := result.InitContainerSpecs[0].Command[2]
		Expect(script).To(ContainSubstring("for f in /plain-cfg-config-template/*"))
		Expect(script).NotTo(ContainSubstring("00-nginx.conf"))
		Expect(script).NotTo(ContainSubstring("01-redis.conf"))
	})

	It("should return empty config when files list is empty", func() {
		result, err := runtimerender.BuildConfig(runtimerender.ConfigParams{
			WorkloadType:  "trpc",
			ConfigMapName: "demo-app",
			Files:         nil,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.MainContainerMounts).To(BeEmpty())
		Expect(result.Volumes).To(BeEmpty())
	})

	It("should return error when duplicate mount paths exist", func() {
		_, err := runtimerender.BuildConfig(runtimerender.ConfigParams{
			WorkloadType:  "trpc",
			ConfigMapName: "demo-app",
			Files: []cfgrender.RenderedMountableFile{
				{FileName: "app.yaml", FilePath: "/etc/app", FileContent: "a"},
				{FileName: "app.yaml", FilePath: "/etc/app", FileContent: "b"},
			},
		})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("duplicate"))
	})
})
