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
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RunPortForwardListener", func() {
	It("starts a local listener and exits when context is cancelled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cfg := &PortForwardOptions{
			AppID:        "myapp",
			EnvName:      "test",
			InstanceID:   "pod-1",
			RemotePort:   8080,
			LocalPort:    freeTCPPort(),
			LocalAddress: "127.0.0.1",
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- RunPortForwardListener(ctx, nil, *cfg)
		}()

		Eventually(func() error {
			conn, err := net.DialTimeout("tcp", cfg.ListenAddress(), 100*time.Millisecond)
			if err != nil {
				return err
			}
			return conn.Close()
		}).Should(Succeed())

		cancel()
		Eventually(errCh).Should(Receive(Succeed()))
	})

	It("starts listener with non-loopback address without error", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cfg := &PortForwardOptions{
			AppID:        "myapp",
			EnvName:      "test",
			InstanceID:   "pod-1",
			RemotePort:   8080,
			LocalPort:    freeTCPPort(),
			LocalAddress: "0.0.0.0",
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- RunPortForwardListener(ctx, nil, *cfg)
		}()

		Eventually(func() error {
			conn, err := net.DialTimeout("tcp", cfg.ListenAddress(), 100*time.Millisecond)
			if err != nil {
				return err
			}
			return conn.Close()
		}).Should(Succeed())

		cancel()
		Eventually(errCh).Should(Receive(Succeed()))
	})

	It("returns a clear error when local port is already in use", func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		defer ln.Close()

		cfg := &PortForwardOptions{
			AppID:        "myapp",
			EnvName:      "test",
			InstanceID:   "pod-1",
			RemotePort:   8080,
			LocalPort:    ln.Addr().(*net.TCPAddr).Port,
			LocalAddress: "127.0.0.1",
		}

		err = RunPortForwardListener(
			context.Background(),
			nil,
			*cfg,
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("listen on 127.0.0.1"))
		Expect(err.Error()).To(ContainSubstring("local port may already be in use"))
	})
})

// freeTCPPort 获取一个可用的本地 TCP 端口。
func freeTCPPort() int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}
