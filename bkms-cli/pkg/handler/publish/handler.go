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

// Package publish 开发模式业务逻辑
package publish

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5" //nolint:gosec // MD5 仅用于文件完整性校验，非安全用途
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	osfilepath "path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// 常量定义
const (
	// maxUploadSize 最大上传文件大小：5GB
	maxUploadSize = (1 << 30) * 5
	// defaultContainerName 默认容器名称
	defaultContainerName = "main"

	// publishStatusSuccess 发布成功状态
	publishStatusSuccess = "success"
	// publishStatusFailed 发布失败状态
	publishStatusFailed = "failed"
)

// Publisher 开发模式文件发布处理器。
type Publisher struct {
	ctx     context.Context
	cli     client.Client
	appID   string
	envName string

	devModeBinPath    string
	restartScriptPath string

	// 以下参数由 Publisher 完成 PreCheck 后设置
	preflight *client.DevModePreflightData
	// Kubernetes 相关 clients
	restConfig *rest.Config
	clientset  *kubernetes.Clientset
}

// NewPublisher 创建发布处理器。
func NewPublisher(ctx context.Context, cli client.Client, appID, envName string) *Publisher {
	return &Publisher{
		ctx:     ctx,
		cli:     cli,
		appID:   appID,
		envName: envName,
	}
}

// PreCheck 执行发布前检查：
// 通过 server 端 DevMode Preflight 接口一次性完成所有校验并获取 publish 所需的上下文信息。
// instanceIDs 为需要发布的实例 ID 列表（publishAll=false 时使用），publishAll 为 true 时由 server 端获取所有 Running 实例。
func (h *Publisher) PreCheck(instanceIDs []string, publishAll bool) error {
	preflight, err := h.cli.DevModePublishPreflight(h.ctx, h.appID, h.envName, instanceIDs, publishAll)
	if err != nil {
		return errors.Wrapf(err, "publish preflight failed for app %s env %s", h.appID, h.envName)
	}

	// 缓存后续发布阶段需要使用的上下文信息
	h.preflight = preflight
	if preflight.DevMode == nil {
		return errors.New("server returned empty devMode config")
	}
	// WorkPath 由 server 端根据应用类型（trpc/taf）返回，如 /data/bkms/dev-mode/trpc 或 /data/bkms/dev-mode/taf
	h.devModeBinPath = osfilepath.Join(preflight.DevMode.WorkPath, "/bin")
	h.restartScriptPath = osfilepath.Join(preflight.DevMode.MountPath, "/restart.sh")

	console.Info("WorkPath: %s, MountPath: %s", preflight.DevMode.WorkPath, preflight.DevMode.MountPath)

	return nil
}

// GetInstanceIDs 获取 server 端校验后返回的实例 ID 列表。
func (h *Publisher) GetInstanceIDs() []string {
	if h.preflight == nil {
		return nil
	}
	return h.preflight.InstanceIDs
}

