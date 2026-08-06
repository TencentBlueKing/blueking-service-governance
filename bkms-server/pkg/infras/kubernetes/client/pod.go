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

package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/trace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
)

// 默认超时时间
const (
	// defaultRequestTimeout 默认请求超时时间，10秒
	defaultRequestTimeout = 30 * time.Second
	// defaultDialTimeout 默认连接超时时间，10秒
	defaultDialTimeout = 10 * time.Second
	// errorStreamTimeout 错误流超时时间，100毫秒
	errorStreamTimeout = 100 * time.Millisecond
	// metadataServiceIP 云厂商元数据服务地址，不允许作为 Pod IP 使用
	metadataServiceIP = "169.254.169.254"
)

// PodClient k8s pod 资源客户端
type PodClient struct {
	Client
}

// NewPodClient 新建 k8s 资源客户端
func NewPodClient(cfg *cluster.Config) *PodClient {
	return &PodClient{Client{cli: dynamic.NewForConfigOrDie(cfg.Rest), cfg: cfg, gvr: gvr.Po}}
}

// GetFirstContainerName 获取指定 Pod 的第一个容器名称
//
// 用于在运行时动态获取 Pod 的容器名，适用于非 bkms 自建 AppModel 类型的应用（其容器名
// 不一定为 "main"）。仅读取 Spec.Containers[0].Name，不考虑 Spec.InitContainers。
func (c *PodClient) GetFirstContainerName(ctx context.Context, namespace, podName string) (string, error) {
	clientSet, err := kubernetes.NewForConfig(c.cfg.Rest)
	if err != nil {
		return "", errors.Wrap(err, "create kubernetes clientset")
	}

	pod, err := clientSet.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "get pod '%s/%s'", namespace, podName)
	}

	if len(pod.Spec.Containers) == 0 {
		return "", errors.Errorf("pod '%s/%s' has no containers", namespace, podName)
	}

	return pod.Spec.Containers[0].Name, nil
}

// ListLogs 获取 pod 日志
func (c *PodClient) ListLogs(
	ctx context.Context, namespace, podName string, opts *corev1.PodLogOptions,
) ([]LogEntry, error) {
	if opts == nil {
		opts = &corev1.PodLogOptions{}
	}
	// 强制要求提供日志时间戳
	opts.Timestamps = true

	clientSet, err := kubernetes.NewForConfig(c.cfg.Rest)
	if err != nil {
		return nil, errors.Wrap(err, "create kubernetes clientset")
	}

	rawLogsContent, err := clientSet.CoreV1().Pods(namespace).GetLogs(podName, opts).DoRaw(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "get logs for pod '%s/%s'", namespace, podName)
	}

	// 解析日志内容
	var logs []LogEntry
	for _, line := range strings.Split(string(rawLogsContent), "\n") {
		if line == "" {
			continue
		}
		timestamp, content, _ := strings.Cut(line, " ")
		// Pod 日志可能包含非 UTF-8 字节（如二进制数据），而 protobuf string
		// 字段要求必须是合法 UTF-8，因此需要将非法字节替换为 Unicode 替换字符
		// uFFFD -> �
		content = strings.ToValidUTF8(content, "\uFFFD")
		logs = append(logs, LogEntry{Timestamp: timestamp, Content: content})
	}

	return logs, nil
}

// OpenLogsStream 打开 pod 日志流
func (c *PodClient) OpenLogsStream(
	ctx context.Context, namespace, podName string, opts *corev1.PodLogOptions,
) (io.ReadCloser, error) {
	if opts == nil {
		opts = &corev1.PodLogOptions{}
	}
	// 强制要求提供日志时间戳
	opts.Timestamps = true

	clientSet, err := kubernetes.NewForConfig(c.cfg.Rest)
	if err != nil {
		return nil, errors.Wrap(err, "create kubernetes clientset")
	}

	reader, err := clientSet.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "stream logs for pod '%s/%s'", namespace, podName)
	}
	return reader, nil
}

