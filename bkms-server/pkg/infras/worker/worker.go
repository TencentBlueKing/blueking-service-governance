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

// Package worker provide async task support
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

const (
	// 最大重连次数
	maxReconnects = 5
	// 初始重试间隔
	initialBackoff = 1 * time.Second
	// 最大重试间隔
	maxBackoff = 30 * time.Second
)

// 消费循环退出原因（sentinel errors）
var (
	// ErrExitStopped 用户主动停止
	ErrExitStopped = errors.New("worker stopped by user request")
	// ErrExitConnectionLost 连接断开
	ErrExitConnectionLost = errors.New("connection lost")
)

// Worker 任务执行器
type Worker struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	queue    string
	consumer string
	// RabbitMQ 连接地址
	uri string
	// 未确认消息的最大数量
	prefetch int
	// Context 变更器
	ctxMutators []ContextMutator
	// 用于优雅退出的控制
	doneChan chan struct{}
	// 用于标识是否主动停止（区分主动停止和异常断开）
	stopChan chan struct{}
	// stopOnce 确保停止信号只发送一次，避免重复调用 Stop 时关闭已关闭的 stopChan
	stopOnce sync.Once
	// 正在处理中的任务，key 是 MessageId（发布时设置的 UUID），value 是 *deliveryHolder
	// 用于断连重连后的 delivery 热替换，避免同一消息被重复投递时启动多个协程
	inFlight sync.Map
}

// deliveryHolder 持有可热替换的 amqp.Delivery 引用
// 当 RabbitMQ 断连重连后，同一消息会被重新投递（新的 delivery），
// 通过 Replace 方法将旧 delivery 替换为新的，使任务协程完成后能用新 delivery 成功 Ack
type deliveryHolder struct {
	mu       sync.Mutex
	delivery amqp.Delivery
	done     bool
}

// Replace 替换为新的 delivery（重连后的新投递）
// 如果任务已完成（done=true），直接 Ack 新 delivery 并返回 false
// 否则替换 delivery 引用并返回 true
func (h *deliveryHolder) Replace(d amqp.Delivery) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		// 任务已完成，直接 Ack 新 delivery
		_ = d.Ack(false)
		return false
	}
	h.delivery = d
	return true
}

// Ack 使用当前最新的 delivery 进行确认，并标记任务已完成
func (h *deliveryHolder) Ack(multiple bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.done = true
	return h.delivery.Ack(multiple)
}

// Nack 使用当前最新的 delivery 进行拒绝，并标记任务已完成
func (h *deliveryHolder) Nack(multiple, requeue bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.done = true
	return h.delivery.Nack(multiple, requeue)
}

// New 新建任务管理器
//
// 参数说明：
// - uri: RabbitMQ 地址（含 vhost），形如：amqps://guest:passwd@127.0.0.1:5672/vhost
// - queue: RabbitMQ 队列名称
// - prefetch: 消息预取数量，为 0 / 过高都可能导致 worker 过载（过量的轮询协程）
// - ctxMutators: Context 变更器，用于根据 Message 内容，构建新的 Context（通过 WithValue 等方式）
//
// 返回值：
// - worker: 新建的 Worker 实例
// - error: 错误信息
//
// NOTE: 目前暂不支持 Options 模式，先只提供必须参数
func New(uri, queue string, prefetch int, ctxMutators []ContextMutator) (*Worker, error) {
	wk := &Worker{
		uri:         uri,
		queue:       queue,
		prefetch:    prefetch,
		ctxMutators: ctxMutators,
		consumer:    fmt.Sprintf("consumer-%s", uuid.New().String()),
		doneChan:    make(chan struct{}),
		stopChan:    make(chan struct{}),
	}

	// 建立初始连接
	if err := wk.connect(); err != nil {
		return nil, err
	}

	return wk, nil
}

// Run 运行 Worker，持续监听队列 & 处理消息
// 当连接断开时会自动尝试重连，重连失败超过阈值会触发 panic
func (w *Worker) Run(ctx context.Context) error {
	go w.consumeLoop(ctx)

	log.Infof(ctx, "consumer started on queue '%s' with consumer '%s'", w.queue, w.consumer)
	return nil
}

