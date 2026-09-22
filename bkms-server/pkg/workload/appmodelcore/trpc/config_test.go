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

package trpc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/trpc"
)

var _ = Describe("DecodeConfig", func() {
	It("decodes legacy file metadata and rejects unknown keys or empty objects", func() {
		legacy := appmodel.TrpcConfig{
			FileName:    "trpc_go.yaml",
			FilePath:    "/etc/",
			Language:    "go",
			FileContent: "secret",
		}
		cfg := trpc.ConfigMapFromLegacy(legacy)
		Expect(cfg).To(Equal(map[string]any{"fileName": "trpc_go.yaml", "filePath": "/etc/"}))

		got, err := trpc.DecodeConfig(cfg, appruntime.FrameworkConfigVersionV1)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(trpc.Config{FileName: "trpc_go.yaml", FilePath: "/etc/"}))

		_, err = trpc.DecodeConfig(map[string]any{
			"fileName": "trpc_go.yaml",
			"filePath": "/etc/",
			"language": "go",
		}, appruntime.FrameworkConfigVersionV1)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid keys: language"))

		_, err = trpc.DecodeConfig(map[string]any{}, appruntime.FrameworkConfigVersionV1)
		Expect(err).To(HaveOccurred())
	})
})