// Publish 执行发布：
// 1. 确认指定的文件是否存在，且大小合法
// 2. 使用 preflight 返回的 Token 和 API 地址初始化 k8s client
// 3. 计算文件 MD5
// 4. 生成随机文件名
// 5. 预先压缩文件为 tar.gz 格式
// 6. 逐个 pod 上传文件并执行 restart.sh，收集每个实例的发布结果
// 7. 上报逐实例发布结果到 server
func (h *Publisher) Publish(filePath string, instanceIDs []string) error {
	if h.preflight == nil {
		return errors.New("preCheck must be called before publish")
	}
	if len(instanceIDs) == 0 {
		return errors.New("no instances to publish")
	}
	slog.Info(fmt.Sprintf("This publish will operate on %d instances", len(instanceIDs)), "instanceIDs", instanceIDs)

	// 1. 确认指定的文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return errors.Wrapf(err, "file %s not found", filePath)
	}
	// 检查文件大小（5GB 以内）
	if fileInfo.Size() > maxUploadSize {
		return errors.Errorf("file size %.2f MB exceeds maximum allowed size %.2f MB",
			float64(fileInfo.Size())/(1024*1024),
			float64(maxUploadSize)/(1024*1024))
	}

	// 2. 使用 preflight 返回的 Token 和 API 地址初始化 k8s client
	console.Info("Connecting to cluster...")
	if buildErr := h.buildKubeClient(h.preflight.Address, h.preflight.Token); buildErr != nil {
		return errors.Wrap(buildErr, "failed to build kubernetes client")
	}

	// 3. 计算文件 MD5
	fileMD5, err := calculateFileMD5(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to calculate file MD5")
	}
	slog.Debug(fmt.Sprintf("File MD5: %s", fileMD5))

	// 4. 生成随机文件名（只需生成一次）
	randomName := generateRandomName()
	slog.Debug(fmt.Sprintf("Random filename: %s", randomName))

	// 5. 预先压缩文件为 tar.gz 格式（只需压缩一次）
	slog.Debug("Compressing file to tar.gz format...")
	tarGzData, err := compressFileToTarGz(filePath, randomName)
	if err != nil {
		return errors.Wrap(err, "failed to compress file to tar.gz")
	}
	namespace := h.preflight.Namespace
	binaryName := osfilepath.Base(filePath)
	slog.Debug(fmt.Sprintf("Compressed size: %.2f MB", float64(len(tarGzData))/(1024*1024)))
	console.Info("==================================================")
	console.Info("Publish workflow: 1. Upload file ==> 2. Execute restart script ==> 3. Done")
	console.Info("==================================================")

	// 6. 逐个 pod 上传文件并执行 restart.sh，失败不中断，继续处理其余实例
	results := make([]client.DevModePublishResult, 0, len(instanceIDs))
	for i, instanceID := range instanceIDs {
		console.Info("\n[%d/%d] Processing instance: %s", i+1, len(instanceIDs), instanceID)
		slog.Debug(fmt.Sprintf("  Random filename: %s", randomName))

		result := client.DevModePublishResult{Instance: instanceID}

		// 上传已压缩的 tar.gz 数据到 pod
		slog.Debug(fmt.Sprintf("  Uploading file to %s...", h.devModeBinPath))
		if err = h.uploadTarGzToPod(h.ctx, tarGzData, instanceID, namespace, h.devModeBinPath); err != nil {
			result.Status = publishStatusFailed
			result.Message = fmt.Sprintf("failed to upload file: %v", err)
			console.Error("  Instance %s failed: %s", instanceID, result.Message)
			results = append(results, result)
			continue
		}
		slog.Debug("  File upload completed!")

		// 执行 restart.sh 脚本
		slog.Debug("  Executing restart.sh script...")
		if err = h.executeRestartScript(h.ctx, instanceID, namespace, h.restartScriptPath, randomName, fileMD5); err != nil {
			result.Status = publishStatusFailed
			result.Message = fmt.Sprintf("failed to execute restart script: %v", err)
			console.Error("  Instance %s failed: %s", instanceID, result.Message)
			results = append(results, result)
			continue
		}

		result.Status = publishStatusSuccess
		results = append(results, result)
		console.Info("  Instance %s restart completed!", instanceID)
	}

	// 7. 上报逐实例发布结果（best-effort，上报失败不影响本次发布结果）
	defer h.report(binaryName, fileInfo.Size(), fileMD5, results)

	// 8. 汇总输出，任一实例失败则返回错误
	failed := lo.Filter(results, func(result client.DevModePublishResult, _ int) bool {
		return result.Status == publishStatusFailed
	})

	console.Info("\n==================================================")
	if len(failed) == 0 {
		console.Info("All instances published successfully!")
		console.Info("==================================================")
		return nil
	}
	console.Error("Publish finished with %d failed instance(s):", len(failed))
	for _, result := range failed {
		console.Error("  - %s: %s", result.Instance, result.Message)
	}
	console.Error("==================================================")
	return errors.Errorf("publish failed on %d instance(s)", len(failed))
}

