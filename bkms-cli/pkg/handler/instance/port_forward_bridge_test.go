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
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

var _ = Describe("port-forward tunnel bridge", func() {
	It("stops listener and idle connections when context is canceled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		fake := &fakePortForwardTunnelClient{
			tunnelFactory: func() io.ReadWriteCloser {
				return newBlockingReadWriteCloser()
			},
			tunnelOpened: make(chan struct{}),
		}
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
			errCh <- RunPortForwardListener(
				ctx,
				fake,
				*cfg,
			)
		}()

		Eventually(func() error {
			conn, err := net.DialTimeout("tcp", cfg.ListenAddress(), 100*time.Millisecond)
			if err != nil {
				return err
			}
			return conn.Close()
		}, time.Second).Should(Succeed())
		localConn, err := net.Dial("tcp", cfg.ListenAddress())
		Expect(err).NotTo(HaveOccurred())
		defer localConn.Close()
		Eventually(fake.tunnelOpened, time.Second).Should(BeClosed())

		cancel()

		Eventually(errCh, time.Second).Should(Receive(Succeed()))
	})

	It("closes local connection and tunnel when bridge context is canceled", func() {
		localSide, cliSide := net.Pipe()
		defer localSide.Close()
		defer cliSide.Close()

		tunnel := newBlockingReadWriteCloser()
		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			errCh <- copyBidirectional(ctx, cliSide, tunnel)
		}()

		cancel()

		Eventually(errCh, time.Second).Should(Receive(Succeed()))
		buf := make([]byte, 1)
		_, err := localSide.Read(buf)
		Expect(err).To(HaveOccurred())
		Eventually(tunnel.closed, time.Second).Should(BeClosed())
	})

	It("copies bytes between the local connection and the remote tunnel", func() {
		localSide, cliSide := net.Pipe()
		defer localSide.Close()
		defer cliSide.Close()

		remoteSide, serverSide := net.Pipe()
		defer remoteSide.Close()
		defer serverSide.Close()

		fake := &fakePortForwardTunnelClient{tunnel: remoteSide}
		cfg := &PortForwardOptions{
			AppID:        "myapp",
			EnvName:      "test",
			InstanceID:   "pod-1",
			RemotePort:   8080,
			LocalPort:    18080,
			LocalAddress: "127.0.0.1",
		}
		errCh := make(chan error, 1)
		go func() {
			errCh <- handlePortForwardConnection(
				context.Background(),
				fake,
				*cfg,
				cliSide,
			)
		}()

		_, err := localSide.Write([]byte("ping"))
		Expect(err).NotTo(HaveOccurred())
		buf := make([]byte, 4)
		_, err = io.ReadFull(serverSide, buf)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buf)).To(Equal("ping"))

		_, err = serverSide.Write([]byte("pong"))
		Expect(err).NotTo(HaveOccurred())
		buf = make([]byte, 4)
		_, err = io.ReadFull(localSide, buf)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(buf)).To(Equal("pong"))

		Expect(localSide.Close()).To(Succeed())
		Eventually(errCh).Should(Receive(Succeed()))
		Expect(fake.opts).To(Equal(client.PortForwardTunnelOptions{
			InstanceID: cfg.InstanceID,
			RemotePort: cfg.RemotePort,
			LocalPort:  cfg.LocalPort,
		}))
	})

	It("opens an independent tunnel for each local TCP connection", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tunnelAClient, tunnelAServer := net.Pipe()
		defer tunnelAClient.Close()
		defer tunnelAServer.Close()
		tunnelBClient, tunnelBServer := net.Pipe()
		defer tunnelBClient.Close()
		defer tunnelBServer.Close()

		probeTunnel := newBlockingReadWriteCloser()
		tunnels := make(chan io.ReadWriteCloser, 3)
		tunnels <- probeTunnel
		tunnels <- tunnelAClient
		tunnels <- tunnelBClient
		fake := &fakePortForwardTunnelClient{
			tunnelFactory: func() io.ReadWriteCloser {
				return <-tunnels
			},
		}
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
			errCh <- RunPortForwardListener(
				ctx,
				fake,
				*cfg,
			)
		}()

		Eventually(func() error {
			conn, err := net.DialTimeout("tcp", cfg.ListenAddress(), 100*time.Millisecond)
			if err != nil {
				return err
			}
			return conn.Close()
		}, time.Second).Should(Succeed())
		// Wait until the readiness probe has consumed its tunnel before opening
		// real connections, so tunnel assignment does not depend on goroutine order.
		Eventually(probeTunnel.closed, time.Second).Should(BeClosed())

		localA, err := net.Dial("tcp", cfg.ListenAddress())
		Expect(err).NotTo(HaveOccurred())
		defer localA.Close()

		_, err = localA.Write([]byte("aaaa"))
		Expect(err).NotTo(HaveOccurred())
		bufA := make([]byte, 4)
		_, err = io.ReadFull(tunnelAServer, bufA)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(bufA)).To(Equal("aaaa"))

		localB, err := net.Dial("tcp", cfg.ListenAddress())
		Expect(err).NotTo(HaveOccurred())
		defer localB.Close()

		_, err = localB.Write([]byte("bbbb"))
		Expect(err).NotTo(HaveOccurred())
		bufB := make([]byte, 4)
		_, err = io.ReadFull(tunnelBServer, bufB)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(bufB)).To(Equal("bbbb"))

		cancel()
		Eventually(errCh, time.Second).Should(Receive(Succeed()))
	})

	It("closes local connection when remote tunnel is closed", func() {
		localSide, cliSide := net.Pipe()
		defer localSide.Close()
		defer cliSide.Close()

		remoteSide, serverSide := net.Pipe()
		defer remoteSide.Close()
		defer serverSide.Close()

		fake := &fakePortForwardTunnelClient{tunnel: remoteSide}
		cfg := &PortForwardOptions{
			AppID:        "myapp",
			EnvName:      "test",
			InstanceID:   "pod-1",
			RemotePort:   8080,
			LocalPort:    18080,
			LocalAddress: "127.0.0.1",
		}
		errCh := make(chan error, 1)
		go func() {
			errCh <- handlePortForwardConnection(context.Background(), fake, *cfg, cliSide)
		}()

		Expect(serverSide.Close()).To(Succeed())
		buf := make([]byte, 1)
		_, err := localSide.Read(buf)
		Expect(err).To(HaveOccurred())
		Eventually(errCh).Should(Receive(Succeed()))
	})

	It("completes silently when remote tunnel is closed normally", func() {
		localSide, cliSide := net.Pipe()
		defer localSide.Close()
		defer cliSide.Close()

		remoteSide, serverSide := net.Pipe()
		defer remoteSide.Close()
		defer serverSide.Close()

		fake := &fakePortForwardTunnelClient{tunnel: remoteSide}
		cfg := &PortForwardOptions{
			AppID:        "myapp",
			EnvName:      "test",
			InstanceID:   "pod-1",
			RemotePort:   8080,
			LocalPort:    18080,
			LocalAddress: "127.0.0.1",
		}
		errCh := make(chan error, 1)
		go func() {
			errCh <- handlePortForwardConnection(context.Background(), fake, *cfg, cliSide)
		}()

		Expect(serverSide.Close()).To(Succeed())
		Eventually(errCh).Should(Receive(Succeed()))
	})

	It("transparently reports server error when opening tunnel fails", func() {
		localSide, cliSide := net.Pipe()
		defer localSide.Close()
		defer cliSide.Close()

		serverErr := errors.New("connect target pod: k8s resource not found (HTTP 500 Internal Server Error)")
		fake := &fakePortForwardTunnelClient{err: serverErr}
		cfg := &PortForwardOptions{
			AppID:        "myapp",
			EnvName:      "test",
			InstanceID:   "pod-1",
			RemotePort:   8080,
			LocalPort:    18080,
			LocalAddress: "127.0.0.1",
		}
		err := handlePortForwardConnection(
			context.Background(),
			fake,
			*cfg,
			cliSide,
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("open port-forward tunnel failed"))
		// Server 端的错误信息应该透传给用户。
		Expect(err.Error()).To(ContainSubstring("k8s resource not found"))
	})

	It("returns wrapped error when tunnel open fails", func() {
		localSide, cliSide := net.Pipe()
		defer localSide.Close()
		defer cliSide.Close()

		fake := &fakePortForwardTunnelClient{err: errFakeTunnel}
		cfg := &PortForwardOptions{
			AppID:        "myapp",
			EnvName:      "test",
			InstanceID:   "pod-1",
			RemotePort:   8080,
			LocalPort:    18080,
			LocalAddress: "127.0.0.1",
		}
		err := handlePortForwardConnection(
			context.Background(),
			fake,
			*cfg,
			cliSide,
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("open port-forward tunnel failed"))
		Expect(err.Error()).To(ContainSubstring(errFakeTunnel.Error()))
	})
})

