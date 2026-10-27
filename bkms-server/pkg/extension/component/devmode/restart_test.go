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

package devmode_test

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	devmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
)

// fakePodExecutor 按脚本内容区分「读取 pid.conf」与「stop」两类命令，模拟 pod 内执行结果
type fakePodExecutor struct {
	mu sync.Mutex
	// readPID 第 n 次（从 1 开始）读取 pid.conf 的结果
	readPID func(pod string, n int) (*k8sclient.ExecResult, error)
	// stop 执行 stop 命令的结果
	stop func(pod string) (*k8sclient.ExecResult, error)

	readCalls map[string]int
	stopCalls map[string]int
	cmds      [][]string
	container string
}

func newFakePodExecutor() *fakePodExecutor {
	return &fakePodExecutor{
		readPID: func(string, int) (*k8sclient.ExecResult, error) {
			return &k8sclient.ExecResult{ExitCode: 1}, nil
		},
		stop: func(string) (*k8sclient.ExecResult, error) {
			return &k8sclient.ExecResult{}, nil
		},
		readCalls: map[string]int{},
		stopCalls: map[string]int{},
	}
}

func (f *fakePodExecutor) Exec(
	_ context.Context, _, pod, container string, cmd []string,
) (*k8sclient.ExecResult, error) {
	f.mu.Lock()
	f.cmds = append(f.cmds, cmd)
	f.container = container
	isStop := strings.Contains(cmd[2], "stop.sh")
	var n int
	if isStop {
		f.stopCalls[pod]++
	} else {
		f.readCalls[pod]++
		n = f.readCalls[pod]
	}
	f.mu.Unlock()

	if isStop {
		return f.stop(pod)
	}
	return f.readPID(pod, n)
}

func (f *fakePodExecutor) counts(pod string) (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.readCalls[pod], f.stopCalls[pod]
}

