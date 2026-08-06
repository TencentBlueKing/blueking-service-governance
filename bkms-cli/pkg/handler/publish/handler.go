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
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	instancehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/instance"
)

// bcsAPIHost BCS API 网关地址，通过 go build -ldflags 注入
var bcsAPIHost = ""

// 常量定义
const (
	// 环境类型：正式环境
	envTypeProduction = "production"
	// 可发布的实例状态
	instanceStatusRunning = "Running"
	// 最大上传文件大小：5GB
	maxUploadSize = (1 << 30) * 5
	// 默认容器名称
	defaultContainerName = "main"
)

// Kubernetes 相关 clients
var (
	restConfig *rest.Config
	clientset  *kubernetes.Clientset
)

// Publisher 开发模式文件发布处理器。
type Publisher struct {
	ctx         context.Context
	cli         client.Client
	workspaceID string
	appID       string
	envName     string

	// 以下参数由 Publisher 完成 PreCheck 后设置
	app               *client.AppMinimal
	env               *client.Env
	devMode           *client.DevModeConfig
	devModeBinPath    string
	restartScriptPath string
}

// NewPublisher 创建发布处理器。
func NewPublisher(ctx context.Context, cli client.Client, workspaceID, appID, envName string) *Publisher {
	return &Publisher{
		ctx:         ctx,
		cli:         cli,
		workspaceID: workspaceID,
		appID:       appID,
		envName:     envName,
	}
}

// PreCheck 执行发布前检查：
// 1. 检查 app 是否存在
// 2. 检查 env 是否存在，且非正式环境
// 3. 确认是否有开启 devmode
// 4. 缓存后续发布阶段需要使用的上下文信息
func (h *Publisher) PreCheck() error {
	// 1. 检查 app 是否存在
	app, err := h.cli.GetAppMinimal(h.ctx, h.workspaceID, h.appID)
	if err != nil {
		return errors.Wrapf(err, "failed to get app %s", h.appID)
	}
	if app == nil {
		return errors.Errorf("app %s not found", h.appID)
	}

	// 2. 检查 env 是否存在，且非正式环境
	env, err := getEnvByName(h.ctx, h.cli, h.workspaceID, h.envName)
	if err != nil {
		return err
	}
	if env.Type == envTypeProduction {
		return errors.Errorf("env %s is a production environment, dev mode is not supported", h.envName)
	}

	// 3. 确认是否有开启 devmode
	devMode, err := h.cli.GetEnvEffectiveDevMode(h.ctx, h.appID, h.envName)
	if err != nil {
		return errors.Wrapf(err, "failed to get effective dev mode for app %s env %s", h.appID, h.envName)
	}
	// DevModeConfig 可能为空
	if devMode == nil {
		return errors.Errorf("dev mode is not configured for app %s env %s", h.appID, h.envName)
	}
	// DevModeConfig.Enabled 为 false 时，不启用 devmode
	if !devMode.Enabled {
		return errors.Errorf("dev mode is not enabled for app %s env %s", h.appID, h.envName)
	}

	// 4. 缓存后续发布阶段需要使用的上下文信息
	h.app = app
	h.env = env
	h.devMode = devMode
	// WorkPath 由 server 端根据应用类型（trpc/taf）返回，如 /data/bkms/dev-mode/trpc 或 /data/bkms/dev-mode/taf
	h.devModeBinPath = osfilepath.Join(devMode.WorkPath, "/bin")
	h.restartScriptPath = osfilepath.Join(devMode.MountPath, "/restart.sh")

	fmt.Printf("App type: %s, WorkPath: %s, MountPath: %s\n", app.Type, devMode.WorkPath, devMode.MountPath)

	return nil
}

// GetAllRunningInstanceIDs 获取所有 Running 状态实例 ID。
func (h *Publisher) GetAllRunningInstanceIDs() ([]string, error) {
	instances, err := instancehandler.ListInstances(
		h.ctx,
		h.cli,
		h.appID,
		h.envName,
		instancehandler.ListInstancesOptions{
			Status: instanceStatusRunning,
		},
	)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"failed to list running app instances for app %s env %s",
			h.appID,
			h.envName,
		)
	}
	if len(instances) == 0 {
		return nil, errors.Errorf("no running instances found for app %s env %s", h.appID, h.envName)
	}

	return lo.Map(instances, func(instance client.Instance, _ int) string {
		return instance.ID
	}), nil
}

