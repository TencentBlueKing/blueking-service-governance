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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
)

func assertTopLevelFields(out *Diagnosis) {
	Expect(out).NotTo(BeNil())
	Expect(out.GeneratedAt).NotTo(BeZero())
	Expect(out.PartialReasons).NotTo(BeNil())
	Expect(out.Groups).NotTo(BeNil())
}

func makeDeploy(phase deploystatus.DeployPhase) *rules.Deploy {
	return &rules.Deploy{DeployID: "d-1", Phase: phase}
}

func makeCrashingInstance(id string, logLines ...string) *rules.Instance {
	return &rules.Instance{ID: id, Status: "CrashLoopBackOff", IsHealthy: false, LogLines: logLines}
}

var trpcFrameworkConfig = rules.FrameworkConfig{
	Supported: true,
	FileName:  "trpc_go.yaml",
	FilePath:  "/app/conf",
	Command:   "/app/server",
}

func findConfigRow(rows []rules.ConfigRow, label string) *rules.ConfigRow {
	for i := range rows {
		if rows[i].Label == label {
			return &rows[i]
		}
	}
	return nil
}

var _ = Describe("diagnose", func() {
	It("returns no_deploy when the env has never been deployed", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{})

		assertTopLevelFields(out)
		Expect(out.EmptyReason).To(Equal(EmptyReasonNoDeploy))
		Expect(out.Deploy).To(BeNil())
	})

	It("returns deploying without running rules while a deploy is in progress", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			Deploy:          makeDeploy(deploystatus.DeployPhaseInProgress),
			InstancesListed: true,
			Instances:       []*rules.Instance{{ID: "pod-a", Status: "Pending", IsHealthy: false}},
		})

		assertTopLevelFields(out)
		Expect(out.EmptyReason).To(Equal(EmptyReasonDeploying))
		Expect(out.Groups).To(BeEmpty())
	})

	DescribeTable("resolves the empty reason when no rule matches",
		func(snap *rules.Snapshot, want string) {
			out := new(Diagnoser).diagnose(snap)
			assertTopLevelFields(out)
			Expect(out.Groups).To(BeEmpty())
			Expect(out.EmptyReason).To(Equal(want))
		},
		Entry("pods listed but none remain",
			&rules.Snapshot{Deploy: makeDeploy(deploystatus.DeployPhaseSucceeded), InstancesListed: true},
			EmptyReasonNoInstance),
		Entry("deploy exists but pods cannot be listed",
			&rules.Snapshot{Deploy: makeDeploy(deploystatus.DeployPhaseSucceeded)},
			EmptyReasonInstancesUnavailable),
		Entry("every instance is ready",
			&rules.Snapshot{
				Deploy:          makeDeploy(deploystatus.DeployPhaseSucceeded),
				InstancesListed: true,
				Instances:       []*rules.Instance{{ID: "pod-a", IsHealthy: true}, {ID: "pod-b", IsHealthy: true}},
			},
			EmptyReasonAllHealthy),
		Entry("build was canceled before anything was deployed",
			&rules.Snapshot{Deploy: &rules.Deploy{Stage: autodeploy.StageBuild, Phase: deploystatus.DeployPhaseOther}},
			EmptyReasonNoDeploy),
	)

	DescribeTable("emits a stage group when there is no abnormal instance",
		func(snap *rules.Snapshot, wantRuleID, wantDetail string) {
			out := new(Diagnoser).diagnose(snap)

			assertTopLevelFields(out)
			Expect(out.EmptyReason).To(BeEmpty())
			Expect(out.Groups).To(HaveLen(1))
			Expect(out.Groups[0].RuleID).To(Equal(wantRuleID))
			Expect(out.Groups[0].Instances).To(BeEmpty())
			Expect(out.Groups[0].CauseDetail).To(ContainSubstring(wantDetail))
		},
		Entry("deploy failed and no instance",
			&rules.Snapshot{Deploy: makeDeploy(deploystatus.DeployPhaseFailed), InstancesListed: true},
			rules.IDDeployFailed, "当前环境没有实例"),
		Entry("deploy failed while every instance is ready",
			&rules.Snapshot{
				Deploy:          makeDeploy(deploystatus.DeployPhaseFailed),
				InstancesListed: true,
				Instances:       []*rules.Instance{{ID: "pod-a", IsHealthy: true}},
			},
			rules.IDDeployFailed, "实例目前都已就绪"),
		Entry("auto deploy failed at the deploy stage",
			&rules.Snapshot{Deploy: &rules.Deploy{
				DeployID: "d-1", Stage: autodeploy.StageDeploy, Phase: deploystatus.DeployPhaseFailed,
			}},
			rules.IDDeployFailed, "最近一次部署也没有成功"),
		Entry("build failed while a previous version is still running",
			&rules.Snapshot{Deploy: &rules.Deploy{
				DeployID: "d-old", BuildID: "b-1", Stage: autodeploy.StageBuild, Phase: deploystatus.DeployPhaseFailed,
			}},
			rules.IDBuildFailed, "运行的仍是上一次部署的版本"),
		Entry("build failed and the env was never deployed",
			&rules.Snapshot{Deploy: &rules.Deploy{
				BuildID: "b-1", Stage: autodeploy.StageBuild, Phase: deploystatus.DeployPhaseFailed,
			}},
			rules.IDBuildFailed, "尚未部署过任何版本"),
	)

	It("routes build failures to the build history action", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{Deploy: &rules.Deploy{
			BuildID: "b-1", Stage: autodeploy.StageBuild, Phase: deploystatus.DeployPhaseFailed,
		}})

		Expect(out.Groups[0].CategoryID).To(Equal(rules.CategoryBuildFailure))
		Expect(out.Groups[0].Category).To(Equal("构建失败"))
		Expect(out.Groups[0].Actions).To(Equal([]rules.Action{{ID: rules.ActionBuildHistory, Label: "查看构建记录"}}))
	})

	It("matches framework-config-not-found and marks a mismatched log path as danger", func() {
		logLine := "open /data/trpc_go.yaml: no such file or directory"
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			Deploy:          makeDeploy(deploystatus.DeployPhaseSucceeded),
			FrameworkConfig: trpcFrameworkConfig,
			Instances: []*rules.Instance{{
				ID:           "pod-a",
				Status:       "CrashLoopBackOff",
				RestartCount: 5,
				LogSource:    rules.LogSourcePrevious,
				LogHint:      rules.LogHintPrevious,
				LogLines:     []string{"[INFO] starting trpc server", logLine},
			}},
		})

		assertTopLevelFields(out)
		Expect(out.Groups).To(HaveLen(1))
		group := out.Groups[0]
		Expect(group.RuleID).To(Equal(rules.IDFrameworkConfigNotFound))
		Expect(group.CategoryID).To(Equal(rules.CategoryConfigDelivery))
		Expect(group.Category).To(Equal("配置下发"))
		Expect(group.Instances[0].Evidence).To(Equal([]string{logLine}))
		Expect(group.Actions).To(Equal([]rules.Action{
			{ID: rules.ActionFrameworkConfig, Label: "前往框架配置文件"},
			{ID: rules.ActionRedeploy, Label: "重新部署"},
		}))

		row := findConfigRow(group.Instances[0].RelatedConfigs, "日志中的读取路径")
		Expect(row).NotTo(BeNil())
		Expect(row.Value).To(Equal("/data/trpc_go.yaml"))
		Expect(row.Danger).To(BeTrue())
		Expect(row.Hint).NotTo(BeEmpty())
	})

	DescribeTable("picks framework config solutions by the shape of the log path",
		func(logLine string, wantDanger bool, wantFirstSolution string) {
			out := new(Diagnoser).diagnose(&rules.Snapshot{
				Deploy:          makeDeploy(deploystatus.DeployPhaseSucceeded),
				FrameworkConfig: trpcFrameworkConfig,
				Instances:       []*rules.Instance{makeCrashingInstance("pod-a", logLine)},
			})

			group := out.Groups[0]
			Expect(group.RuleID).To(Equal(rules.IDFrameworkConfigNotFound))
			Expect(findConfigRow(group.Instances[0].RelatedConfigs, "日志中的读取路径").Danger).To(Equal(wantDanger))
			Expect(group.Solutions[0]).To(ContainSubstring(wantFirstSolution))
		},
		Entry("absolute path differs from config",
			"open /data/trpc_go.yaml: no such file or directory", true, "改为一致"),
		Entry("relative path depends on working directory",
			"open trpc_go.yaml: no such file or directory", true, "相对路径"),
		Entry("same absolute path still missing",
			"open /app/conf/trpc_go.yaml: no such file or directory", false, "对比日志中的报错"),
		Entry("no path can be extracted",
			"load trpc_go.yaml failed: file not found", false, "对比日志中的报错"),
	)

	DescribeTable("does not treat a non-config missing file as framework-config-not-found",
		func(logLine string) {
			out := new(Diagnoser).diagnose(&rules.Snapshot{
				Deploy:          makeDeploy(deploystatus.DeployPhaseSucceeded),
				FrameworkConfig: trpcFrameworkConfig,
				Instances:       []*rules.Instance{makeCrashingInstance("pod-c", logLine)},
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

	It("does not emit framework-config-not-found for apps without a framework config file", func() {
		// Helm 类应用、blank 框架的 FrameworkConfig 为零值
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			Deploy: makeDeploy(deploystatus.DeployPhaseSucceeded),
			Instances: []*rules.Instance{
				makeCrashingInstance("pod-h", "open /data/trpc_go.yaml: no such file or directory"),
			},
		})

		Expect(out.Groups).To(HaveLen(1))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDUnknown))
		Expect(out.Groups[0].Actions).To(Equal([]rules.Action{
			{ID: rules.ActionDeployConfig, Label: "前往部署配置"},
			{ID: rules.ActionRedeploy, Label: "重新部署"},
		}))
	})

	It("groups by rule in rule-set order and sorts instances by restart count", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			Deploy:          makeDeploy(deploystatus.DeployPhaseFailed),
			FrameworkConfig: trpcFrameworkConfig,
			Instances: []*rules.Instance{
				makeCrashingInstance("pod-unknown", "panic: unexpected nil pointer"),
				{
					ID:           "pod-config-1",
					RestartCount: 1,
					LogLines:     []string{"open /data/trpc_go.yaml: no such file or directory"},
				},
				{
					ID:           "pod-config-9",
					RestartCount: 9,
					LogLines:     []string{"open /data/trpc_go.yaml: no such file or directory"},
				},
			},
		})

		Expect(out.AbnormalCount).To(Equal(3))
		Expect(out.Groups).To(HaveLen(2))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDFrameworkConfigNotFound))
		Expect(out.Groups[0].Instances[0].ID).To(Equal("pod-config-9"))
		Expect(out.Groups[0].Instances[1].ID).To(Equal("pod-config-1"))
		Expect(out.Groups[1].RuleID).To(Equal(rules.IDUnknown))
	})

	It("keeps pod events on the instance and workload events on the group", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			Deploy:          makeDeploy(deploystatus.DeployPhaseSucceeded),
			InstancesListed: true,
			Events: []rules.Event{
				{InstanceID: "pod-a", Content: "pod event", Level: "Warning"},
				{InstanceID: "pod-b", Content: "other pod", Level: "Warning"},
				{ResourceKind: "Deployment", ResourceName: "demo", Content: "quota exceeded", Level: "Warning"},
			},
			Instances: []*rules.Instance{makeCrashingInstance("pod-a")},
		})

		Expect(out.Groups).To(HaveLen(1))
		Expect(out.Groups[0].Instances[0].RelatedEvents).To(Equal([]rules.Event{
			{InstanceID: "pod-a", Level: "Warning", Content: "pod event"},
		}))
		Expect(out.Groups[0].RelatedEvents).To(Equal([]rules.Event{
			{ResourceKind: "Deployment", ResourceName: "demo", Level: "Warning", Content: "quota exceeded"},
		}))
	})

	It("attaches workload events when deploy failed and there is no instance", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			Deploy: makeDeploy(deploystatus.DeployPhaseFailed),
			Events: []rules.Event{{Content: "quota exceeded", Level: "Warning"}},
		})

		Expect(out.Groups).To(HaveLen(1))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDDeployFailed))
		Expect(out.Groups[0].RelatedEvents).To(Equal([]rules.Event{{Content: "quota exceeded", Level: "Warning"}}))
	})

	It("copies partial reasons from the snapshot", func() {
		out := new(Diagnoser).diagnose(&rules.Snapshot{
			InstancesListed: true,
			PartialReasons:  []string{PartialReasonLogs, PartialReasonEvents},
			Deploy:          makeDeploy(deploystatus.DeployPhaseSucceeded),
			Instances:       []*rules.Instance{{ID: "pod-a", IsHealthy: true}},
		})

		Expect(out.PartialReasons).To(Equal([]string{PartialReasonLogs, PartialReasonEvents}))
		Expect(out.EmptyReason).To(Equal(EmptyReasonAllHealthy))
	})
})

var _ = DescribeTable("buildStartCommand",
	func(workload appmodel.Workload, want string) {
		Expect(buildStartCommand(workload)).To(Equal(want))
	},
	Entry("joins command with args",
		appmodel.Workload{Command: []string{"/app/server"}, Args: []string{"-conf", "/app/conf/trpc_go.yaml"}},
		"/app/server -conf /app/conf/trpc_go.yaml"),
	Entry("command only", appmodel.Workload{Command: []string{"run"}}, "run"),
	Entry("nothing configured", appmodel.Workload{}, ""),
)
