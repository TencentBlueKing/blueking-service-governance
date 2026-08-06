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

package instance

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// fakeEnvClient 实现 client.Client 接口中 ListEnvs 方法的 fake。
type fakeEnvClient struct {
	client.Client
	envs []client.Env
	err  error
}

func (f *fakeEnvClient) ListEnvs(_ context.Context, _ string) ([]client.Env, error) {
	return f.envs, f.err
}

var _ = Describe("preflightCheckEnvType", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should reject production environment", func() {
		cli := &fakeEnvClient{
			envs: []client.Env{
				{Name: "test-env", Type: "test"},
				{Name: "prod-env", Type: "production"},
			},
		}
		err := preflightCheckEnvType(ctx, cli, "ws-test", "prod-env")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("non-production"))
	})

	It("should allow test environment", func() {
		cli := &fakeEnvClient{
			envs: []client.Env{
				{Name: "test-env", Type: "test"},
			},
		}
		err := preflightCheckEnvType(ctx, cli, "ws-test", "test-env")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow development environment", func() {
		cli := &fakeEnvClient{
			envs: []client.Env{
				{Name: "dev-env", Type: "development"},
			},
		}
		err := preflightCheckEnvType(ctx, cli, "ws-test", "dev-env")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow staging environment", func() {
		cli := &fakeEnvClient{
			envs: []client.Env{
				{Name: "staging-env", Type: "staging"},
			},
		}
		err := preflightCheckEnvType(ctx, cli, "ws-test", "staging-env")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return error when ListEnvs fails", func() {
		cli := &fakeEnvClient{
			err: errors.New("network error"),
		}
		err := preflightCheckEnvType(ctx, cli, "ws-test", "test-env")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("network error"))
	})

	It("should skip check when env not found in list (feature env)", func() {
		cli := &fakeEnvClient{
			envs: []client.Env{
				{Name: "test-env", Type: "test"},
			},
		}
		err := preflightCheckEnvType(ctx, cli, "ws-test", "feature-env-xyz")
		Expect(err).NotTo(HaveOccurred())
	})
})