// GetSpecifiedInstanceIDs 获取指定的实例 ID，将校验指定实例是否存在。
func (h *Publisher) GetSpecifiedInstanceIDs(instanceIDs []string) ([]string, error) {
	if len(instanceIDs) == 0 {
		return nil, errors.New("no instances specified")
	}

	instances, err := instancehandler.ListInstances(
		h.ctx,
		h.cli,
		h.appID,
		h.envName,
		instancehandler.ListInstancesOptions{},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list app instances for app %s env %s", h.appID, h.envName)
	}

	instanceSet := make(map[string]struct{}, len(instances))
	for _, inst := range instances {
		instanceSet[inst.ID] = struct{}{}
	}

	validatedInstanceIDs := make([]string, 0, len(instanceIDs))
	notFoundInstanceIDs := make([]string, 0)
	for _, instanceID := range instanceIDs {
		if _, ok := instanceSet[instanceID]; !ok {
			notFoundInstanceIDs = append(notFoundInstanceIDs, instanceID)
			continue
		}
		validatedInstanceIDs = append(validatedInstanceIDs, instanceID)
	}

	if len(notFoundInstanceIDs) > 0 {
		return nil, errors.Errorf("instances not found: %v", notFoundInstanceIDs)
	}

	return validatedInstanceIDs, nil
}

// Publish 执行发布：
// 1. 确认指定的文件是否存在，且大小合法
// 2. 初始化 k8s client
// 3. 计算文件 MD5
// 4. 生成随机文件名
// 5. 预先压缩文件为 tar.gz 格式
// 6. 逐个 pod 上传文件并执行 restart.sh
func (h *Publisher) Publish(filePath string, instanceIDs []string, bcsToken string) error {
	if h.env == nil || h.devMode == nil || h.app == nil {
		return errors.New("preCheck must be called before publish")
	}
	if len(instanceIDs) == 0 {
		return errors.New("no instances to publish")
	}

	// 开始前打印本次操作的实例信息
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

	// 2. 初始化 k8s client
	if buildErr := buildKubeClient(h.env.Cluster.ClusterID, bcsToken); buildErr != nil {
		return errors.Wrap(buildErr, "failed to build kubernetes client")
	}

	// 3. 计算文件 MD5
	fileMD5, err := calculateFileMD5(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to calculate file MD5")
	}
	fmt.Printf("File MD5: %s\n", fileMD5)

	// 4. 生成随机文件名（只需生成一次）
	randomName := generateRandomName()
	fmt.Printf("Random filename: %s\n", randomName)

	// 5. 预先压缩文件为 tar.gz 格式（只需压缩一次）
	fmt.Println("Compressing file to tar.gz format...")
	tarGzData, err := compressFileToTarGz(filePath, randomName)
	if err != nil {
		return errors.Wrap(err, "failed to compress file to tar.gz")
	}
	fmt.Printf("Compressed size: %.2f MB\n", float64(len(tarGzData))/(1024*1024))

	namespace := h.env.Cluster.Namespace
	fmt.Println("==================================================")
	fmt.Println("Publish workflow: 1. Upload file ==> 2. Execute restart script ==> 3. Done")
	fmt.Println("==================================================")

	// 6. 逐个 pod 上传文件并执行 restart.sh
	for i, instanceID := range instanceIDs {
		fmt.Printf("\n[%d/%d] Processing instance: %s\n", i+1, len(instanceIDs), instanceID)
		fmt.Printf("  Random filename: %s\n", randomName)

		// 上传已压缩的 tar.gz 数据到 pod
		fmt.Printf("  Uploading file to %s...\n", h.devModeBinPath)
		if err = uploadTarGzToPod(h.ctx, tarGzData, instanceID, namespace, h.devModeBinPath); err != nil {
			return errors.Wrapf(err, "failed to upload file to pod %s", instanceID)
		}
		fmt.Printf("  File upload completed!\n")

		// 执行 restart.sh 脚本
		fmt.Printf("  Executing restart.sh script...\n")
		if err = executeRestartScript(h.ctx, instanceID, namespace, h.restartScriptPath, randomName, fileMD5); err != nil {
			return errors.Wrapf(err, "failed to execute restart script on pod %s", instanceID)
		}
		fmt.Printf("  Instance %s restart completed!\n", instanceID)
	}

	fmt.Println("\n==================================================")
	fmt.Println("All instances published successfully!")
	fmt.Println("==================================================")

	return nil
}