var _ = Describe("Restarter", func() {
	var records *devmode.PublishRecordStore
	var ctx context.Context
	var appID string
	var executor *fakePodExecutor
	var restarter *devmode.Restarter

	newTarget := func(envName string, instances ...string) devmode.RestartTarget {
		return devmode.RestartTarget{
			EnvName:     envName,
			Namespace:   "ns-" + envName,
			Container:   "main",
			MountPath:   "/data/bkms/dev-mode/trpc/configmap-scripts",
			InstanceIDs: instances,
		}
	}

	// listRecords 按环境列出 restart 记录，新的在前
	listRecords := func(envName string) []devmode.PublishRecord {
		items, err := records.List(ctx, devmode.ByAppEnv(appID, envName), devmode.NewestFirst())
		Expect(err).NotTo(HaveOccurred())
		return items
	}

	BeforeEach(func() {
		records = testutil.NewStore(GinkgoT(), devmode.PublishRecordMeta)
		ctx = testutil.WithTenantOrDefault(context.Background())
		appID = "test-app-" + stringx.Random(6)
		executor = newFakePodExecutor()
		restarter = devmode.NewRestarter(
			records,
			devmode.WithRestartPollInterval(10*time.Millisecond),
			devmode.WithRestartWaitStartTimeout(300*time.Millisecond),
		)
	})

	// executeAll 依次创建记录并执行，模拟异步任务逐个执行
	executeAll := func(
		reqCtx context.Context, targets []devmode.RestartTarget,
	) ([]devmode.RestartInstanceResult, error) {
		prepared, err := restarter.Prepare(reqCtx, appID, "admin", targets)
		if err != nil {
			return nil, err
		}
		results := make([]devmode.RestartInstanceResult, 0, len(prepared))
		for _, item := range prepared {
			results = append(results, restarter.Execute(
				reqCtx, appID, item.RecordID, item.Target, item.Instance, executor,
			))
		}
		return results, nil
	}

	It("restarts the process and waits until pid.conf is recreated", func() {
		executor.readPID = func(_ string, n int) (*k8sclient.ExecResult, error) {
			switch n {
			case 1:
				return &k8sclient.ExecResult{Stdout: "100\n"}, nil
			case 2:
				// monitor 尚未拉起，pid.conf 不存在
				return &k8sclient.ExecResult{ExitCode: 1}, nil
			case 3:
				// 偶发 exec 异常不中断轮询
				return nil, errors.New("connection reset")
			default:
				return &k8sclient.ExecResult{Stdout: "200\n"}, nil
			}
		}

		results, err := executeAll(ctx, []devmode.RestartTarget{newTarget("stag", "pod-1")})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].EnvName).To(Equal("stag"))
		Expect(results[0].Instance).To(Equal("pod-1"))
		Expect(results[0].Status).To(Equal(devmode.RecordStatusSuccess))
		Expect(results[0].OldPID).To(Equal("100"))
		Expect(results[0].NewPID).To(Equal("200"))

		_, stopCalls := executor.counts("pod-1")
		Expect(stopCalls).To(Equal(1))
		// 脚本挂载路径通过位置参数 $1 传入
		Expect(executor.cmds[0][:2]).To(Equal([]string{"bash", "-c"}))
		Expect(executor.cmds[0][3:]).To(Equal([]string{"bash", "/data/bkms/dev-mode/trpc/configmap-scripts"}))
		Expect(executor.container).To(Equal("main"))

		saved := listRecords("stag")
		Expect(saved).To(HaveLen(1))
		Expect(saved[0].Action).To(Equal(devmode.RecordActionRestart))
		Expect(saved[0].Status).To(Equal(devmode.RecordStatusSuccess))
		Expect(saved[0].Operator).To(Equal("admin"))
		Expect(saved[0].Message).To(ContainSubstring("100 -> 200"))
	})

	It("fails without polling when stop.sh exits with non-zero code", func() {
		executor.readPID = func(string, int) (*k8sclient.ExecResult, error) {
			return &k8sclient.ExecResult{Stdout: "100"}, nil
		}
		executor.stop = func(string) (*k8sclient.ExecResult, error) {
			return &k8sclient.ExecResult{ExitCode: 21, Stderr: "remaining process\n"}, nil
		}

		results, err := executeAll(ctx, []devmode.RestartTarget{newTarget("stag", "pod-1")})
		Expect(err).NotTo(HaveOccurred())
		Expect(results[0].Status).To(Equal(devmode.RecordStatusFailed))
		Expect(results[0].Message).To(ContainSubstring("code 21"))
		Expect(results[0].Message).To(ContainSubstring("remaining process"))

		readCalls, _ := executor.counts("pod-1")
		Expect(readCalls).To(Equal(1))

		saved := listRecords("stag")
		Expect(saved[0].Status).To(Equal(devmode.RecordStatusFailed))
	})

	It("fails when pid.conf is not recreated before timeout", func() {
		results, err := executeAll(ctx, []devmode.RestartTarget{newTarget("stag", "pod-1")})
		Expect(err).NotTo(HaveOccurred())
		Expect(results[0].Status).To(Equal(devmode.RecordStatusFailed))
		Expect(results[0].OldPID).To(BeEmpty())
		Expect(results[0].Message).To(ContainSubstring("timed out waiting for monitor"))
	})

	It("does not stop the process when reading pid.conf fails", func() {
		executor.readPID = func(string, int) (*k8sclient.ExecResult, error) {
			return nil, errors.New("pod not found")
		}

		results, err := executeAll(ctx, []devmode.RestartTarget{newTarget("stag", "pod-1")})
		Expect(err).NotTo(HaveOccurred())
		Expect(results[0].Status).To(Equal(devmode.RecordStatusFailed))
		Expect(results[0].Message).To(ContainSubstring("pod not found"))

		_, stopCalls := executor.counts("pod-1")
		Expect(stopCalls).To(BeZero())
	})

	It("keeps result order across envs and isolates failures between instances", func() {
		executor.readPID = func(pod string, n int) (*k8sclient.ExecResult, error) {
			if n == 1 {
				return &k8sclient.ExecResult{Stdout: "1"}, nil
			}
			return &k8sclient.ExecResult{Stdout: "2"}, nil
		}
		executor.stop = func(pod string) (*k8sclient.ExecResult, error) {
			if pod == "pod-b" {
				return &k8sclient.ExecResult{ExitCode: 1}, nil
			}
			return &k8sclient.ExecResult{}, nil
		}

		results, err := executeAll(ctx, []devmode.RestartTarget{
			newTarget("stag", "pod-a", "pod-b"),
			newTarget("test", "pod-c"),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(3))
		Expect([]string{results[0].Instance, results[1].Instance, results[2].Instance}).
			To(Equal([]string{"pod-a", "pod-b", "pod-c"}))
		Expect(results[0].Status).To(Equal(devmode.RecordStatusSuccess))
		Expect(results[1].Status).To(Equal(devmode.RecordStatusFailed))
		Expect(results[2].Status).To(Equal(devmode.RecordStatusSuccess))
		Expect(results[2].EnvName).To(Equal("test"))

		stagRecords := listRecords("stag")
		Expect(stagRecords).To(HaveLen(2))
		testRecords := listRecords("test")
		Expect(testRecords).To(HaveLen(1))
	})

	It("keeps waiting for the process to start after the request is cancelled during stop", func() {
		reqCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		executor.readPID = func(_ string, n int) (*k8sclient.ExecResult, error) {
			if n == 1 {
				return &k8sclient.ExecResult{Stdout: "100"}, nil
			}
			return &k8sclient.ExecResult{Stdout: "200"}, nil
		}
		// stop 执行期间请求被取消（如 CLI 超时断开）
		executor.stop = func(string) (*k8sclient.ExecResult, error) {
			cancel()
			return &k8sclient.ExecResult{}, nil
		}

		results, err := executeAll(reqCtx, []devmode.RestartTarget{newTarget("stag", "pod-1")})
		Expect(err).NotTo(HaveOccurred())
		Expect(results[0].Status).To(Equal(devmode.RecordStatusSuccess))
		Expect(results[0].NewPID).To(Equal("200"))

		saved := listRecords("stag")
		Expect(saved[0].Status).To(Equal(devmode.RecordStatusSuccess))
	})

	It("returns an error without stopping anything when the request is cancelled before execution", func() {
		reqCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := executeAll(reqCtx, []devmode.RestartTarget{newTarget("stag", "pod-1")})
		Expect(err).To(HaveOccurred())

		readCalls, stopCalls := executor.counts("pod-1")
		Expect(readCalls).To(BeZero())
		Expect(stopCalls).To(BeZero())
	})

	It("does nothing when there are no target instances", func() {
		results, err := executeAll(ctx, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(BeEmpty())
	})
})
