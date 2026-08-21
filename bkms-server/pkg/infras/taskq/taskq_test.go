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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

type sampleArgs struct {
	ID   string `json:"id"`
	Step int    `json:"step"`
}

type stringerArgs struct {
	ID string `json:"id"`
}

func (a stringerArgs) String() string {
	return "id=" + a.ID
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

	It("omits args from the log prefix unless Args implements Stringer", func() {
		Expect(formatHandlerArgs(sampleArgs{ID: "secret"})).To(BeEmpty())
		Expect(formatHandlerArgs(stringerArgs{ID: "abc"})).To(Equal(" args=id=abc"))
	})

	It("logs ErrFixedRetry as in progress rather than a failure", func() {
		Expect(formatHandlerResult(nil)).To(BeEmpty())
		Expect(formatHandlerResult(errors.Wrap(ErrFixedRetry, "ticket in progress"))).
			To(Equal(" in_progress=ticket in progress: taskq: retry with fixed interval"))
		Expect(formatHandlerResult(errors.Wrap(ErrStopRetry, "bad arg"))).
			To(Equal(" err=bad arg: taskq: stop retry"))
		Expect(formatHandlerResult(stderrors.New("boom"))).To(Equal(" err=boom"))
	})

	It("flattens multi-line errors so one execution stays one log line", func() {
		// wrapStopRetry 用 errors.Join, Error() 以 \n 拼接, 不压平会被日志采集按行切开
		result := formatHandlerResult(wrapStopRetry(stderrors.New("boom")))
		Expect(result).NotTo(ContainSubstring("\n"))
		Expect(result).To(Equal(" err=boom; skip retry for the task"))
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

var _ = Describe("Test taskq exhausted handler", func() {
	It("decodes payload and passes the last error to the registered callback", func() {
		var (
			gotArgs sampleArgs
			gotErr  error
		)
		tt := NewTaskType[sampleArgs]("test.exhausted", func(_ context.Context, _ sampleArgs) error {
			return nil
		}).OnExhausted(func(_ context.Context, args sampleArgs, lastErr error) {
			gotArgs, gotErr = args, lastErr
		})

		fn, ok := getExhaustedHandler(tt.Name())
		Expect(ok).To(BeTrue())
		payload := lo.Must(wrapEnvelope(
			auth.User{ID: "tester"}, lo.Must(json.Marshal(sampleArgs{ID: "abc", Step: 7})),
		))
		safeCallExhaustedHandler(context.Background(), fn, tt.Name(), payload, stderrors.New("boom"))

		Expect(gotArgs.ID).To(Equal("abc"))
		Expect(gotArgs.Step).To(Equal(7))
		Expect(gotErr).To(MatchError("boom"))
	})

	It("recovers panic from the callback to keep the worker alive", func() {
		tt := NewTaskType[sampleArgs]("test.exhausted-panic", func(_ context.Context, _ sampleArgs) error {
			return nil
		}).OnExhausted(func(_ context.Context, _ sampleArgs, _ error) {
			panic("callback exploded")
		})

		fn, ok := getExhaustedHandler(tt.Name())
		Expect(ok).To(BeTrue())
		payload := lo.Must(json.Marshal(sampleArgs{ID: "abc"}))
		Expect(func() {
			safeCallExhaustedHandler(context.Background(), fn, tt.Name(), payload, stderrors.New("boom"))
		}).NotTo(Panic())
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

		ctx := auth.WithUser(context.Background(), auth.User{ID: "tester"})
		err := Enqueue(ctx, task)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("client not initialized"))
	})

	It("Enqueue returns error when auth user is missing", func() {
		tt := NewTaskType[sampleArgs]("test.nouser", func(_ context.Context, _ sampleArgs) error {
			return nil
		})
		task := tt.NewTask(sampleArgs{ID: "x"})
		Expect(task).NotTo(BeNil())

		err := Enqueue(context.Background(), task)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("auth user not found"))
	})
})

var _ = Describe("Test taskq auth envelope", func() {
	It("wraps and restores auth user without mixing into business args", func() {
		argsPayload := lo.Must(json.Marshal(sampleArgs{ID: "build-1", Step: 2}))
		user := auth.User{ID: "alice", Cred: auth.UserCredential{AccessToken: "tok"}}
		wrapped := lo.Must(wrapEnvelope(user, argsPayload))
		Expect(string(wrapped)).To(ContainSubstring(`"_authUser"`))
		Expect(string(wrapped)).To(ContainSubstring(`"_args"`))

		ctx, rawArgs := restoreEnvelope(context.Background(), wrapped)
		gotUser, err := auth.GetUser(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(gotUser.ID).To(Equal("alice"))
		Expect(gotUser.Cred.AccessToken).To(Equal("tok"))

		var args sampleArgs
		Expect(json.Unmarshal(rawArgs, &args)).To(Succeed())
		Expect(args.ID).To(Equal("build-1"))
		Expect(args.Step).To(Equal(2))
	})

	It("Handler restores envelope user into context", func() {
		var gotUser auth.User
		var got sampleArgs
		task := NewTaskType[sampleArgs]("with-auth", func(ctx context.Context, a sampleArgs) error {
			var err error
			gotUser, err = auth.GetUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			got = a
			return nil
		})
		argsPayload := lo.Must(json.Marshal(sampleArgs{ID: "x", Step: 3}))
		wrapped := lo.Must(wrapEnvelope(auth.User{ID: "bob"}, argsPayload))
		err := task.Handler()(context.Background(), asynq.NewTask("with-auth", wrapped))
		Expect(err).NotTo(HaveOccurred())
		Expect(gotUser.ID).To(Equal("bob"))
		Expect(got.ID).To(Equal("x"))
		Expect(got.Step).To(Equal(3))
	})

	It("Handler still decodes raw args payload without envelope", func() {
		var got sampleArgs
		task := NewTaskType[sampleArgs]("raw", func(_ context.Context, a sampleArgs) error {
			got = a
			return nil
		})
		payload, _ := json.Marshal(sampleArgs{ID: "legacy", Step: 1})
		err := task.Handler()(context.Background(), asynq.NewTask("raw", payload))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID).To(Equal("legacy"))
	})
})
