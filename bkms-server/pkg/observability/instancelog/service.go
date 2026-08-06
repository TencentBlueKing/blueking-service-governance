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

// Package instancelog 提供实例日志查询的业务编排逻辑
package instancelog

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

const (
	// DownloadLogsMaxTailLines 下载日志最多返回 10 万行
	DownloadLogsMaxTailLines int64 = 100 * 1000
	// DownloadLogsMaxLimitBytes 下载日志最大体积为 30MB
	DownloadLogsMaxLimitBytes int64 = 30 * 1024 * 1024
)

// ErrInstanceNotFound 表示目标实例不属于当前应用部署，或实例不存在
var ErrInstanceNotFound = errors.New("app instance not found")

// LogManager 实例日志管理器，封装日志查询和下载的业务编排逻辑
type LogManager struct {
	namespace     string
	containerName string
	podClient     *k8sclient.PodClient
}

// DownloadResult 日志下载结果
type DownloadResult struct {
	Reader   io.ReadCloser
	Filename string
}

// LogEntry 是单行实例日志。
type LogEntry struct {
	// 日志时间戳
	Timestamp string
	// 日志内容
	Content string
}

// NewLogManager 创建日志管理器实例
// 包括：获取集群环境信息、创建 PodClient、确定容器名称
func NewLogManager(
	ctx context.Context,
	deployRecordStore appmodeldeploy.RecordStore,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	trafficLaneName string,
	instanceID string,
) (*LogManager, error) {
	podClient := k8sclient.NewPodClient(cluster.NewConfig(env.Cluster.ClusterID))
	namespace := env.Cluster.Namespace
	containerName := defaults.WorkloadMainContainerName
	if bkmsapp.IsAppModelType(app.Type) {
		record, err := deployRecordStore.GetLatest(ctx, app.ID, env.Name, trafficLaneName)
		if err != nil {
			return nil, errors.Wrapf(err, "get latest deploy record for app %s", app.ID)
		}
		if len(record.LabelSelector) == 0 {
			return nil, errors.New("deploy record label selector is empty")
		}
		labelSelector := labels.SelectorFromSet(record.LabelSelector).String()
		pods, err := podClient.List(ctx, record.Namespace, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return nil, errors.Wrapf(
				err, "list namespace %s labelSelector [%s] pods", record.Namespace, labelSelector,
			)
		}
		found := false
		for _, pod := range pods.Items {
			if pod.GetName() == instanceID {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.Wrapf(ErrInstanceNotFound, "instance %s does not belong to app %s", instanceID, app.ID)
		}
		namespace = record.Namespace
	} else {
		// 非 AppModel 类应用主容器名称不一定是 main
		var err error
		containerName, err = podClient.GetFirstContainerName(ctx, env.Cluster.Namespace, instanceID)
		if err != nil {
			return nil, errors.Wrap(err, "get first container name")
		}
	}

	return &LogManager{
		namespace:     namespace,
		containerName: containerName,
		podClient:     podClient,
	}, nil
}

// ListLogs 查询实例日志
func (m *LogManager) ListLogs(
	ctx context.Context,
	instanceID string,
	previous bool,
	tailLines int64,
) ([]*LogEntry, error) {
	if previous {
		hasPreviousLogs, err := m.hasPreviousContainerLogs(ctx, instanceID)
		if err != nil {
			return nil, errors.Wrap(err, "check previous container logs")
		}
		// 无最近一次重启日志，直接返回空数据
		if !hasPreviousLogs {
			return nil, nil
		}
	}

	logs, err := m.podClient.ListLogs(
		ctx,
		m.namespace,
		instanceID,
		&corev1.PodLogOptions{
			Container: m.containerName,
			Previous:  previous,
			TailLines: &tailLines,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "list logs from pod")
	}

	// 组装返回格式
	logEntries := lo.Map(logs, func(log k8sclient.LogEntry, _ int) *LogEntry {
		return &LogEntry{Timestamp: log.Timestamp, Content: log.Content}
	})

	return logEntries, nil
}

// PrepareDownload 准备实例日志下载
func (m *LogManager) PrepareDownload(ctx context.Context, instanceID string, previous bool) (*DownloadResult, error) {
	filename := buildInstanceLogDownloadFilename(instanceID, m.containerName)

	if previous {
		hasPreviousLogs, err := m.hasPreviousContainerLogs(ctx, instanceID)
		if err != nil {
			return nil, errors.Wrap(err, "check previous container logs")
		}
		// 无最近一次重启日志，直接返回空文件流
		if !hasPreviousLogs {
			return &DownloadResult{Reader: io.NopCloser(strings.NewReader("")), Filename: filename}, nil
		}
	}

	tailLines := DownloadLogsMaxTailLines
	limitBytes := DownloadLogsMaxLimitBytes
	reader, err := m.podClient.OpenLogsStream(
		ctx,
		m.namespace,
		instanceID,
		&corev1.PodLogOptions{
			Container:  m.containerName,
			Previous:   previous,
			TailLines:  &tailLines,
			LimitBytes: &limitBytes,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "prepare download logs from pod")
	}

	return &DownloadResult{Reader: reader, Filename: filename}, nil
}

// hasPreviousContainerLogs 检查是否有最近一次重启日志
func (m *LogManager) hasPreviousContainerLogs(ctx context.Context, instanceID string) (bool, error) {
	pod, err := m.podClient.Get(ctx, m.namespace, instanceID, metav1.GetOptions{})
	if err != nil {
		return false, errors.Wrap(err, "get pod")
	}

	for _, status := range mapx.GetList(pod.Object, "status.containerStatuses") {
		statusMap, ok := status.(map[string]any)
		if !ok {
			continue
		}
		if mapx.GetStr(statusMap, "name") != m.containerName {
			continue
		}
		return mapx.GetInt64(statusMap, "restartCount") > 0, nil
	}

	return false, nil
}

var invalidFilenamePartCharRE = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// sanitizeAttachmentFilenamePart 清理附件文件名的各个部分， 只保留字母、数字和 . - _ 字符，其余替换为下划线
func sanitizeAttachmentFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return invalidFilenamePartCharRE.ReplaceAllString(value, "_")
}

// buildInstanceLogDownloadFilename 构建实例日志下载的文件名
func buildInstanceLogDownloadFilename(instanceID, containerName string) string {
	return fmt.Sprintf(
		"%s-%s-%s.log",
		sanitizeAttachmentFilenamePart(instanceID),
		sanitizeAttachmentFilenamePart(containerName),
		time.Now().Format("20060102150405"),
	)
}
