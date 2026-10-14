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

package deploydiagnosis

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
)

func requiredTopLevelFields(out *serializer.DeployDiagnosisOutputObj) {
	Expect(out).NotTo(BeNil())
	Expect(out.GeneratedAt).NotTo(BeZero())
	Expect(out.PartialReasons).NotTo(BeNil())
	Expect(out.Groups).NotTo(BeNil())
}

var _ = Describe("diagnose", func() {
	It("returns no_instance when there are no pods and deploy has not failed", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType:   bkmsapp.AppTypeTRPC,
			Deploy:    serializer.DeployOutputObj{Status: "deployed"},
			Instances: []*rules.Instance{},
		})

		requiredTopLevelFields(out)
		Expect(out.EmptyReason).To(Equal(serializer.EmptyReasonNoInstance))
		Expect(out.Groups).To(BeEmpty())
		Expect(out.Summary.TotalCount).To(Equal(0))
		Expect(out.Summary.AbnormalCount).To(Equal(0))
	})

	It("returns unknown when there are no pods and deploy has failed", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType:   bkmsapp.AppTypeTRPC,
			Deploy:    serializer.DeployOutputObj{Status: "failed", Message: "workload not ready"},
			Instances: []*rules.Instance{},
		})

		requiredTopLevelFields(out)
		Expect(out.EmptyReason).To(BeEmpty())
		Expect(out.Groups).To(HaveLen(1))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDUnknown))
		Expect(out.Groups[0].Instances).To(BeEmpty())
		Expect(out.Groups[0].Cause).NotTo(BeEmpty())
	})

	It("returns unknown when every instance is Ready but deploy has failed", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType: bkmsapp.AppTypeTRPC,
			Deploy:  serializer.DeployOutputObj{Status: "polling-timeout"},
			Instances: []*rules.Instance{
				{ID: "pod-a", Status: "Running", IsHealthy: true},
			},
		})

		requiredTopLevelFields(out)
		Expect(out.EmptyReason).To(BeEmpty())
		Expect(out.Summary.TotalCount).To(Equal(1))
		Expect(out.Summary.AbnormalCount).To(Equal(0))
		Expect(out.Groups).To(HaveLen(1))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDUnknown))
		Expect(out.Groups[0].Instances).To(BeEmpty())
	})

	It("returns all_healthy when every instance is Ready and deploy has not failed", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType: bkmsapp.AppTypeTRPC,
			Deploy:  serializer.DeployOutputObj{Status: "deployed"},
			Instances: []*rules.Instance{
				{ID: "pod-a", Status: "Running", IsHealthy: true},
				{ID: "pod-b", Status: "Running", IsHealthy: true},
			},
		})

		requiredTopLevelFields(out)
		Expect(out.EmptyReason).To(Equal(serializer.EmptyReasonAllHealthy))
		Expect(out.Groups).To(BeEmpty())
		Expect(out.Summary.TotalCount).To(Equal(2))
		Expect(out.Summary.AbnormalCount).To(Equal(0))
	})

	It("matches config-file-not-found and marks a mismatched log path as danger", func() {
		logLine := "open /data/trpc_go.yaml: no such file or directory"
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType:   bkmsapp.AppTypeTRPC,
			Deploy:    serializer.DeployOutputObj{DeployID: "d-1", Status: "deployed"},
			Framework: rules.Framework{FileName: "trpc_go.yaml", FilePath: "/app/conf", Command: "/app/server"},
			Instances: []*rules.Instance{
				{
					ID:           "pod-a",
					Status:       "CrashLoopBackOff",
					RestartCount: 5,
					IsHealthy:    false,
					LogSource:    serializer.LogSourcePrevious,
					LogHint:      previousLogHint,
					LogLines:     []string{"[INFO] starting trpc server", logLine},
				},
			},
		})

		requiredTopLevelFields(out)
		Expect(out.EmptyReason).To(BeEmpty())
		Expect(out.Groups).To(HaveLen(1))
		group := out.Groups[0]
		Expect(group.RuleID).To(Equal(rules.IDConfigFileNotFound))
		Expect(group.Instances).To(HaveLen(1))
		Expect(group.Instances[0].Evidence).To(ContainElement(logLine))
		Expect(group.Actions).To(Equal([]*serializer.DiagnosisActionObj{
			{ID: serializer.ActionFrameworkConfig, Label: "前往框架配置文件"},
			{ID: serializer.ActionRedeploy, Label: "重新部署"},
		}))

		var logPathRow *serializer.RelatedConfigRowObj
		for _, row := range group.Instances[0].RelatedConfigs {
			if row.Label == "日志中的读取路径" {
				logPathRow = row
			}
		}
		Expect(logPathRow).NotTo(BeNil())
		Expect(logPathRow.Value).To(Equal("/data/trpc_go.yaml"))
		Expect(logPathRow.Danger).To(BeTrue())
		Expect(logPathRow.Hint).To(Equal(rules.PathMismatchHint))
	})

	DescribeTable("does not treat a non-config missing file as config-file-not-found",
		func(logLine string) {
			out := new(Diagnoser).diagnose(&rules.Snapshot{
				AppType: bkmsapp.AppTypeTRPC,
				Framework: rules.Framework{
					FileName: "trpc_go.yaml",
					FilePath: "/app/conf",
				},
				Instances: []*rules.Instance{
					{
						ID:        "pod-c",
						Status:    "CrashLoopBackOff",
						IsHealthy: false,
						LogLines:  []string{logLine},
					},
				},
			})

			Expect(out.Groups).To(HaveLen(1))
			Expect(out.Groups[0].RuleID).To(Equal(rules.IDUnknown))
		},
		Entry("missing binary", "open /usr/bin/server: no such file or directory"),
		Entry("missing socket", "dial unix /var/run/app.sock: connect: no such file or directory"),
		Entry("permission denied", "failed to load config: permission denied"),
		Entry("yaml syntax", "yaml: field server not found"),
		Entry("other config file", "open /data/settings.yaml: no such file or directory"),
	)

	It("puts the more specific rule group before unknown", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType: bkmsapp.AppTypeTRPC,
			Framework: rules.Framework{
				FileName: "trpc_go.yaml",
				FilePath: "/app/conf",
			},
			Instances: []*rules.Instance{
				{
					ID:        "pod-unknown",
					IsHealthy: false,
					LogLines:  []string{"panic: unexpected nil pointer"},
				},
				{
					ID:        "pod-config",
					IsHealthy: false,
					LogLines:  []string{"open /data/trpc_go.yaml: no such file or directory"},
				},
			},
		})

		Expect(out.Groups).To(HaveLen(2))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDConfigFileNotFound))
		Expect(out.Groups[1].RuleID).To(Equal(rules.IDUnknown))
	})

	It("marks a relative config path as danger", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType: bkmsapp.AppTypeTRPC,
			Framework: rules.Framework{
				FileName: "trpc_go.yaml",
				FilePath: "/app/conf",
			},
			Instances: []*rules.Instance{
				{
					ID:        "pod-rel",
					IsHealthy: false,
					LogLines:  []string{"open trpc_go.yaml: no such file or directory"},
				},
			},
		})

		Expect(out.Groups[0].RuleID).To(Equal(rules.IDConfigFileNotFound))
		Expect(out.Groups[0].Instances[0].RelatedConfigs[2].Danger).To(BeTrue())
	})

	It("falls back to unknown when an abnormal instance does not match config-file-not-found", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType: bkmsapp.AppTypeTRPC,
			Instances: []*rules.Instance{
				{
					ID:        "pod-b",
					Status:    "CrashLoopBackOff",
					IsHealthy: false,
					LogLines:  []string{"panic: unexpected nil pointer"},
				},
			},
		})

		requiredTopLevelFields(out)
		Expect(out.Groups).To(HaveLen(1))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDUnknown))
		Expect(out.Groups[0].Actions).To(Equal([]*serializer.DiagnosisActionObj{
			{ID: serializer.ActionDeployConfig, Label: "前往部署配置"},
			{ID: serializer.ActionRedeploy, Label: "重新部署"},
		}))
	})

	It("does not emit config-file-not-found for Helm apps", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			AppType: bkmsapp.AppTypeHelm,
			Instances: []*rules.Instance{
				{
					ID:        "pod-h",
					Status:    "CrashLoopBackOff",
					IsHealthy: false,
					LogLines:  []string{"open /data/trpc_go.yaml: no such file or directory"},
				},
			},
		})

		Expect(out.Groups).To(HaveLen(1))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDUnknown))
	})
})