// getEnvByName 通过环境名称获取环境信息
func getEnvByName(ctx context.Context, cli client.Client, workspaceID, envName string) (*client.Env, error) {
	envs, err := cli.ListEnvs(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list envs for workspace %s", workspaceID)
	}

	for i := range envs {
		if envs[i].Name != envName {
			continue
		}
		if envs[i].Cluster == nil {
			return nil, errors.Errorf("env %s cluster not found", envName)
		}
		if envs[i].Cluster.ClusterID == "" {
			return nil, errors.Errorf("env %s cluster id not found", envName)
		}
		if envs[i].Cluster.Namespace == "" {
			return nil, errors.Errorf("env %s namespace not found", envName)
		}
		return &envs[i], nil
	}

	return nil, errors.Errorf("env %s not found in workspace %s", envName, workspaceID)
}

// buildKubeClient 构造 K8s client
// 使用 BCS 配置（--bcs-token）通过 BCS API 网关连接集群。
// bcsToken 优先级：命令行参数 --bcs-token > 配置文件中的 bcs.token。
// 如果通过命令行传入了 token，会自动保存到配置文件中，后续无需再次传入。
func buildKubeClient(clusterID, bcsToken string) error {
	var err error

	// 优先使用命令行传入的 bcsToken
	token := strings.TrimSpace(bcsToken)
	if token != "" {
		// 命令行传入了新 token，保存到配置文件中，后续不需要再传
		config.G.BCS.Token = token
		if err = config.G.Dump(); err != nil {
			return errors.Wrap(err, "failed to save BCS Token to config file")
		}
		fmt.Println("BCS Token saved to config file, no need to pass --bcs-token next time.")
	} else {
		// 没有通过命令行传入，则从配置文件中读取
		token = config.G.BCS.Token
	}

	if token == "" {
		return errors.New(
			"BCS Token is required. Please pass --bcs-token on first use, it will be saved for future use",
		)
	}

	fmt.Printf("Using BCS API to connect cluster %s...\n", clusterID)
	restConfig = &rest.Config{
		BearerToken:     token,
		TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		Host:            fmt.Sprintf("%s/clusters/%s/", bcsAPIHost, clusterID),
	}

	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
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

// uploadTarGzToPod 上传已压缩的 tar.gz 数据到 pod
func uploadTarGzToPod(ctx context.Context, tarGzData []byte, podName, namespace, binPath string) error {
	// 使用 bytes.Reader 来读取压缩数据
	reader := bytes.NewReader(tarGzData)

	// 在容器中执行：mkdir -p <target-path> && tar -xzf - -C <target-path>
	cmd := []string{
		"sh", "-c",
		fmt.Sprintf("mkdir -p %s && tar -xzf - -C %s", binPath, binPath),
	}

	return execInPod(ctx, cmd, reader, podName, namespace, defaultContainerName)
}

// executeRestartScript 在 pod 中执行 restart.sh 脚本
func executeRestartScript(ctx context.Context, podName, namespace, scriptPath, randomName, md5sum string) error {
	// 执行 restart.sh 脚本，传入随机名称和 md5 值
	cmd := []string{
		"bash", scriptPath, randomName, md5sum,
	}

	return execInPod(ctx, cmd, nil, podName, namespace, defaultContainerName)
}

// execInPod 使用 SPDY 在 Pod 内执行指定命令
func execInPod(ctx context.Context, cmd []string, stdin io.Reader, podName, namespace, container string) error {
	req := clientset.CoreV1().RESTClient().Post().
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

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
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
