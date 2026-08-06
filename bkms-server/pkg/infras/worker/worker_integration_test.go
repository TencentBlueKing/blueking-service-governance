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

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/ctxkey"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// newTestQueue creates a new unique queue for testing.
func newTestQueue() string {
	return fmt.Sprintf("test-worker-%s", uuid.New().String())
}

// deleteQueue deletes a test queue.
func deleteQueue(uri, queue string) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return
	}
	defer ch.Close()

	_, _ = ch.QueueDelete(queue, false, false, false)
}

// publishRaw publishes raw bytes directly to a queue, bypassing Worker serialization.
func publishRaw(uri, queue string, body []byte) error {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Publish(
		"", queue, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			MessageId:    uuid.New().String(),
		},
	)
}

// queueMessageCount returns the number of messages currently in the queue.
func queueMessageCount(uri, queue string) (int, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return 0, err
	}
	defer ch.Close()

	q, err := ch.QueueDeclarePassive(queue, false, false, false, false, nil)
	if err != nil {
		return 0, err
	}
	return q.Messages, nil
}

// authedTestCtx returns a context populated with a fake authenticated user,
// used to satisfy the auth-user requirement of publisher.apply.
func authedTestCtx() context.Context {
	user := auth.User{
		ID:   "test-user",
		Cred: auth.UserCredential{AccessToken: "test-token"},
	}
	return context.WithValue(context.Background(), ctxkey.AuthUser, user)
}

