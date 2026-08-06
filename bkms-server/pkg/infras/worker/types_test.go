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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/ctxkey"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("Message", func() {
	const (
		testTaskName = taskName("test-task")
		testUserID   = "u-001"
		testToken    = "tk-xyz"
	)

	authedUser := func() auth.User {
		return auth.User{
			ID:   testUserID,
			Cred: auth.UserCredential{AccessToken: testToken},
		}
	}

	Describe("newMessage", func() {
		Context("when context contains a valid auth user", func() {
			It("should populate AuthUser correctly", func() {
				ctx := context.WithValue(context.Background(), ctxkey.AuthUser, authedUser())

				msg, err := newMessage(ctx, testTaskName, map[string]string{"foo": "bar"})

				Expect(err).NotTo(HaveOccurred())
				Expect(msg.TaskName).To(Equal(testTaskName))
				Expect(msg.AuthUser.ID).To(Equal(testUserID))
				Expect(msg.AuthUser.Cred.AccessToken).To(Equal(testToken))

				// Data must be valid json carrying the args
				var args map[string]string
				Expect(json.Unmarshal(msg.Data, &args)).To(Succeed())
				Expect(args).To(HaveKeyWithValue("foo", "bar"))
			})
		})

		Context("when context contains the maintenance user", func() {
			It("should populate AuthUser with the maintenance identity", func() {
				ctx := auth.WithMaintenanceUser(context.Background())

				msg, err := newMessage(ctx, testTaskName, map[string]string{"foo": "bar"})

				Expect(err).NotTo(HaveOccurred())
				Expect(msg.AuthUser.ID).To(Equal(auth.MaintenanceUserID))
				Expect(msg.AuthUser.Cred.IsEmpty()).To(BeTrue())
			})
		})

		Context("when context has no auth user", func() {
			It("should return an error and refuse to build the message", func() {
				_, err := newMessage(context.Background(), testTaskName, struct{}{})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("auth user not found"))
			})
		})

		Context("when context carries an auth user with empty ID", func() {
			It("should also return an error", func() {
				ctx := context.WithValue(context.Background(), ctxkey.AuthUser, auth.User{})

				_, err := newMessage(ctx, testTaskName, struct{}{})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("auth user not found"))
			})
		})
	})

	Describe("BuildContext", func() {
		Context("when message carries the AuthUser field directly", func() {
			It("should restore the user via ctxkey.AuthUser (primary path)", func() {
				msg := Message{TaskName: testTaskName, AuthUser: authedUser()}

				ctx, err := msg.BuildContext()

				Expect(err).NotTo(HaveOccurred())
				got, ok := ctx.Value(ctxkey.AuthUser).(auth.User)
				Expect(ok).To(BeTrue())
				Expect(got.ID).To(Equal(testUserID))
				Expect(got.Cred.AccessToken).To(Equal(testToken))
			})
		})

		Context("when AuthUser is empty", func() {
			It("should return an error so the caller can Nack the message", func() {
				msg := Message{TaskName: testTaskName}

				_, err := msg.BuildContext()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("resolve auth user"))
			})
		})

		Context("when a custom mutator is provided", func() {
			It("should run mutators after the auth user has been injected", func() {
				type ctxKey string
				const traceKey ctxKey = "trace-id"

				msg := Message{TaskName: testTaskName, AuthUser: authedUser()}
				mutator := func(ctx context.Context, _ Message) (context.Context, error) {
					return context.WithValue(ctx, traceKey, "trace-123"), nil
				}

				ctx, err := msg.BuildContext(mutator)

				Expect(err).NotTo(HaveOccurred())
				Expect(ctx.Value(traceKey)).To(Equal("trace-123"))
				_, ok := ctx.Value(ctxkey.AuthUser).(auth.User)
				Expect(ok).To(BeTrue())
			})
		})
	})
})
