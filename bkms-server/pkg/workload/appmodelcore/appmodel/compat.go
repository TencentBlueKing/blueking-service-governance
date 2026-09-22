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

package appmodel

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/datatype"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

// FillSnapshot copies workload compatibility fields into a runtime snapshot.
func (w Workload) FillSnapshot(s *appruntime.Snapshot) {
	if s == nil {
		return
	}
	s.WorkloadType = w.Type
	s.TrpcConfigLanguage = w.TrpcConfig.Language
	s.TrpcFileName = w.TrpcConfig.FileName
	s.TrpcFilePath = w.TrpcConfig.FilePath
	s.HasTrpcFileContent = w.TrpcConfig.FileContent != ""
	s.TafFileName = w.TafConfig.FileName
	s.TafFilePath = w.TafConfig.FilePath
	s.HasTafFileContent = w.TafConfig.FileContent != ""
	s.HasTrpcConfig = w.TrpcConfig.FileName != "" ||
		w.TrpcConfig.FilePath != "" ||
		w.TrpcConfig.Language != "" ||
		w.TrpcConfig.FileContent != ""
	s.HasTafConfig = w.TafConfig.FileName != "" ||
		w.TafConfig.FilePath != "" ||
		w.TafConfig.FileContent != ""
	s.FrameworkConfig = datatype.CloneMap(w.FrameworkConfig)
	s.FrameworkConfigVersion = w.FrameworkConfigVersion
}

// RuntimeSnapshot merges an application-level snapshot with this workload.
func (w Workload) RuntimeSnapshot(base appruntime.Snapshot) appruntime.Snapshot {
	w.FillSnapshot(&base)
	return base
}
