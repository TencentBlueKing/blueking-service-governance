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

package workload

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("buildProbe", func() {
	It("should return nil when probe is nil", func() {
		p, err := buildProbe(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(p).To(BeNil())
	})

	It("should build exec probe", func() {
		probe := &appmodel.Probe{
			ProbeHandler: &appmodel.ProbeHandler{
				TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
				ExecAction:  &appmodel.ExecAction{Command: []string{"echo", "ok"}},
			},
			InitialDelaySeconds: 5,
		}
		p, err := buildProbe(probe)
		Expect(err).NotTo(HaveOccurred())
		Expect(p.Exec.Command).To(Equal([]string{"echo", "ok"}))
		Expect(p.InitialDelaySeconds).To(Equal(int32(5)))
	})

	It("should build httpGet probe", func() {
		probe := &appmodel.Probe{
			ProbeHandler: &appmodel.ProbeHandler{
				TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeHTTP},
				HTTPGetAction: &appmodel.HTTPGetAction{
					URL:  "http://example.com/healthz",
					Port: 8080,
					Headers: map[string]string{
						"X-Test": "true",
					},
				},
			},
		}
		p, err := buildProbe(probe)
		Expect(err).NotTo(HaveOccurred())
		Expect(p.HTTPGet.Host).To(Equal("example.com"))
		Expect(p.HTTPGet.Path).To(Equal("/healthz"))
		Expect(p.HTTPGet.Port.IntValue()).To(Equal(8080))
		Expect(p.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTP))
		Expect(p.HTTPGet.HTTPHeaders).To(ContainElement(corev1.HTTPHeader{
			Name:  "X-Test",
			Value: "true",
		}))
	})

	It("should build tcpSocket probe", func() {
		probe := &appmodel.Probe{
			ProbeHandler: &appmodel.ProbeHandler{
				TypeWrapper:     appmodel.TypeWrapper{Type: appmodel.ProbeTypeTCP},
				TCPSocketAction: &appmodel.TCPSocketAction{Port: 9090},
			},
		}
		p, err := buildProbe(probe)
		Expect(err).NotTo(HaveOccurred())
		Expect(p.TCPSocket.Port.IntValue()).To(Equal(9090))
	})

	It("should build exec shell probe as exec with sh -c", func() {
		probe := &appmodel.Probe{
			ProbeHandler: &appmodel.ProbeHandler{
				TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
				ExecAction:  &appmodel.ExecAction{ShCommand: "echo ok && exit 0"},
			},
		}
		p, err := buildProbe(probe)
		Expect(err).NotTo(HaveOccurred())
		Expect(p.Exec).NotTo(BeNil())
		Expect(p.Exec.Command).To(Equal([]string{"/bin/sh", "-c", "echo ok && exit 0"}))
	})

	It("should reject invalid handler type", func() {
		probe := &appmodel.Probe{
			ProbeHandler: &appmodel.ProbeHandler{
				TypeWrapper: appmodel.TypeWrapper{Type: "Unknown"},
			},
		}
		_, err := buildProbe(probe)
		Expect(err).To(HaveOccurred())
	})

	It("should reject invalid httpGet url", func() {
		probe := &appmodel.Probe{
			ProbeHandler: &appmodel.ProbeHandler{
				TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeHTTP},
				HTTPGetAction: &appmodel.HTTPGetAction{
					URL:  "xxx://not-a-valid-url:aaa:aaa/",
					Port: 8080,
				},
			},
		}
		_, err := buildProbe(probe)
		Expect(err).To(HaveOccurred())
	})
})