// report 上报逐实例发布结果到 server（best-effort）。
func (h *Publisher) report(binaryName string, fileSize int64, fileMD5 string, results []client.DevModePublishResult) {
	if len(results) == 0 {
		return
	}
	err := h.cli.ReportDevModePublish(h.ctx, h.appID, h.envName, client.DevModePublishReportOptions{
		BinaryName: binaryName,
		FileSize:   fileSize,
		MD5:        fileMD5,
		Results:    results,
	})
	if err != nil {
		console.Warn("Warning: failed to report publish result to server: %v", err)
	}
}

// buildKubeClient 使用 token 连接集群。
func (h *Publisher) buildKubeClient(clusterAddress, token string) error {
	var err error

	if token == "" {
		return errors.New("token is empty")
	}

	slog.Debug(fmt.Sprintf("Cluster address: %s", clusterAddress))
	h.restConfig = &rest.Config{
		BearerToken:     token,
		Host:            clusterAddress,
		TLSClientConfig: rest.TLSClientConfig{Insecure: true},
	}

	h.clientset, err = kubernetes.NewForConfig(h.restConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	return nil
}

// uploadTarGzToPod 上传已压缩的 tar.gz 数据到 pod
func (h *Publisher) uploadTarGzToPod(ctx context.Context, tarGzData []byte, podName, namespace, binPath string) error {
	// 使用 bytes.Reader 来读取压缩数据
	reader := bytes.NewReader(tarGzData)

	// 在容器中执行：mkdir -p <target-path> && tar -xzf - -C <target-path>
	cmd := []string{
		"sh", "-c",
		fmt.Sprintf("mkdir -p %s && tar -xzf - -C %s", binPath, binPath),
	}

	return h.execInPod(ctx, cmd, reader, podName, namespace, defaultContainerName)
}

// executeRestartScript 在 pod 中执行 restart.sh 脚本
func (h *Publisher) executeRestartScript(
	ctx context.Context,
	podName, namespace, scriptPath, randomName, md5sum string,
) error {
	// 执行 restart.sh 脚本，传入随机名称和 md5 值
	cmd := []string{
		"bash", scriptPath, randomName, md5sum,
	}

	return h.execInPod(ctx, cmd, nil, podName, namespace, defaultContainerName)
}

// execInPod 使用 SPDY 在 Pod 内执行指定命令
func (h *Publisher) execInPod(
	ctx context.Context,
	cmd []string,
	stdin io.Reader,
	podName, namespace, container string,
) error {
	req := h.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdin:     stdin != nil,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(h.restConfig, "POST", req.URL())
	if err != nil {
		return errors.Wrapf(err, "failed to create SPDY executor for pod %s", podName)
	}

	streamOpts := remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	}

	if err = exec.StreamWithContext(ctx, streamOpts); err != nil {
		return errors.Wrapf(err, "failed to execute command %v in pod %s", cmd, podName)
	}

	return nil
}

// calculateFileMD5 计算文件的 MD5 值
func calculateFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New() //nolint:gosec // MD5 仅用于文件完整性校验，非安全用途
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// generateRandomName 生成随机文件名（基于时间戳）
func generateRandomName() string {
	return fmt.Sprintf("bin_%d", time.Now().UnixNano())
}

// compressFileToTarGz 将文件压缩为 tar.gz 格式（只执行一次）
func compressFileToTarGz(filePath, randomName string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file %s", filePath)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "failed to stat file")
	}

	// 创建 buffer 存储压缩后的数据
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	// 写入 tar header
	tarHeader := &tar.Header{
		Name:     randomName,
		Mode:     0o755, // 可执行权限
		Size:     fileInfo.Size(),
		ModTime:  fileInfo.ModTime(),
		Typeflag: tar.TypeReg,
	}
	if err = tarWriter.WriteHeader(tarHeader); err != nil {
		return nil, errors.Wrap(err, "failed to write tar header")
	}

	// 写入文件内容
	if _, err = io.Copy(tarWriter, file); err != nil {
		return nil, errors.Wrap(err, "failed to copy file to tar")
	}

	// 关闭 tarWriter 和 gzipWriter 以刷新数据
	if err = tarWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close tar writer")
	}
	if err = gzipWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close gzip writer")
	}

	return buf.Bytes(), nil
}