type fakePortForwardTunnelClient struct {
	mu            sync.Mutex
	tunnel        io.ReadWriteCloser
	tunnelFactory func() io.ReadWriteCloser
	err           error
	opts          client.PortForwardTunnelOptions

	tunnelOpened   chan struct{}
	tunnelOpenOnce sync.Once
}

func (f *fakePortForwardTunnelClient) OpenPortForwardTunnel(
	ctx context.Context,
	appID, envName string,
	opts client.PortForwardTunnelOptions,
) (io.ReadWriteCloser, error) {
	f.mu.Lock()
	f.opts = opts
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	if f.tunnelOpened != nil {
		f.tunnelOpenOnce.Do(func() { close(f.tunnelOpened) })
	}
	if f.tunnelFactory != nil {
		return f.tunnelFactory(), nil
	}
	return f.tunnel, nil
}

type blockingReadWriteCloser struct {
	closed chan struct{}
	once   sync.Once
}

func newBlockingReadWriteCloser() *blockingReadWriteCloser {
	return &blockingReadWriteCloser{closed: make(chan struct{})}
}

func (b *blockingReadWriteCloser) Read(p []byte) (int, error) {
	<-b.closed
	return 0, io.ErrClosedPipe
}

func (b *blockingReadWriteCloser) Write(p []byte) (int, error) {
	<-b.closed
	return 0, io.ErrClosedPipe
}

func (b *blockingReadWriteCloser) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}

var errFakeTunnel = bytes.ErrTooLarge
