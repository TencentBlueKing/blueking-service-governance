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

package taskq

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"time"

	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type sampleArgs struct {
	ID   string `json:"id"`
	Step int    `json:"step"`
}

var _ = Describe("Test taskq TaskType handler", func() {
	It("decodes payload into typed args when executed", func() {
		var got sampleArgs
		task := NewTaskType[sampleArgs]("decode", func(_ context.Context, a sampleArgs) error {
			got = a
			return nil
		})
		payload, _ := json.Marshal(sampleArgs{ID: "x", Step: 3})
		err := task.Handler()(context.Background(), asynq.NewTask("decode", payload))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID).To(Equal("x"))
		Expect(got.Step).To(Equal(3))
	})

	It("wraps StopRetry on bad args payload", func() {
		task := NewTaskType[sampleArgs]("bad", func(_ context.Context, _ sampleArgs) error { return nil })
		err := task.Handler()(context.Background(), asynq.NewTask("bad", []byte("not-json")))
		Expect(err).To(HaveOccurred())
		// 反序列化失败应停止重试。
		Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
	})

	It("wraps StopRetry when handler returns ErrStopRetry", func() {
		task := NewTaskType[sampleArgs]("skip", func(_ context.Context, _ sampleArgs) error {
			return errors.Wrap(ErrStopRetry, "bad arg")
		})
		payload := lo.Must(json.Marshal(sampleArgs{ID: "a"}))
		err := task.Handler()(context.Background(), asynq.NewTask("skip", payload))
		Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
	})

	It("returns error as-is for retryable failures", func() {
		task := NewTaskType[sampleArgs]("retry", func(_ context.Context, _ sampleArgs) error {
			return errors.Wrap(ErrFixedRetry, "still running")
		})
		payload := lo.Must(json.Marshal(sampleArgs{ID: "a"}))
		err := task.Handler()(context.Background(), asynq.NewTask("retry", payload))
		Expect(err).To(HaveOccurred())
		// 非 StopRetry, 交由重试策略处理。
		Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeFalse())
		Expect(stderrors.Is(err, ErrFixedRetry)).To(BeTrue())
	})
})

var _ = Describe("Test taskq retry semantics", func() {
	It("returns fixed interval when error wraps ErrFixedRetry", func() {
		fn := retryDelayFunc(30 * time.Second)
		err := errors.Wrap(ErrFixedRetry, "still running")
		Expect(fn(1, err, asynq.NewTask("x", nil))).To(Equal(30 * time.Second))
	})

	It("uses per-task fixed interval over the global default", func() {
		_ = NewTaskType[sampleArgs]("test.fixed-interval", func(_ context.Context, _ sampleArgs) error {
			return nil
		}).WithFixedRetryInterval(45 * time.Second)

		fn := retryDelayFunc(5 * time.Second)
		err := errors.Wrap(ErrFixedRetry, "still running")
		Expect(fn(1, err, asynq.NewTask("test.fixed-interval", nil))).To(Equal(45 * time.Second))
		Expect(fn(1, err, asynq.NewTask("other", nil))).To(Equal(5 * time.Second))
	})

	It("uses default backoff for non-fixed errors", func() {
		interval := 5 * time.Second
		fn := retryDelayFunc(interval)
		// 非 ErrFixedRetry 走默认退避
		Expect(fn(1, stderrors.New("transient"), asynq.NewTask("x", nil))).NotTo(Equal(interval))
	})

	It("wrapStopRetry makes error match asynq.SkipRetry", func() {
		base := stderrors.New("terminal")
		err := wrapStopRetry(base)
		Expect(stderrors.Is(err, asynq.SkipRetry)).To(BeTrue())
		Expect(stderrors.Is(err, base)).To(BeTrue())
	})
})

var _ = Describe("Test taskq NewTask and Enqueue", func() {
	It("NewTask produces a Task with correct name and serialized payload", func() {
		tt := NewTaskType[sampleArgs]("test.newtask", func(_ context.Context, _ sampleArgs) error {
			return nil
		})
		task := tt.NewTask(sampleArgs{ID: "abc", Step: 7})
		Expect(task).NotTo(BeNil())
		Expect(task.name).To(Equal("test.newtask"))
		Expect(task.payload).To(ContainSubstring(`"id":"abc"`))
		Expect(task.payload).To(ContainSubstring(`"step":7`))
	})

	It("NewTask returns nil for non-serializable args", func() {
		type bad struct {
			Ch chan int `json:"ch"`
		}
		tt := NewTaskType[bad]("test.bad", func(_ context.Context, _ bad) error {
			return nil
		})
		task := tt.NewTask(bad{Ch: make(chan int)})
		Expect(task).To(BeNil())
	})

	It("Enqueue returns error for nil task", func() {
		err := Enqueue(context.Background(), nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("task is nil"))
	})

	It("Enqueue returns error when client is not initialized", func() {
		origClient := client
		client = nil
		defer func() { client = origClient }()

		tt := NewTaskType[sampleArgs]("test.noclient", func(_ context.Context, _ sampleArgs) error {
			return nil
		})
		task := tt.NewTask(sampleArgs{ID: "x"})
		Expect(task).NotTo(BeNil())

		err := Enqueue(context.Background(), task)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("client not initialized"))
	})
})