// SendHTTPRequest 通过 port-forward 向 pod 发送 HTTP 请求
//
// 该方法通过 Kubernetes port-forward 机制与 Pod 建立连接，将 HTTP 请求发送到指定端口，
// 适用于需要直接访问 Pod 内部服务的场景。
func (c *PodClient) SendHTTPRequest(
	rootCtx context.Context, namespace, podName string, port int32, req *http.Request,
) (*http.Response, error) {
	requestID := c.getRequestID(rootCtx)
	// 检查 context 是否已经有 deadline，如果没有则设置默认超时
	ctx, cancel := c.ensureContextTimeout(rootCtx, defaultRequestTimeout)
	if cancel != nil {
		defer cancel()
	}

	// 预处理请求 body
	if err := c.prepareRequestBody(req); err != nil {
		return nil, err
	}

	// 创建 kubernetes clientset
	clientSet, err := kubernetes.NewForConfig(c.cfg.Rest)
	if err != nil {
		return nil, errors.Wrap(err, "create kubernetes clientset")
	}

	// 构建 portforward 请求
	spdyReq := clientSet.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	// 创建 SPDY Transport 和 Dialer
	transport, upgrader, err := spdy.RoundTripperFor(c.cfg.Rest)
	if err != nil {
		return nil, errors.Wrap(err, "create spdy round tripper")
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   defaultDialTimeout,
	}
	dialer := spdy.NewDialer(upgrader, httpClient, http.MethodPost, spdyReq.URL())

	// 建立 SPDY 连接
	streamConn, err := c.dialWithTimeout(ctx, dialer)
	if err != nil {
		return nil, errors.Wrapf(err, "dial port-forward to pod '%s' in namespace '%s'", podName, namespace)
	}
	defer streamConn.Close()

	// 创建 error stream
	errorStream, err := c.createErrorStream(streamConn, port, requestID)
	if err != nil {
		return nil, errors.Wrapf(err, "create error stream to pod '%s' in namespace '%s'", podName, namespace)
	}
	defer errorStream.Close()

	// 异步读取错误流
	errorChan := c.readErrorStreamAsync(ctx, errorStream)

	// 创建 data stream
	dataStream, err := c.createDataStream(streamConn, port, requestID)
	if err != nil {
		return nil, errors.Wrapf(err, "create data stream to pod '%s' in namespace '%s'", podName, namespace)
	}
	defer dataStream.Close()

	// 执行 HTTP 请求并获取响应
	resp, err := c.executeHTTPRequest(ctx, dataStream, req, errorChan)
	if err != nil {
		return nil, errors.Wrapf(err, "execute HTTP request to pod '%s' in namespace '%s'", podName, namespace)
	}

	return resp, nil
}

// ensureContextTimeout 确保 context 有超时设置
func (c *PodClient) ensureContextTimeout(
	ctx context.Context,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return context.WithTimeout(ctx, timeout)
	}
	return ctx, nil
}

// prepareRequestBody 预处理请求 body，设置 Content-Length
func (c *PodClient) prepareRequestBody(req *http.Request) error {
	if req.Body == nil {
		// GET 请求或没有 Body 的请求，明确设置 ContentLength 为 0
		req.ContentLength = 0
		return nil
	}

	// 如果有 Body，需要读取并计算长度
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return errors.Wrap(err, "read request body")
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.ContentLength = int64(len(bodyBytes))

	return nil
}

// dialResult SPDY 连接结果
type dialResult struct {
	conn httpstream.Connection
	err  error
}

// dialWithTimeout 带超时的 SPDY 连接
func (c *PodClient) dialWithTimeout(ctx context.Context, dialer httpstream.Dialer) (httpstream.Connection, error) {
	dialChan := make(chan dialResult, 1)

	go func() {
		streamConn, _, err := dialer.Dial(portforward.PortForwardProtocolV1Name)
		dialChan <- dialResult{conn: streamConn, err: err}
	}()

	select {
	case result := <-dialChan:
		if result.err != nil {
			return nil, errors.Wrap(result.err, "dial spdy connection")
		}
		return result.conn, nil
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "dial timeout or cancelled")
	}
}

// createErrorStream 创建错误流
func (c *PodClient) createErrorStream(
	conn httpstream.Connection, port int32, requestID string,
) (httpstream.Stream, error) {
	header := http.Header{}
	header.Set(corev1.StreamType, corev1.StreamTypeError)
	header.Set(corev1.PortHeader, fmt.Sprintf("%d", port))
	header.Set(corev1.PortForwardRequestIDHeader, requestID)

	stream, err := conn.CreateStream(header)
	if err != nil {
		return nil, errors.Wrapf(err, "create error stream for port %d", port)
	}
	return stream, nil
}

// createDataStream 创建数据流
func (c *PodClient) createDataStream(
	conn httpstream.Connection, port int32, requestID string,
) (httpstream.Stream, error) {
	header := http.Header{}
	header.Set(corev1.StreamType, corev1.StreamTypeData)
	header.Set(corev1.PortHeader, fmt.Sprintf("%d", port))
	header.Set(corev1.PortForwardRequestIDHeader, requestID)

	stream, err := conn.CreateStream(header)
	if err != nil {
		return nil, errors.Wrapf(err, "create data stream for port %d", port)
	}

	return stream, nil
}

