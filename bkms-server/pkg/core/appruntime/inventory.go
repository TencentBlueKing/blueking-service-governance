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

package appruntime

// RecordSample is a desensitized scan row. It never includes file content.
type RecordSample struct {
	AppID              string   `json:"appID"`
	PersistedType      string   `json:"persistedType"`
	TargetType         string   `json:"targetType"`
	Framework          string   `json:"framework"`
	Language           string   `json:"language"`
	IssueCodes         []string `json:"issueCodes"`
	HasTrpcFileContent bool     `json:"hasTrpcFileContent"`
	HasTafFileContent  bool     `json:"hasTafFileContent"`
}

// InventorySummary aggregates inspect reports for the dry-run scan command.
type InventorySummary struct {
	Total           int            `json:"total"`
	ByPersistedType map[string]int `json:"byPersistedType"`
	ByIssue         map[string]int `json:"byIssue"`
	Blocking        int            `json:"blocking"`
	Samples         []RecordSample `json:"samples"`
}

// Summarize builds a desensitized inventory from inspect reports.
func Summarize(reports []Report, snapshots []Snapshot, sampleLimit int) InventorySummary {
	summary := InventorySummary{
		Total:           len(reports),
		ByPersistedType: map[string]int{},
		ByIssue:         map[string]int{},
	}
	if sampleLimit <= 0 {
		sampleLimit = 20
	}
	byApp := map[string]Snapshot{}
	for _, snap := range snapshots {
		byApp[snap.AppID] = snap
	}
	for _, report := range reports {
		summary.ByPersistedType[report.PersistedType]++
		if report.Blocking() {
			summary.Blocking++
		}
		codes := make([]string, 0, len(report.Issues))
		for _, issue := range report.Issues {
			summary.ByIssue[issue.Code]++
			codes = append(codes, issue.Code)
		}
		if len(codes) == 0 {
			continue
		}
		if len(summary.Samples) >= sampleLimit {
			continue
		}
		snap := byApp[report.AppID]
		summary.Samples = append(summary.Samples, RecordSample{
			AppID:              report.AppID,
			PersistedType:      report.PersistedType,
			TargetType:         report.Stack.Type,
			Framework:          report.Stack.Framework,
			Language:           report.Stack.Language,
			IssueCodes:         codes,
			HasTrpcFileContent: snap.HasTrpcFileContent,
			HasTafFileContent:  snap.HasTafFileContent,
		})
	}
	return summary
}
