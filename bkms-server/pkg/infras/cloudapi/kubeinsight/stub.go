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

package kubeinsight

import (
	"context"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// stubClusterReport 本地开发时返回的固定集群巡检报告
var stubClusterReport = &ClusterReport{
	ClusterID: "XXX-K8S-00001",
	ClusterInfo: &ClusterInfo{
		ClusterID:      "XXX-K8S-00001",
		ClusterName:    "stub-cluster-default",
		ClusterType:    "single",
		Provider:       "tke",
		ManageType:     "MANAGED_CLUSTER",
		Creator:        "stub-user",
		BusinessID:     "100001",
		ProjectID:      "stub-project-a",
		ClusterVersion: "1.28.3",
		NetworkParams: &NetworkParams{
			Cidrs:         []string{"127.0.0.0/16"},
			MaxNodePodNum: 256,
			MaxServiceNum: 4096,
		},
		OSRuntimeInfo: &OSRuntimeInfo{
			OSImage:        "TencentOS Server 3.1",
			Runtime:        "containerd",
			RuntimeVersion: "1.6.28",
		},
		Region:      "ap-guangzhou",
		VpcID:       "vpc-stub001",
		NodeCount:   5,
		MasterCount: 3,
	},
	AbnormalItems: []CheckItem{
		{
			Timestamp:           "2026-06-11T10:00:00Z",
			LastUpdateTimestamp: "2026-06-11T10:30:00Z",
			Key:                 "timecheck_drift",
			Description:         "节点产生时间漂移",
			Category:            "节点异常",
			ContextMsg:          "node-01 时间偏差超过 5s",
			Level:               "WARN",
			ResourceKey:         "127.0.0.1",
			ResourceType:        "node",
			ErrorDetail:         "NTP 同步失败，时间偏差 8.2s",
			Solutions:           "检查 NTP 服务状态，重启 chronyd",
			Recovered:           false,
			RecordCount:         3,
		},
	},
	Score:     85,
	StartTime: "2026-06-11T10:00:00Z",
	EndTime:   "2026-06-11T10:05:00Z",
}

// StubApiClient 测试用的 KubeInsight API 客户端实现，返回模拟数据
type StubApiClient struct{}

// NewStub 创建 StubApiClient
func NewStub() *StubApiClient {
	return &StubApiClient{}
}

// GetLatestClusterReport 模拟获取集群最新检查报告
func (s *StubApiClient) GetLatestClusterReport(
	ctx context.Context,
	clusterID string,
	generatePDF bool,
) (*ClusterReport, []byte, error) {
	log.Infof(ctx, "Stub: GetLatestClusterReport request: clusterID=%s, generatePDF=%v", clusterID, generatePDF)

	report := *stubClusterReport
	report.ClusterID = clusterID
	report.ClusterInfo.ClusterID = clusterID

	var pdfData []byte
	if generatePDF {
		pdfData = []byte("%PDF-1.4 stub pdf content")
	}

	return &report, pdfData, nil
}