// readErrorStreamAsync 异步读取错误流
func (c *PodClient) readErrorStreamAsync(ctx context.Context, errorStream httpstream.Stream) <-chan error {
	errorChan := make(chan error, 1)

	go func() {
		defer close(errorChan)

		message, err := io.ReadAll(errorStream)
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			errorChan <- ctx.Err()
			return
		default:
		}

		if err != nil && err != io.EOF {
			errorChan <- errors.Wrap(err, "read from error stream")
			return
		}
		if len(message) > 0 {
			errorChan <- errors.Errorf("error from pod: %s", string(message))
			return
		}
		errorChan <- nil
	}()

	return errorChan
}

// httpResult HTTP 请求结果
type httpResult struct {
	resp *http.Response
	body []byte
	err  error
}

// executeHTTPRequest 执行 HTTP 请求并获取响应
func (c *PodClient) executeHTTPRequest(
	ctx context.Context, dataStream httpstream.Stream, req *http.Request, errorChan <-chan error,
) (*http.Response, error) {
	resultChan := make(chan httpResult, 1)

	go func() {
		defer close(resultChan)

		// 将 HTTP 请求写入 data stream
		if err := req.Write(dataStream); err != nil {
			resultChan <- httpResult{err: errors.Wrap(err, "write HTTP request to stream")}
			return
		}

		// 关闭写入端（half-close），告知服务端请求已发送完毕
		// 但保持读取端打开以接收响应
		if closer, ok := dataStream.(interface{ CloseWrite() error }); ok {
			_ = closer.CloseWrite()
		}

		// 从 data stream 读取 HTTP 响应
		resp, err := http.ReadResponse(bufio.NewReader(dataStream), req)
		if err != nil {
			resultChan <- httpResult{err: errors.Wrap(err, "read HTTP response from stream")}
			return
		}

		// 读取响应 body 并创建新的 response（因为原始 stream 会被关闭）
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			resultChan <- httpResult{err: errors.Wrap(err, "read response body")}
			return
		}

		resultChan <- httpResult{resp: resp, body: body, err: nil}
	}()

	// 等待 HTTP 响应完成
	var result httpResult
	var resultReceived bool

	for !resultReceived {
		select {
		case result = <-resultChan:
			resultReceived = true
		case streamErr := <-errorChan:
			// 错误流有错误，立即返回
			if streamErr != nil {
				return nil, streamErr
			}
			// 错误流正常关闭（无错误），继续等待 HTTP 响应
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "request timeout or cancelled")
		}
	}

	// 检查 HTTP 请求处理结果
	if result.err != nil {
		return nil, result.err
	}

	// 短暂检查错误流是否有延迟到达的错误
	select {
	case streamErr := <-errorChan:
		if streamErr != nil {
			return nil, streamErr
		}
	case <-time.After(errorStreamTimeout):
		// 超时，没有错误
	}

	// 构建新的响应，包含读取的 body
	result.resp.Body = io.NopCloser(bytes.NewReader(result.body))

	return result.resp, nil
}

// getRequestID 获取请求 ID
func (c *PodClient) getRequestID(ctx context.Context) string {
	requestID := trace.GetTraceID(ctx)
	if requestID != "" {
		return requestID
	}

	requestID = trace.GetSpanID(ctx)
	if requestID != "" {
		return requestID
	}

	return cast.ToString(time.Now().UnixNano())
}

// TCPPortForward TCP 端口转发会话
//
// 通过 K8s port-forward 机制在本地创建 TCP 监听器，将 TCP 连接桥接到 Pod 指定端口。
type TCPPortForward struct {
	// LocalPort 本地监听端口
	LocalPort int

	listener   net.Listener
	streamConn httpstream.Connection
	closed     chan struct{}
	closeOnce  sync.Once
}

// Close 关闭端口转发，释放所有资源（幂等）
func (pf *TCPPortForward) Close() {
	pf.closeOnce.Do(func() {
		close(pf.closed)
		if pf.listener != nil {
			_ = pf.listener.Close()
		}
		if pf.streamConn != nil {
			_ = pf.streamConn.Close()
		}
	})
}