var _ = Describe("Worker Integration", func() {
	var (
		queue    string
		consumer *Worker
		ctx      context.Context
	)

	BeforeEach(func() {
		if rabbitmqURI == "" {
			Skip("RABBITMQ_URI_FOR_TEST not set, skipping integration tests")
		}
		ctx = authedTestCtx()
		queue = newTestQueue()
	})

	AfterEach(func() {
		if consumer != nil {
			_ = consumer.Stop(ctx)
			_ = consumer.Close()
			consumer = nil
		}
		deleteQueue(rabbitmqURI, queue)
	})

	Describe("End-to-End Task Publish and Consume", func() {
		It("should publish a task and consume it with correct arguments", func() {
			type SimpleArgs struct {
				Name string `json:"name"`
			}
			type SimpleResult struct {
				OK bool `json:"ok"`
			}

			tn := taskName(fmt.Sprintf("test-simple-%s", uuid.New().String()))
			received := make(chan SimpleArgs, 1)

			RegisterTask[SimpleArgs, SimpleResult](
				tn, func(ctx context.Context, args SimpleArgs) (SimpleResult, error) {
					received <- args
					return SimpleResult{OK: true}, nil
				},
			)

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher.Close()

			_, err = publisher.apply(ctx, tn, SimpleArgs{Name: "hello"})
			Expect(err).NotTo(HaveOccurred())

			Eventually(received, 5*time.Second).Should(Receive(Equal(SimpleArgs{Name: "hello"})))
		})

		It("should correctly serialize and deserialize complex struct arguments", func() {
			type Address struct {
				City    string `json:"city"`
				ZipCode string `json:"zipCode"`
			}
			type ComplexArgs struct {
				Name    string  `json:"name"`
				Age     int     `json:"age"`
				Score   float64 `json:"score"`
				Address Address `json:"address"`
			}
			type ComplexResult struct {
				OK bool `json:"ok"`
			}

			tn := taskName(fmt.Sprintf("test-complex-%s", uuid.New().String()))
			received := make(chan ComplexArgs, 1)

			RegisterTask[ComplexArgs, ComplexResult](
				tn, func(ctx context.Context, args ComplexArgs) (ComplexResult, error) {
					received <- args
					return ComplexResult{OK: true}, nil
				},
			)

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher.Close()

			expected := ComplexArgs{
				Name:  "Alice",
				Age:   30,
				Score: 99.5,
				Address: Address{
					City:    "Shenzhen",
					ZipCode: "518000",
				},
			}

			_, err = publisher.apply(ctx, tn, expected)
			Expect(err).NotTo(HaveOccurred())

			Eventually(received, 5*time.Second).Should(Receive(Equal(expected)))
		})
	})

	Describe("Message Ack/Nack Behavior", func() {
		It("should Ack the message when the task succeeds, leaving the queue empty", func() {
			type AckArgs struct {
				Value string `json:"value"`
			}
			type AckResult struct{}

			tn := taskName(fmt.Sprintf("test-ack-%s", uuid.New().String()))
			done := make(chan struct{}, 1)

			RegisterTask[AckArgs, AckResult](tn, func(ctx context.Context, args AckArgs) (AckResult, error) {
				done <- struct{}{}
				return AckResult{}, nil
			})

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher.Close()

			_, err = publisher.apply(ctx, tn, AckArgs{Value: "test"})
			Expect(err).NotTo(HaveOccurred())

			Eventually(done, 5*time.Second).Should(Receive())

			// Wait a bit for Ack to propagate
			time.Sleep(500 * time.Millisecond)

			count, err := queueMessageCount(rabbitmqURI, queue)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(0))
		})

		It("should Nack the message without requeue when the task fails", func() {
			type NackArgs struct {
				Value string `json:"value"`
			}
			type NackResult struct{}

			tn := taskName(fmt.Sprintf("test-nack-%s", uuid.New().String()))
			done := make(chan struct{}, 1)

			RegisterTask[NackArgs, NackResult](tn, func(ctx context.Context, args NackArgs) (NackResult, error) {
				done <- struct{}{}
				return NackResult{}, fmt.Errorf("intentional failure")
			})

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher.Close()

			_, err = publisher.apply(ctx, tn, NackArgs{Value: "fail"})
			Expect(err).NotTo(HaveOccurred())

			Eventually(done, 5*time.Second).Should(Receive())

			// Wait a bit for Nack to propagate
			time.Sleep(500 * time.Millisecond)

			count, err := queueMessageCount(rabbitmqURI, queue)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(0))
		})
	})

	Describe("Error Handling", func() {
		It("should Nack a message with an unregistered task name and continue processing", func() {
			type ValidArgs struct {
				Value string `json:"value"`
			}
			type ValidResult struct{}

			validTN := taskName(fmt.Sprintf("test-valid-after-unknown-%s", uuid.New().String()))
			received := make(chan ValidArgs, 1)

			RegisterTask[ValidArgs, ValidResult](
				validTN, func(ctx context.Context, args ValidArgs) (ValidResult, error) {
					received <- args
					return ValidResult{}, nil
				},
			)

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			// Publish a message with an unregistered task name
			unknownMsg := Message{
				TaskName: taskName(fmt.Sprintf("unknown-task-%s", uuid.New().String())),
				Data:     json.RawMessage(`{"value":"ignored"}`),
			}
			body, err := json.Marshal(unknownMsg)
			Expect(err).NotTo(HaveOccurred())
			Expect(publishRaw(rabbitmqURI, queue, body)).To(Succeed())

			// Then publish a valid task
			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher.Close()

			_, err = publisher.apply(ctx, validTN, ValidArgs{Value: "after-unknown"})
			Expect(err).NotTo(HaveOccurred())

			Eventually(received, 5*time.Second).Should(Receive(Equal(ValidArgs{Value: "after-unknown"})))
		})

		It("should Nack a message with invalid JSON body and continue processing", func() {
			type ValidArgs2 struct {
				Value string `json:"value"`
			}
			type ValidResult2 struct{}

			validTN := taskName(fmt.Sprintf("test-valid-after-badjson-%s", uuid.New().String()))
			received := make(chan ValidArgs2, 1)

			RegisterTask[ValidArgs2, ValidResult2](
				validTN, func(ctx context.Context, args ValidArgs2) (ValidResult2, error) {
					received <- args
					return ValidResult2{}, nil
				},
			)

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			// Publish invalid JSON
			Expect(publishRaw(rabbitmqURI, queue, []byte(`{not valid json}`))).To(Succeed())

			// Then publish a valid task
			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher.Close()

			_, err = publisher.apply(ctx, validTN, ValidArgs2{Value: "after-bad-json"})
			Expect(err).NotTo(HaveOccurred())

			Eventually(received, 5*time.Second).Should(Receive(Equal(ValidArgs2{Value: "after-bad-json"})))
		})
	})

	Describe("Connection Loss and Reconnection", func() {
		It("should reconnect after connection loss and resume consuming", func() {
			type ReConnArgs struct {
				Seq int `json:"seq"`
			}
			type ReConnResult struct{}

			tn := taskName(fmt.Sprintf("test-reconnect-%s", uuid.New().String()))
			received := make(chan ReConnArgs, 2)

			RegisterTask[ReConnArgs, ReConnResult](
				tn, func(ctx context.Context, args ReConnArgs) (ReConnResult, error) {
					received <- args
					return ReConnResult{}, nil
				},
			)

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			// Publish a task before connection loss to verify normal operation
			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = publisher.apply(ctx, tn, ReConnArgs{Seq: 1})
			Expect(err).NotTo(HaveOccurred())
			publisher.Close()

			Eventually(received, 5*time.Second).Should(Receive(Equal(ReConnArgs{Seq: 1})))

			// Forcefully close the consumer's underlying connection to simulate network failure
			consumer.conn.Close()

			// Wait for reconnection (at least initialBackoff = 1s + buffer)
			time.Sleep(3 * time.Second)

			// Publish a new task after reconnection
			publisher2, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher2.Close()

			_, err = publisher2.apply(ctx, tn, ReConnArgs{Seq: 2})
			Expect(err).NotTo(HaveOccurred())

			Eventually(received, 10*time.Second).Should(Receive(Equal(ReConnArgs{Seq: 2})))
		})
	})

	Describe("DeliveryHolder: in-flight deduplication on reconnect", func() {
		type HolderArgs struct {
			Value string `json:"value"`
		}
		type HolderResult struct{}

		// newBlockingTask 注册一个阻塞任务，返回 (执行信号 chan, 放行 chan)
		newBlockingTask := func() (taskName, chan struct{}, chan struct{}) {
			tn := taskName(fmt.Sprintf("test-holder-%s", uuid.New().String()))
			execSignal := make(chan struct{}, 10)
			gate := make(chan struct{})
			RegisterTask[HolderArgs, HolderResult](
				tn, func(_ context.Context, _ HolderArgs) (HolderResult, error) {
					execSignal <- struct{}{}
					<-gate
					return HolderResult{}, nil
				},
			)
			return tn, execSignal, gate
		}

		// simulateReconnect 断开连接并等待重连完成
		simulateReconnect := func(w *Worker) {
			w.conn.Close()
			time.Sleep(3 * time.Second)
		}

		// expectQueueEmpty 断言队列为空
		expectQueueEmpty := func() {
			time.Sleep(1 * time.Second)
			count, err := queueMessageCount(rabbitmqURI, queue)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(0))
		}

		It("should deduplicate re-delivered messages across multiple reconnections", func() {
			tn, execSignal, gate := newBlockingTask()

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher.Close()

			_, err = publisher.apply(ctx, tn, HolderArgs{Value: "dedup"})
			Expect(err).NotTo(HaveOccurred())

			// Task starts exactly once
			Eventually(execSignal, 5*time.Second).Should(Receive())

			// Reconnect 3 times — task must NOT be re-executed
			for i := 0; i < 3; i++ {
				simulateReconnect(consumer)
				Consistently(execSignal, 1*time.Second).ShouldNot(Receive())
			}

			// Release task, verify queue drained via latest delivery
			close(gate)
			expectQueueEmpty()
		})

		It("should directly Ack re-delivered message when task already completed", func() {
			tn, execSignal, gate := newBlockingTask()

			var err error
			consumer, err = New(rabbitmqURI, queue, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumer.Run(ctx)).To(Succeed())

			publisher, err := New(rabbitmqURI, queue, 0, nil)
			Expect(err).NotTo(HaveOccurred())
			defer publisher.Close()

			_, err = publisher.apply(ctx, tn, HolderArgs{Value: "done-before-reconn"})
			Expect(err).NotTo(HaveOccurred())

			// Wait for task to start, then let it complete (done=true)
			Eventually(execSignal, 5*time.Second).Should(Receive())
			close(gate)
			time.Sleep(500 * time.Millisecond)

			// Reconnect — re-delivered message should be directly Acked by Replace
			simulateReconnect(consumer)
			expectQueueEmpty()
		})
	})
})
