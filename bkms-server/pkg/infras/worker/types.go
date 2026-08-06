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
	"sync"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/ctxkey"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// ---------------------------- Task Definition & Registry ----------------------------

// taskName 任务名称
type taskName string

// taskFunc 任务函数，接受特定参数并返回执行结果 & 错误
type taskFunc[Args any, Result any] func(ctx context.Context, args Args) (Result, error)

// definition 任务定义
type definition struct {
	// NewArgs 用于创建空参数实例
	NewArgs func() any
	// NewResult 用于创建空执行结果实例
	NewResult func() any
	// ExecFunc 任务执行函数
	ExecFunc func(ctx context.Context, args any) (result any, err error)
}

// registry 任务注册表，记录任务名称到对应任务的映射
type registry struct {
	mapping map[taskName]definition
	sync.RWMutex
}

// get 获取任务定义
func (r *registry) get(name taskName) (definition, bool) {
	r.RLock()
	defer r.RUnlock()

	d, ok := r.mapping[name]
	return d, ok
}

// set 设置任务定义
func (r *registry) set(name taskName, def definition) {
	r.Lock()
	defer r.Unlock()

	// 所有任务必须在启动时注册，因此名称冲突时直接 panic
	if _, ok := r.mapping[name]; ok {
		panic(fmt.Sprintf("task %s already registered", name))
	}

	r.mapping[name] = def
}

// ----------------------------------- Task Message -----------------------------------

// Message 异步任务消息
//
// 字段说明：
// - TaskName：任务名称，用于在消费侧通过 registry 找到对应的处理函数
// - AuthUser：发起任务的用户身份（包括用户 ID 与 Credential），由 Producer 从 ctx 中提取后写入
// - Data：任务参数 Json，由具体任务自行反序列化
type Message struct {
	TaskName taskName        `json:"taskName"`
	AuthUser auth.User       `json:"authUser,omitempty"`
	Data     json.RawMessage `json:"data"`
}

// ContextMutator Context 变更器，适用于根据 msg 内容，使用 WithValue 等方法构造 Context
type ContextMutator func(ctx context.Context, msg Message) (context.Context, error)

// newMessage 构造异步任务消息
//
// 通过 ctxkey.AuthUser 从 ctx 中读取已认证用户身份并写入消息体。
// 若 ctx 中不存在已认证用户（即非授权场景），将返回错误，禁止派发匿名任务。
func newMessage(ctx context.Context, taskName taskName, args any) (Message, error) {
	msg := Message{TaskName: taskName}

	// 从 Context 中读取用户身份信息
	user, ok := ctx.Value(ctxkey.AuthUser).(auth.User)
	if !ok || user.ID == "" {
		return msg, errors.Errorf("auth user not found in context for task '%s'", taskName)
	}
	msg.AuthUser = user

	// 将参数序列化为 Json
	data, err := json.Marshal(args)
	if err != nil {
		return msg, errors.Wrapf(err, "marshal args for task '%s'", taskName)
	}
	msg.Data = data

	return msg, nil
}

// BuildContext 根据消息内容构造消费侧的 Context。
//
// AuthUser 非空时恢复用户身份；AuthUser 为空时返回错误，调用方应据此 Nack 该消息。
func (m Message) BuildContext(mutators ...ContextMutator) (context.Context, error) {
	ctx := context.Background()

	user, err := m.resolveAuthUser()
	if err != nil {
		return ctx, errors.Wrapf(err, "resolve auth user for task '%s'", m.TaskName)
	}
	ctx = context.WithValue(ctx, ctxkey.AuthUser, user)

	// 执行 Context 变更器（调用方提供）
	for _, mutator := range mutators {
		if ctx, err = mutator(ctx, m); err != nil {
			return ctx, errors.Wrapf(err, "mutate context for task '%s'", m.TaskName)
		}
	}
	return ctx, nil
}

// resolveAuthUser 从消息中解析出用户身份。
func (m Message) resolveAuthUser() (auth.User, error) {
	if m.AuthUser.ID != "" {
		return m.AuthUser, nil
	}

	return auth.User{}, errors.New("auth user not found in message")
}
