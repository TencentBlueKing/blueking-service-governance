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

package serializer

// --- KubeInsight URI 参数 ---

// KubeInsightReportQueryInput 获取巡检报告的查询参数
type KubeInsightReportQueryInput struct {
	EnvID       string `form:"envID" binding:"required,min=1"`
	GeneratePDF bool   `form:"generatePDF"`
}

// --- KubeInsight Output ---

// ClusterReportOutput 集群巡检报告输出
type ClusterReportOutput struct {
	ClusterID     string             `json:"clusterID"`
	StartTime     string             `json:"startTime"`
	EndTime       string             `json:"endTime"`
	ClusterInfo   *ClusterInfoOutput `json:"clusterInfo"`
	AbnormalItems []*CheckItemOutput `json:"abnormalItems"`
	Score         int32              `json:"score"`
	PdfData       []byte             `json:"pdfData,omitempty"`
}

// ClusterInfoOutput 集群信息输出
type ClusterInfoOutput struct {
	ClusterID      string               `json:"clusterID"`
	ClusterName    string               `json:"clusterName"`
	ClusterType    string               `json:"clusterType"`
	Provider       string               `json:"provider"`
	ManageType     string               `json:"manageType"`
	Creator        string               `json:"creator"`
	BusinessID     string               `json:"businessID"`
	ProjectID      string               `json:"projectID"`
	ClusterVersion string               `json:"clusterVersion"`
	NetworkParams  *NetworkParamsOutput `json:"networkParams"`
	OSRuntimeInfo  *OSRuntimeInfoOutput `json:"osRuntimeInfo"`
	Region         string               `json:"region"`
	VpcID          string               `json:"vpcID"`
	NodeCount      int32                `json:"nodeCount"`
	MasterCount    int32                `json:"masterCount"`
}

// NetworkParamsOutput 网络参数输出
type NetworkParamsOutput struct {
	Cidrs         []string `json:"cidrs"`
	MaxNodePodNum int32    `json:"maxNodePodNum"`
	MaxServiceNum int32    `json:"maxServiceNum"`
}

// OSRuntimeInfoOutput OS/镜像/运行时信息输出
type OSRuntimeInfoOutput struct {
	OSImage        string `json:"osImage"`
	Runtime        string `json:"runtime"`
	RuntimeVersion string `json:"runtimeVersion"`
}

// CheckItemOutput 检查项输出
type CheckItemOutput struct {
	Timestamp           string `json:"timestamp"`
	LastUpdateTimestamp string `json:"lastUpdateTimestamp"`
	Key                 string `json:"key"`
	Description         string `json:"description"`
	Category            string `json:"category"`
	ContextMsg          string `json:"contextMsg"`
	Level               string `json:"level"`
	ResourceKey         string `json:"resourceKey"`
	ResourceType        string `json:"resourceType"`
	ErrorDetail         string `json:"errorDetail"`
	Solutions           string `json:"solutions"`
	Recovered           bool   `json:"recovered"`
	RecordCount         int32  `json:"recordCount"`
}

// GetLatestEnvReportOutput 获取最新环境巡检报告的响应
type GetLatestEnvReportOutput struct {
	Data *ClusterReportOutput `json:"data"`
}
