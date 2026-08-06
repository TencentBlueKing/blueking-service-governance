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

package lifecycle

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
		Entry("converts postStart with Exec action",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					Lifecycle: &appmodel.Lifecycle{
						PostStart: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"sh", "-c", "echo hello"},
							},
						},
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.PostStart).NotTo(BeNil())
				Expect(spec.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
				Expect(spec.PostStart.Command).To(Equal([]string{"sh", "-c", "echo hello"}))
				Expect(spec.PreStop).To(BeNil())
			},
		),
		Entry("converts preStop with Http action",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					Lifecycle: &appmodel.Lifecycle{
						PreStop: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeHTTP},
							HTTPGetAction: &appmodel.HTTPGetAction{
								URL: "http://localhost:8080/shutdown",
								Headers: map[string]string{
									"X-Custom": "value",
								},
							},
						},
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.PreStop).NotTo(BeNil())
				Expect(spec.PreStop.Type).To(Equal(appmodel.LifecycleTypeHTTP))
				Expect(spec.PreStop.URL).To(Equal("http://localhost:8080/shutdown"))
				Expect(spec.PreStop.Headers).To(Equal(map[string]string{"X-Custom": "value"}))
				Expect(spec.PostStart).To(BeNil())
			},
		),
		Entry("converts Exec action with sleepSeconds parameter",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					Lifecycle: &appmodel.Lifecycle{
						PostStart: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command:      []string{"touch", "/tmp/ready"},
								SleepSeconds: lo.ToPtr(int64(5)),
							},
						},
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.PostStart).NotTo(BeNil())
				Expect(spec.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
				Expect(spec.PostStart.Command).To(Equal([]string{"touch", "/tmp/ready"}))
				Expect(spec.PostStart.SleepSeconds).To(Equal(lo.ToPtr(int64(5))))
			},
		),
		Entry("converts Exec action with shell command",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					Lifecycle: &appmodel.Lifecycle{
						PostStart: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
							ExecAction: &appmodel.ExecAction{
								ShCommand: "test -f /tmp/ready",
							},
						},
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.PostStart).NotTo(BeNil())
				Expect(spec.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
				Expect(spec.PostStart.ShCommand).To(Equal("test -f /tmp/ready"))
				Expect(spec.PostStart.Command).To(BeEmpty())
			},
		),
		Entry("converts terminationGracePeriodSeconds only",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					TerminationGracePeriodSeconds: lo.ToPtr(int64(30)),
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.TerminationGracePeriodSeconds).To(Equal(lo.ToPtr(int64(30))))
				Expect(spec.PostStart).To(BeNil())
				Expect(spec.PreStop).To(BeNil())
			},
		),
		Entry("converts terminationGracePeriodSeconds with lifecycle hooks",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					TerminationGracePeriodSeconds: lo.ToPtr(int64(60)),
					Lifecycle: &appmodel.Lifecycle{
						PreStop: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"sh", "-c", "sleep 10"},
							},
						},
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.TerminationGracePeriodSeconds).To(Equal(lo.ToPtr(int64(60))))
				Expect(spec.PreStop).NotTo(BeNil())
				Expect(spec.PreStop.Type).To(Equal(appmodel.LifecycleTypeExec))
			},
		),
		Entry("converts both postStart and preStop",
			&appmodel.AppModel{
				Workload: appmodel.Workload{
					Lifecycle: &appmodel.Lifecycle{
						PostStart: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"sh", "-c", "echo started"},
							},
						},
						PreStop: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeHTTP},
							HTTPGetAction: &appmodel.HTTPGetAction{
								URL: "http://localhost:8080/health",
							},
						},
					},
				},
			},
			func(spec *Spec) {
				Expect(spec).NotTo(BeNil())
				Expect(spec.PostStart).NotTo(BeNil())
				Expect(spec.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
				Expect(spec.PreStop).NotTo(BeNil())
				Expect(spec.PreStop.Type).To(Equal(appmodel.LifecycleTypeHTTP))
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
		Entry("applies postStart Exec action",
			&Spec{
				PostStart: &Handler{
					Type:    appmodel.LifecycleTypeExec,
					Command: []string{"sh", "-c", "setup"},
				},
			},
			&appmodel.AppModel{
				AppID:    "app-1",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.Lifecycle).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
				Expect(
					appModel.Workload.Lifecycle.PostStart.ExecAction.Command,
				).To(Equal([]string{"sh", "-c", "setup"}))
			},
		),
		Entry("applies preStop Http action",
			&Spec{
				PreStop: &Handler{
					Type: appmodel.LifecycleTypeHTTP,
					URL:  "http://localhost:8080/shutdown",
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
			&appmodel.AppModel{
				AppID:    "app-2",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.Lifecycle).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PreStop).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PreStop.Type).To(Equal(appmodel.LifecycleTypeHTTP))
				Expect(
					appModel.Workload.Lifecycle.PreStop.HTTPGetAction.URL,
				).To(Equal("http://localhost:8080/shutdown"))
			},
		),
		Entry("applies Exec action with sleepSeconds parameter",
			&Spec{
				PostStart: &Handler{
					Type:         appmodel.LifecycleTypeExec,
					SleepSeconds: lo.ToPtr(int64(10)),
				},
			},
			&appmodel.AppModel{
				AppID:    "app-3",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.Lifecycle).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart.ExecAction).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart.ExecAction.SleepSeconds).To(Equal(lo.ToPtr(int64(10))))
			},
		),
		Entry("applies Exec action with shell command",
			&Spec{
				PostStart: &Handler{
					Type:      appmodel.LifecycleTypeExec,
					ShCommand: "curl -sf localhost/ready",
				},
			},
			&appmodel.AppModel{
				AppID:    "app-shell",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.Lifecycle).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
				Expect(
					appModel.Workload.Lifecycle.PostStart.ExecAction.ShCommand,
				).To(Equal("curl -sf localhost/ready"))
				Expect(appModel.Workload.Lifecycle.PostStart.ExecAction.Command).To(BeEmpty())
			},
		),
		Entry("overrides existing lifecycle when spec is not nil",
			&Spec{
				PostStart: &Handler{
					Type:    appmodel.LifecycleTypeExec,
					Command: []string{"new-cmd"},
				},
			},
			&appmodel.AppModel{
				AppID: "app-4",
				Workload: appmodel.Workload{
					Lifecycle: &appmodel.Lifecycle{
						PostStart: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeHTTP},
							HTTPGetAction: &appmodel.HTTPGetAction{
								URL: "http://old-url",
							},
						},
					},
				},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.Lifecycle).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart.Type).To(Equal(appmodel.LifecycleTypeExec))
				Expect(appModel.Workload.Lifecycle.PostStart.ExecAction.Command).To(Equal([]string{"new-cmd"}))
			},
		),
		Entry("clears lifecycle when spec is nil",
			nil,
			&appmodel.AppModel{
				AppID: "app-5",
				Workload: appmodel.Workload{
					TerminationGracePeriodSeconds: lo.ToPtr(int64(30)),
					Lifecycle: &appmodel.Lifecycle{
						PostStart: &appmodel.LifecycleHandler{
							TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
							ExecAction: &appmodel.ExecAction{
								Command: []string{"cmd"},
							},
						},
					},
				},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.Lifecycle).To(BeNil())
				Expect(appModel.Workload.TerminationGracePeriodSeconds).To(BeNil())
			},
		),
		Entry("applies both postStart and preStop",
			&Spec{
				PostStart: &Handler{
					Type:    appmodel.LifecycleTypeExec,
					Command: []string{"start"},
				},
				PreStop: &Handler{
					Type: appmodel.LifecycleTypeHTTP,
					URL:  "http://localhost:8080/stop",
				},
				TerminationGracePeriodSeconds: lo.ToPtr(int64(30)),
			},
			&appmodel.AppModel{
				AppID:    "app-6",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.Lifecycle).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PostStart).NotTo(BeNil())
				Expect(appModel.Workload.Lifecycle.PreStop).NotTo(BeNil())
				Expect(appModel.Workload.TerminationGracePeriodSeconds).To(Equal(lo.ToPtr(int64(30))))
			},
		),
		Entry("applies terminationGracePeriodSeconds",
			&Spec{
				TerminationGracePeriodSeconds: lo.ToPtr(int64(45)),
			},
			&appmodel.AppModel{
				AppID:    "app-7",
				Workload: appmodel.Workload{},
			},
			func(appModel *appmodel.AppModel) {
				Expect(appModel.Workload.Lifecycle).To(BeNil())
				Expect(appModel.Workload.TerminationGracePeriodSeconds).To(Equal(lo.ToPtr(int64(45))))
			},
		),
	)
})