// Stop 优雅停止任务管理器，停止接收新任务并等待当前任务完成
func (w *Worker) Stop(ctx context.Context) error {
	log.Info(ctx, "stopping task worker...")

	// 发送停止信号，只发送一次，避免重复调用 Stop 时关闭已关闭的 stopChan
	w.stopOnce.Do(func() {
		close(w.stopChan)
	})

	// Cancel 消费者，停止接收新消息，通过关闭 msgs chan 触发 Worker 协程退出
	if w.channel != nil {
		if err := w.channel.Cancel(w.consumer, false); err != nil {
			log.Errorf(ctx, "failed to cancel consumer: %v", err)
			// 不返回错误，继续等待 doneChan
		} else {
			log.Infof(ctx, "consumer '%s' canceled successfully", w.consumer)
		}
	}

	// 等待 Worker 协程完成（由 defer 触发）
	// 如果 ctx 超时，Stop 只停止继续等待并返回错误，不保证正在执行的任务 goroutine 已经完全退出
	select {
	case <-w.doneChan:
		log.Info(ctx, "task worker stopped successfully")
		return nil
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "stop task worker")
	}
}

// Close 关闭资源
func (w *Worker) Close() error {
	if w.channel != nil {
		_ = w.channel.Close()
	}
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// connect 建立 RabbitMQ 连接和通道
func (w *Worker) connect() error {
	conn, err := amqp.Dial(w.uri)
	if err != nil {
		return errors.Wrapf(err, "connect to RabbitMQ: %s", w.uri)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return errors.Wrapf(err, "open channel")
	}

	// 定义持久化队列（Durable），禁用 AutoDelete, Exclusive, NoWait
	_, err = channel.QueueDeclare(w.queue, true, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return errors.Wrapf(err, "declare queue: %s", w.queue)
	}

	w.conn = conn
	w.channel = channel
	return nil
}

// reconnect 尝试重新建立 RabbitMQ 连接
// 使用指数递增间隔重试策略（1-2-4-8-16...最长 30 秒），最大重试 6 次，超过后触发 panic
func (w *Worker) reconnect(ctx context.Context) {
	// 关闭旧的连接资源
	w.closeConnection()

	backoff := initialBackoff
	for attempt := 1; attempt <= maxReconnects; attempt++ {
		log.Warnf(
			ctx, "attempting to reconnect to RabbitMQ (attempt %d/%d), waiting %v...",
			attempt, maxReconnects, backoff,
		)

		time.Sleep(backoff)

		if err := w.connect(); err != nil {
			log.Errorf(ctx, "reconnect attempt %d failed: %v", attempt, err)
			// 计算下一次退避时间（指数递增，上限为 maxBackoff）
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		log.Info(ctx, "reconnected to RabbitMQ successfully")
		return
	}

	// 超过最大重试次数，触发 panic 让进程中断
	log.Errorf(ctx, "failed to reconnect after %d attempts, triggering panic for process interruption", maxReconnects)
	panic("RabbitMQ connection lost and reconnect failed, triggering process interruption")
}

// closeConnection 关闭现有的连接和通道（忽略错误）
func (w *Worker) closeConnection() {
	if w.channel != nil {
		_ = w.channel.Close()
		w.channel = nil
	}
	if w.conn != nil {
		_ = w.conn.Close()
		w.conn = nil
	}
}

// startConsumer 启动消费者
func (w *Worker) startConsumer(ctx context.Context) (<-chan amqp.Delivery, error) {
	// 设置 prefetch，控制未确认消息的最大数量
	if w.prefetch > 0 {
		if err := w.channel.Qos(w.prefetch, 0, false); err != nil {
			return nil, errors.Wrapf(err, "set prefetch count to %d", w.prefetch)
		}
		log.Infof(ctx, "prefetch count set to %d", w.prefetch)
	}

	// 注册消费者，禁用 AutoAck, Exclusive, NoLocal, NoWait
	msgs, err := w.channel.Consume(w.queue, w.consumer, false, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "register consumer")
	}

	return msgs, nil
}

// consumeLoop 消费循环，支持断线重连
func (w *Worker) consumeLoop(ctx context.Context) {
	defer close(w.doneChan)

	for {
		// 启动消费者
		msgs, err := w.startConsumer(ctx)
		if err != nil {
			log.Errorf(ctx, "failed to start consumer: %v, attempting reconnect...", err)
			w.reconnect(ctx)
			continue
		}

		// 消费消息，根据返回原因决定后续操作
		err = w.processMessages(ctx, msgs)
		switch {
		case errors.Is(err, ErrExitStopped):
			log.Info(ctx, "worker stopped by user request")
			return
		case errors.Is(err, ErrExitConnectionLost):
			log.Warn(ctx, "connection lost, attempting reconnect...")
			w.reconnect(ctx)
		default:
			// 防御性处理：未知原因也尝试重连
			log.Warnf(ctx, "unknown exit reason: %v, attempting reconnect...", err)
			w.reconnect(ctx)
		}
	}
}

// processMessages 处理消息直到通道关闭，返回退出原因
func (w *Worker) processMessages(ctx context.Context, msgs <-chan amqp.Delivery) error {
	for {
		select {
		case <-w.stopChan:
			return ErrExitStopped
		case delivery, ok := <-msgs:
			if !ok {
				// 消息通道已关闭
				log.Info(ctx, "message channel closed")
				return ErrExitConnectionLost
			}
			// 处理消息
			go w.handle(ctx, delivery)
		}
	}
}

// handle 处理 & 执行 RabbitMQ 消息中的任务
//
// 通过 inFlight + deliveryHolder 实现 delivery 热替换：
// - 首次收到消息时，创建 deliveryHolder 并启动任务协程
// - 断连重连后同一消息被重新投递时，仅替换 delivery 引用，不启动新协程
// - 任务完成后，使用最新的 delivery 进行 Ack/Nack（确保走的是活跃的 channel）
func (w *Worker) handle(ctx context.Context, delivery amqp.Delivery) {
	msgID := delivery.MessageId

	// 检查是否已有协程在处理同一条消息（断连重连后的重复投递）
	if existing, loaded := w.inFlight.Load(msgID); loaded {
		holder := existing.(*deliveryHolder)
		if holder.Replace(delivery) {
			log.Infof(ctx, "task message '%s' already in-flight, delivery replaced", msgID)
		} else {
			log.Infof(ctx, "task message '%s' already completed, new delivery acked directly", msgID)
		}
		return
	}

	// 首次处理，创建 holder 并尝试存入 inFlight
	holder := &deliveryHolder{delivery: delivery}
	if existing, loaded := w.inFlight.LoadOrStore(msgID, holder); loaded {
		// 并发竞争：另一个协程刚刚存入了，替换 delivery 即可
		existing.(*deliveryHolder).Replace(delivery)
		return
	}
	defer w.inFlight.Delete(msgID)

	// 1. 解析从队列中提取到的消息
	var msg Message
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		log.Errorf(ctx, "error unmarshal message: %v", err)
		// 格式错误的消息，不重新入队
		_ = holder.Nack(false, false)
		return
	}

	// 2. 查找任务定义
	def, exists := globalRegistry.get(msg.TaskName)
	if !exists {
		log.Errorf(ctx, "task '%s' not found", msg.TaskName)
		// 任务名称错误的消息，不重新入队
		_ = holder.Nack(false, false)
		return
	}
	reportTaskReceived(msg.TaskName)

	// 3. 构造任务执行所需的参数
	args := def.NewArgs()
	if err := json.Unmarshal(msg.Data, args); err != nil {
		log.Errorf(ctx, "error unmarshal args for task '%s': %v", msg.TaskName, err)
		// 参数格式错误的消息，不重新入队
		_ = holder.Nack(false, false)
		return
	}

	// 4. 根据异步任务消息重建 Context
	ctx, err := msg.BuildContext(w.ctxMutators...)
	if err != nil {
		log.Errorf(ctx, "rebuild context for task '%s' failed: %v", msg.TaskName, err)
		// 重建 Context 失败，不重新入队
		_ = holder.Nack(false, false)
		return
	}

	// 5. 执行具体任务
	log.Infof(ctx, "executing task '%s'", msg.TaskName)
	started := time.Now()
	// TODO 目前忽略任务执行结果，后续可能会需要处理
	_, err = def.ExecFunc(ctx, args)
	if err != nil {
		reportTaskExecution(msg.TaskName, statusErr, started)
		log.Errorf(ctx, "execute task '%s' failed: %v", msg.TaskName, err)
		// ExecFunc 中已有重试机制，这里直接 Nack 不重新入队
		_ = holder.Nack(false, false)
		return
	}

	// 6. 确认消息处理成功
	reportTaskExecution(msg.TaskName, statusOK, started)
	log.Infof(ctx, "task '%s' completed successfully", msg.TaskName)
	_ = holder.Ack(false)
}

// apply 下发异步任务
func (w *Worker) apply(ctx context.Context, taskName taskName, args any) (string, error) {
	// 生成任务 ID
	taskID := uuid.New().String()

	// 构造消息（传递用户信息以在 Worker 中继续使用）
	msg, err := newMessage(ctx, taskName, args)
	if err != nil {
		return "", errors.Wrapf(err, "generate message for task '%s'", taskName)
	}

	// 将消息序列化为 Json
	body, err := json.Marshal(msg)
	if err != nil {
		return "", errors.Wrapf(err, "marshal message for task '%s'", taskName)
	}

	// 发布消息到队列，不指定交换机
	err = w.channel.PublishWithContext(
		ctx,
		"",
		w.queue,
		false,
		false,
		// 设置消息 ID & 持久化
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			MessageId:    taskID,
		},
	)
	if err != nil {
		return "", errors.Wrapf(err, "publish message for task '%s'", taskName)
	}

	return taskID, nil
}
