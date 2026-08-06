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

package discovery

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	k8sgvr "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
)

var _ = Describe("Discovery", func() {
	var (
		clusterID string
		cfg       *cluster.Config
		err       error
	)

	BeforeEach(func() {
		redis.InitClientForTest()
		clusterID = "test-cluster"
		cfg, err = testutil.TestClusterConfig(clusterID)
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("GetServerVersion", func() {
		Context("when cluster is available", func() {
			It("should return server version", func() {
				versionInfo, err := GetServerVersion(cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(versionInfo).NotTo(BeNil())
				// 目前 k8s 版本都是 1.x，还没到 2.x
				Expect(versionInfo.Major).To(Equal("1"))
			})
		})
	})

	Describe("GetGroupVersionResource", func() {
		Context("with specified group version", func() {
			It("should return resource with group version", func() {
				// 特殊指定 groupVersion 时候会在这
				gvr, err := GetGroupVersionResource(cfg, "Pod", "v1")
				Expect(err).NotTo(HaveOccurred())
				Expect(*gvr).To(Equal(k8sgvr.Po))
			})
		})

		Context("with preferred version", func() {
			It("should return resource with preferred version", func() {
				gvr, err := GetGroupVersionResource(cfg, "Pod", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(*gvr).To(Equal(k8sgvr.Po))
			})
		})

		Context("with other group version", func() {
			It("should return error", func() {
				_, err := GetGroupVersionResource(cfg, "Pod", "apps/v1")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrKindNotFound))
			})
		})

		Context("with invalid group version", func() {
			It("should return error", func() {
				_, err := GetGroupVersionResource(cfg, "Pod", "v201")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrKindNotFound))
			})
		})

		Context("with invalid resource kind", func() {
			It("should return error", func() {
				_, err := GetGroupVersionResource(cfg, "Invalid", "")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrKindNotFound))
			})
		})
	})

	Describe("GetResPreferredVersion", func() {
		Context("with valid resource kind", func() {
			It("should return pod preferred version", func() {
				version, err := GetResPreferredVersion(cfg, "Pod")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("v1"))
			})

			It("should return deployment preferred version", func() {
				version, err := GetResPreferredVersion(cfg, "Deployment")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("apps/v1"))
			})

			It("should return cronjob preferred version", func() {
				version, err := GetResPreferredVersion(cfg, "CronJob")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(BeElementOf("batch/v1beta1", "batch/v1"))
			})

			It("should return crd preferred version", func() {
				version, err := GetResPreferredVersion(cfg, "CustomResourceDefinition")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("apiextensions.k8s.io/v1"))
			})
		})

		Context("with invalid resource kind", func() {
			It("should return error", func() {
				_, err := GetResPreferredVersion(cfg, "Invalid")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrKindNotFound))
			})
		})
	})
})