// CreateTCPPortForward 创建 TCP 端口转发
//
// 在本地监听随机端口，当有 TCP 连接进入时，通过 K8s SPDY data stream 将流量桥接到
// Pod 的指定端口。调用者通过 TCPPortForward.LocalPort 获取本地端口号，使用完毕后
// 必须调用 Close() 释放资源。
//
// 注意：当前实现仅接受一个连接（适用于单次请求-响应场景）。
func (c *PodClient) CreateTCPPortForward(
	ctx context.Context, namespace, podName string, port int32,
) (*TCPPortForward, error) {
	// 创建 kubernetes clientset
	clientSet, err := kubernetes.NewForConfig(c.cfg.Rest)
	if err != nil {
		return nil, errors.Wrap(err, "create kubernetes clientset")
	}

	// 构建 portforward 请求
	spdyReq := clientSet.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	// 创建 SPDY Transport 和 Dialer
	transport, upgrader, err := spdy.RoundTripperFor(c.cfg.Rest)
	if err != nil {
		return nil, errors.Wrap(err, "create spdy round tripper")
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   defaultDialTimeout,
	}
	dialer := spdy.NewDialer(upgrader, httpClient, http.MethodPost, spdyReq.URL())

	// 建立 SPDY 连接
	streamConn, err := c.dialWithTimeout(ctx, dialer)
	if err != nil {
		return nil, errors.Wrapf(err, "dial port-forward to pod '%s' in namespace '%s'", podName, namespace)
	}
	// 在本地监听随机端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		streamConn.Close()
		return nil, errors.Wrap(err, "listen on local port")
	}

	localPort := listener.Addr().(*net.TCPAddr).Port

	pf := &TCPPortForward{
		LocalPort:  localPort,
		listener:   listener,
		streamConn: streamConn,
		closed:     make(chan struct{}),
	}

	// 后台 goroutine：接受连接并按照 kubectl portforward 标准实现桥接到 SPDY data stream
	go func() {
		defer pf.Close()

		conn, err := listener.Accept()
		if err != nil {
			// listener 被 Close 时会返回错误，属正常退出
			return
		}
		defer conn.Close()

		requestID := c.getRequestID(ctx)

		// 创建 error stream（对齐 kubectl 实现：Accept 后创建，立即 Close 写端，保留读端）
		errorStream, err := c.createErrorStream(streamConn, port, requestID)
		if err != nil {
			log.Errorf(ctx, "[TCPPortForward] create error stream failed: %v", err)
			return
		}
		// 关闭写端：告知 kubelet 本端不会向 error stream 写数据（保留读端接收错误信息）
		errorStream.Close()
		defer streamConn.RemoveStreams(errorStream)

		// 异步读取 error stream（如果 kubelet 报错会写入这里）
		errorChan := make(chan error, 1)
		go func() {
			message, readErr := io.ReadAll(errorStream)
			if readErr != nil {
				errorChan <- fmt.Errorf("read error stream: %w", readErr)
			} else if len(message) > 0 {
				errorChan <- fmt.Errorf("error from pod: %s", string(message))
			}
			close(errorChan)
		}()

		// 创建 data stream
		dataStream, err := c.createDataStream(streamConn, port, requestID)
		if err != nil {
			log.Errorf(ctx, "[TCPPortForward] create data stream failed: %v", err)
			return
		}
		defer streamConn.RemoveStreams(dataStream)

		remoteDone := make(chan struct{})
		localError := make(chan struct{})

		// SPDY -> TCP（远端到本地）
		go func() {
			io.Copy(conn, dataStream) //nolint:errcheck
			close(remoteDone)
		}()

		// TCP -> SPDY（本地到远端），完成后关闭 data stream 写端
		go func() {
			defer dataStream.Close()
			_, copyErr := io.Copy(dataStream, conn)
			if copyErr != nil {
				close(localError)
			}
		}()

		// 等待远端完成或本地出错
		select {
		case <-remoteDone:
		case <-localError:
		case <-pf.closed:
		}

		// Reset dataStream 避免阻塞 errorChan（对齐 kubectl 标准注释）
		_ = dataStream.Reset()

		// 等待 error channel，输出 kubelet 报错
		if podErr := <-errorChan; podErr != nil {
			log.Errorf(ctx, "[TCPPortForward] pod error: %v", podErr)
		}
	}()

	return pf, nil
}

// ResolvePodIPFromManifest 从 Pod manifest 中提取并校验 Pod IP
// 含运行状态检查、IP 合法性校验和安全限制校验，适用于 port-forward 等需要安全防护的场景
func ResolvePodIPFromManifest(manifest map[string]any) (string, error) {
	if !isPodRunning(manifest) {
		return "", errors.New("pod is not running")
	}
	podIP := mapx.GetStr(manifest, "status.podIP")
	if podIP == "" {
		return "", errors.New("pod IP is empty")
	}
	parsedPodIP := net.ParseIP(podIP)
	if parsedPodIP == nil {
		return "", errors.New("pod IP is invalid")
	}
	if isForbiddenPodIP(parsedPodIP) {
		return "", errors.New("pod IP is forbidden")
	}
	return podIP, nil
}

// isPodRunning 判断 Pod 是否处于 Running 状态
func isPodRunning(manifest map[string]any) bool {
	return mapx.GetStr(manifest, "status.phase") == "Running"
}

// isForbiddenPodIP 判断 Pod IP 是否属于禁止访问的地址范围
//
// 禁止使用本地回环、链路本地、未指定地址，以及云厂商元数据服务地址，避免绕过
// port-forward 的安全边界访问节点本地或敏感元数据服务
func isForbiddenPodIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.Equal(net.ParseIP(metadataServiceIP))
}
