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

// Package portforward 提供应用实例 Pod 端口转发服务。
package portforward

import (
	"context"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

// PodClient 查询 Kubernetes Pod 资源。
type PodClient interface {
	Get(ctx context.Context, namespace, name string, opts metav1.GetOptions) (*unstructured.Unstructured, error)
	List(ctx context.Context, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error)
}

// PodClientFactory 基于集群 ID 创建 PodClient。
type PodClientFactory func(clusterID string) PodClient

// StreamOpener 打开目标 Pod 端口转发字节流。
type StreamOpener func(ctx context.Context, target Target) (io.ReadWriteCloser, error)

// Target 端口转发目标实例信息。
type Target struct {
	ClusterID  string
	Namespace  string
	PodName    string
	PodIP      string
	RemotePort int32
}

var defaultTargetDialer = net.Dialer{
	Timeout:   5 * time.Second,
	KeepAlive: 30 * time.Second,
}

// DefaultStreamOpener 使用普通 TCP 直连目标 Pod IP 和端口。
func DefaultStreamOpener(ctx context.Context, target Target) (io.ReadWriteCloser, error) {
	address := net.JoinHostPort(target.PodIP, cast.ToString(target.RemotePort))
	return defaultTargetDialer.DialContext(ctx, "tcp", address)
}

// Service 应用实例端口转发服务。
type Service struct {
	deployRecordStore appmodeldeploy.RecordStore
	podClientFactory  PodClientFactory
	streamOpener      StreamOpener
}

// Option Service 选项。
type Option func(*Service)

// WithPodClientFactory 指定 PodClientFactory，主要用于测试。
func WithPodClientFactory(factory PodClientFactory) Option {
	return func(s *Service) {
		s.podClientFactory = factory
	}
}

// WithStreamOpener 指定 StreamOpener，主要用于测试。
func WithStreamOpener(opener StreamOpener) Option {
	return func(s *Service) {
		s.streamOpener = opener
	}
}

// NewService 创建端口转发服务。
func NewService(deployRecordStore appmodeldeploy.RecordStore, opts ...Option) *Service {
	s := &Service{
		deployRecordStore: deployRecordStore,
		podClientFactory: func(clusterID string) PodClient {
			return k8sclient.NewPodClient(cluster.NewConfig(clusterID))
		},
		streamOpener: DefaultStreamOpener,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetDeployedPod 获取当前已部署的指定实例 Pod，封装“获取最新部署记录 → 查询 Pod → 校验 labels”的公共逻辑。
func (s *Service) GetDeployedPod(
	ctx context.Context, appID, envName, instanceID string,
) (*unstructured.Unstructured, *appmodeldeploy.Record, error) {
	record, err := s.deployRecordStore.GetLatest(ctx, appID, envName, "")
	if err != nil {
		return nil, nil, errors.Wrap(err, "deploy record not found")
	}

	pod, err := s.podClientFactory(record.ClusterID).Get(ctx, record.Namespace, instanceID, metav1.GetOptions{})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get pod '%s' in namespace '%s'", instanceID, record.Namespace)
	}

	// 校验 Pod labels 是否匹配部署记录的 LabelSelector，确保实例属于当前应用环境。
	if !matchesLabelSelector(pod, record.LabelSelector) {
		return nil, nil, errors.New("pod does not belong to current application environment")
	}

	return pod, record, nil
}

// ResolveTarget 根据应用 ID、环境名称、实例 ID 和远程端口，解析出用于端口转发的目标实例信息。
// 校验实例是否属于当前应用环境，提取可用的 Pod IP，并校验端口号合法性。
func (s *Service) ResolveTarget(
	ctx context.Context,
	appID, envName, instanceID string,
	remotePort int32,
) (*Target, error) {
	if remotePort < 1 || remotePort > 65535 {
		return nil, errors.New("remote port must be between 1 and 65535")
	}

	log.InfoAttrs(ctx, "resolving port-forward target",
		slog.String("app_id", appID),
		slog.String("env_name", envName),
		slog.String("instance_id", instanceID),
		slog.Int("remote_port", int(remotePort)),
	)

	pod, record, err := s.GetDeployedPod(ctx, appID, envName, instanceID)
	if err != nil {
		return nil, err
	}

	podIP, err := k8sclient.ResolvePodIPFromManifest(pod.Object)
	if err != nil {
		return nil, err
	}

	return &Target{
		ClusterID:  record.ClusterID,
		Namespace:  record.Namespace,
		PodName:    instanceID,
		PodIP:      podIP,
		RemotePort: remotePort,
	}, nil
}

// OpenTargetStream 解析目标实例并打开 TCP 连接，供 handler 层在 WebSocket 升级前调用。
func (s *Service) OpenTargetStream(
	ctx context.Context,
	appID, envName, instanceID string,
	remotePort int32,
) (io.ReadWriteCloser, error) {
	target, err := s.ResolveTarget(ctx, appID, envName, instanceID, remotePort)
	if err != nil {
		return nil, err
	}

	log.InfoAttrs(ctx, "opening target pod connection",
		slog.String("app_id", appID),
		slog.String("env_name", envName),
		slog.String("instance_id", instanceID),
		slog.Int("remote_port", int(remotePort)),
	)

	podStream, err := s.streamOpener(ctx, *target)
	if err != nil {
		log.WarnAttrs(ctx, "open target pod connection failed",
			slog.String("app_id", appID),
			slog.String("env_name", envName),
			slog.String("instance_id", instanceID),
			slog.Int("remote_port", int(remotePort)),
			slog.String("reason", classifyTargetOpenError(err)),
		)
		return nil, errors.New("open target pod connection failed")
	}
	return podStream, nil
}

// matchesLabelSelector 校验 Pod 的 labels 是否包含部署记录的所有 LabelSelector 键值对。
func matchesLabelSelector(pod *unstructured.Unstructured, selector map[string]string) bool {
	podLabels := pod.GetLabels()
	for key, value := range selector {
		if podLabels[key] != value {
			return false
		}
	}
	return true
}
