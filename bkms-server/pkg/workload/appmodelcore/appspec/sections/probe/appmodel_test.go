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

package probe

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("app model conversion", func() {
	DescribeTable("FromAppModel",
		func(appModel *appmodel.AppModel, checkFunc func(*Spec)) {
			result := FromAppModel(appModel)
			checkFunc(result)
		},
		Entry("returns nil for an empty app model",
			&appmodel.AppModel{
				Workload: appmodel.Workload{},
			},
			func(spec *Spec) {
				Expect(spec).To(BeNil())
			},
		),
		Entry("converts liveness probe with Exec handler",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					LivenessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"sh", "-c", "test -f /tmp/ready"},
							},
						},
						InitialDelaySeconds: 10,
						TimeoutSeconds:      5,
						PeriodSeconds:       10,
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.Liveness).NotTo(BeNil())
				Expect(spec.Liveness.Handler).NotTo(BeNil())
				Expect(spec.Liveness.Handler.Type).To(Equal(appmodel.ProbeTypeExec))
				Expect(spec.Liveness.Handler.Command).To(Equal([]string{"sh", "-c", "test -f /tmp/ready"}))
				Expect(spec.Liveness.InitialDelaySeconds).To(Equal(lo.ToPtr(int32(10))))
				Expect(spec.Liveness.TimeoutSeconds).To(Equal(lo.ToPtr(int32(5))))
				Expect(spec.Liveness.PeriodSeconds).To(Equal(lo.ToPtr(int32(10))))
			},
		),
		Entry("converts readiness probe with Http handler",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					ReadinessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeHTTP},
							HTTPGetAction: &appmodel.HTTPGetAction{
								URL:  "http://localhost/health",
								Port: 8080,
								Headers: map[string]string{
									"X-Check": "ready",
								},
							},
						},
						InitialDelaySeconds: 5,
						TimeoutSeconds:      2,
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.Readiness).NotTo(BeNil())
				Expect(spec.Readiness.Handler).NotTo(BeNil())
				Expect(spec.Readiness.Handler.Type).To(Equal(appmodel.ProbeTypeHTTP))
				Expect(spec.Readiness.Handler.URL).To(Equal("http://localhost/health"))
				Expect(spec.Readiness.Handler.Port).To(Equal(int32(8080)))
				Expect(spec.Readiness.InitialDelaySeconds).To(Equal(lo.ToPtr(int32(5))))
			},
		),
		Entry("converts startup probe with Tcp handler",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					StartupProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper:     appmodel.TypeWrapper{Type: appmodel.ProbeTypeTCP},
							TCPSocketAction: &appmodel.TCPSocketAction{Port: 3306},
						},
						FailureThreshold: 30,
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.Startup).NotTo(BeNil())
				Expect(spec.Startup.Handler).NotTo(BeNil())
				Expect(spec.Startup.Handler.Type).To(Equal(appmodel.ProbeTypeTCP))
				Expect(spec.Startup.Handler.Port).To(Equal(int32(3306)))
				Expect(spec.Startup.FailureThreshold).To(Equal(lo.ToPtr(int32(30))))
			},
		),
		Entry("converts liveness probe with exec shell handler",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					LivenessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
							ExecAction:  &appmodel.ExecAction{ShCommand: "test -f /tmp/ready"},
						},
						InitialDelaySeconds: 1,
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.Liveness).NotTo(BeNil())
				Expect(spec.Liveness.Handler).NotTo(BeNil())
				Expect(spec.Liveness.Handler.Type).To(Equal(appmodel.ProbeTypeExec))
				Expect(spec.Liveness.Handler.ShCommand).To(Equal("test -f /tmp/ready"))
				Expect(spec.Liveness.InitialDelaySeconds).To(Equal(lo.ToPtr(int32(1))))
			},
		),
		Entry("converts probe with zero timing fields (omitted)",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					LivenessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"test"},
							},
						},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.Liveness).NotTo(BeNil())
				Expect(spec.Liveness.InitialDelaySeconds).To(BeNil())
				Expect(spec.Liveness.TimeoutSeconds).To(BeNil())
			},
		),
		Entry("converts all three probes simultaneously",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					LivenessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"alive"},
							},
						},
						PeriodSeconds: 20,
					},
					ReadinessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeHTTP},
							HTTPGetAction: &appmodel.HTTPGetAction{
								URL:  "http://localhost/ready",
								Port: 8080,
							},
						},
						InitialDelaySeconds: 5,
					},
					StartupProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeTCP},
							TCPSocketAction: &appmodel.TCPSocketAction{
								Port: 3306,
							},
						},
						FailureThreshold: 30,
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.Liveness).NotTo(BeNil())
				Expect(spec.Readiness).NotTo(BeNil())
				Expect(spec.Startup).NotTo(BeNil())
			},
		),
	)

	DescribeTable("ApplyToAppModel",
		func(
			spec *Spec,
			appModel *appmodel.AppModel,
			checkFunc func(*appmodel.AppModel),
		) {
			result := ApplyToAppModel(spec, appModel)
			checkFunc(result)
		},
		Entry("applies liveness probe with Exec handler",
			&Spec{
				Liveness: &Probe{
					Handler: &Handler{
						Type:    appmodel.ProbeTypeExec,
						Command: []string{"sh", "-c", "ready"},
					},
					InitialDelaySeconds: lo.ToPtr(int32(15)),
					TimeoutSeconds:      lo.ToPtr(int32(3)),
				},
			},
			&appmodel.AppModel{
				AppID:    "app-1",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.LivenessProbe).NotTo(BeNil())
				Expect(appModel.Workload.LivenessProbe.ProbeHandler).NotTo(BeNil())
				Expect(appModel.Workload.LivenessProbe.ProbeHandler.Command).To(Equal([]string{"sh", "-c", "ready"}))
				Expect(appModel.Workload.LivenessProbe.InitialDelaySeconds).To(Equal(int32(15)))
				Expect(appModel.Workload.LivenessProbe.TimeoutSeconds).To(Equal(int32(3)))
			},
		),
		Entry("applies liveness probe with exec shell handler",
			&Spec{
				Liveness: &Probe{
					Handler: &Handler{
						Type:      appmodel.ProbeTypeExec,
						ShCommand: "curl -sf localhost",
					},
					InitialDelaySeconds: lo.ToPtr(int32(5)),
					TimeoutSeconds:      lo.ToPtr(int32(3)),
				},
			},
			&appmodel.AppModel{
				AppID:    "app-shell",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.LivenessProbe).NotTo(BeNil())
				Expect(appModel.Workload.LivenessProbe.ProbeHandler).NotTo(BeNil())
				Expect(appModel.Workload.LivenessProbe.ProbeHandler.Type).To(Equal(appmodel.ProbeTypeExec))
				Expect(appModel.Workload.LivenessProbe.ProbeHandler.ShCommand).To(Equal("curl -sf localhost"))
				Expect(appModel.Workload.LivenessProbe.InitialDelaySeconds).To(Equal(int32(5)))
				Expect(appModel.Workload.LivenessProbe.TimeoutSeconds).To(Equal(int32(3)))
			},
		),
		Entry("applies readiness probe with Http handler",
			&Spec{
				Readiness: &Probe{
					Handler: &Handler{
						Type: appmodel.ProbeTypeHTTP,
						URL:  "http://localhost/status",
						Port: 8080,
						Headers: map[string]string{
							"Authorization": "Bearer token",
						},
					},
					InitialDelaySeconds: lo.ToPtr(int32(10)),
					PeriodSeconds:       lo.ToPtr(int32(5)),
				},
			},
			&appmodel.AppModel{
				AppID:    "app-2",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.ReadinessProbe).NotTo(BeNil())
				Expect(appModel.Workload.ReadinessProbe.ProbeHandler).NotTo(BeNil())
				Expect(appModel.Workload.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(10)))
				Expect(appModel.Workload.ReadinessProbe.PeriodSeconds).To(Equal(int32(5)))
			},
		),
		Entry("applies startup probe with Tcp handler",
			&Spec{
				Startup: &Probe{
					Handler: &Handler{
						Type: appmodel.ProbeTypeTCP,
						Port: 5432,
					},
					FailureThreshold: lo.ToPtr(int32(20)),
				},
			},
			&appmodel.AppModel{
				AppID:    "app-3",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.StartupProbe).NotTo(BeNil())
				Expect(appModel.Workload.StartupProbe.ProbeHandler).NotTo(BeNil())
				Expect(appModel.Workload.StartupProbe.FailureThreshold).To(Equal(int32(20)))
			},
		),
		Entry("overrides existing probe when spec is not nil",
			&Spec{
				Liveness: &Probe{
					Handler: &Handler{
						Type: appmodel.ProbeTypeHTTP,
						URL:  "http://localhost/health",
						Port: 8080,
					},
				},
			},
			&appmodel.AppModel{
				AppID: "app-4",
				Workload: appmodel.Workload{
					LivenessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"old-check"},
							},
						},
					},
				},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.LivenessProbe).NotTo(BeNil())
				Expect(appModel.Workload.LivenessProbe.ProbeHandler.Type).To(Equal(appmodel.ProbeTypeHTTP))
			},
		),
		Entry("clears all probes when spec is nil",
			nil,
			&appmodel.AppModel{
				AppID: "app-5",
				Workload: appmodel.Workload{
					LivenessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"check"},
							},
						},
					},
					ReadinessProbe: &appmodel.Probe{
						ProbeHandler: &appmodel.ProbeHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeHTTP},
							HTTPGetAction: &appmodel.HTTPGetAction{
								URL: "http://localhost/ready",
							},
						},
					},
				},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.LivenessProbe).To(BeNil())
				Expect(appModel.Workload.ReadinessProbe).To(BeNil())
				Expect(appModel.Workload.StartupProbe).To(BeNil())
			},
		),
		Entry("applies all three probes simultaneously",
			&Spec{
				Liveness: &Probe{
					Handler: &Handler{
						Type:    appmodel.ProbeTypeExec,
						Command: []string{"alive"},
					},
					PeriodSeconds: lo.ToPtr(int32(15)),
				},
				Readiness: &Probe{
					Handler: &Handler{
						Type: appmodel.ProbeTypeHTTP,
						URL:  "http://localhost/ready",
						Port: 8080,
					},
					InitialDelaySeconds: lo.ToPtr(int32(5)),
				},
				Startup: &Probe{
					Handler: &Handler{
						Type: appmodel.ProbeTypeTCP,
						Port: 3306,
					},
					FailureThreshold: lo.ToPtr(int32(30)),
				},
			},
			&appmodel.AppModel{
				AppID:    "app-6",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.LivenessProbe).NotTo(BeNil())
				Expect(appModel.Workload.ReadinessProbe).NotTo(BeNil())
				Expect(appModel.Workload.StartupProbe).NotTo(BeNil())
			},
		),
	)
})
