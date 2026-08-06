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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

var _ = Describe("Build", func() {
	Describe("when given valid params", func() {
		It("should generate all K8s injection objects with correct values", func() {
			params := bscpcfg.Params{
				BscpBizID: "100",
				AppNames:  "bkms-order-svc-dev,bkms-user-svc-dev",
				MountPath: "/custom/path", // 主容器 bscp-share 的挂载路径
				FeedAddr:  "bscp-feed.example.com",
				Token:     "test-token-abc123",
			}

			result := bscpcfg.Build(params)

			// InitContainers
			Expect(result.InitContainers).To(HaveLen(1))
			initContainer := result.InitContainers[0]
			Expect(initContainer.Name).To(Equal(bscpcfg.InitContainerName))
			Expect(initContainer.Image).To(Equal(bscpcfg.InitImage))
			Expect(initContainer.Args).To(Equal([]string{"--file-cache-enabled=false"}))
			Expect(initContainer.VolumeMounts).To(HaveLen(1))
			Expect(initContainer.VolumeMounts[0].Name).To(Equal(bscpcfg.VolumeName))
			Expect(initContainer.VolumeMounts[0].MountPath).To(Equal(bscpcfg.BscpDownloadPath))

			// init 容器环境变量值验证
			envMap := make(map[string]string)
			for _, env := range initContainer.Env {
				envMap[env.Name] = env.Value
			}
			Expect(envMap).To(HaveLen(5))
			Expect(envMap["biz"]).To(Equal("100"))
			Expect(envMap["app"]).To(Equal("bkms-order-svc-dev,bkms-user-svc-dev"))
			Expect(envMap["feed_addrs"]).To(Equal("bscp-feed.example.com"))
			Expect(envMap["token"]).To(Equal("test-token-abc123"))
			Expect(envMap["temp_dir"]).To(Equal(bscpcfg.BscpDownloadPath))

			// Containers (sidecar)
			Expect(result.Containers).To(HaveLen(1))
			sidecar := result.Containers[0]
			Expect(sidecar.Name).To(Equal(bscpcfg.SidecarContainerName))
			Expect(sidecar.Image).To(Equal(bscpcfg.SidecarImage))
			Expect(sidecar.Args).To(Equal([]string{"--file-cache-enabled=false"}))
			Expect(sidecar.VolumeMounts).To(HaveLen(2))
			// [0] bscp-temp（与 init 共享的临时目录）
			Expect(sidecar.VolumeMounts[0].Name).To(Equal(bscpcfg.VolumeName))
			Expect(sidecar.VolumeMounts[0].MountPath).To(Equal(bscpcfg.BscpDownloadPath))
			// [1] bscp-share（与主容器共享，固定路径）
			Expect(sidecar.VolumeMounts[1].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(sidecar.VolumeMounts[1].MountPath).To(Equal(bscpcfg.BscpShareBasePath))

			// sidecar 环境变量应与 init 容器一致
			sidecarEnvMap := make(map[string]string)
			for _, env := range sidecar.Env {
				sidecarEnvMap[env.Name] = env.Value
			}
			Expect(sidecarEnvMap).To(Equal(envMap))

			// Volumes：bscp-temp + bscp-share，均为 emptyDir
			Expect(result.Volumes).To(HaveLen(2))
			Expect(result.Volumes[0].Name).To(Equal(bscpcfg.VolumeName))
			Expect(result.Volumes[0].VolumeSource.EmptyDir).NotTo(BeNil())
			Expect(result.Volumes[1].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(result.Volumes[1].VolumeSource.EmptyDir).NotTo(BeNil())

			// MainContainerVolumeMounts：主容器挂载 bscp-share 到用户指定的 MountPath
			Expect(result.MainContainerVolumeMounts).To(HaveLen(1))
			Expect(result.MainContainerVolumeMounts[0].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(result.MainContainerVolumeMounts[0].MountPath).To(Equal("/custom/path"))
		})
	})
})
