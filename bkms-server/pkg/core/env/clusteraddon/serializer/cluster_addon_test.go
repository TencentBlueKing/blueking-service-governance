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

package serializer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon/serializer"
)

var _ = Describe("ClusterAddonInfoOutput", func() {
	enCtx := i18n.WithLang(context.Background(), i18n.LangEN)

	It("should localize the description of builtin addons", func() {
		out := new(serializer.ClusterAddonInfoOutput).FromModel(enCtx, clusteraddon.ClusterAddonInfo{
			Name:        "agones",
			Description: "Agones 是 Kubernetes 上的游戏服务器管理器，用于管理游戏服务器实例的部署、伸缩和调度。",
		})
		Expect(out.Description).To(HavePrefix("Agones is a Kubernetes-based game server manager"))
	})

	It("should fall back to the defined description for addons without messages", func() {
		out := new(serializer.ClusterAddonInfoOutput).FromModel(enCtx, clusteraddon.ClusterAddonInfo{
			Name:        "custom-addon",
			Description: "部署侧自定义的插件描述",
		})
		Expect(out.Description).To(Equal("部署侧自定义的插件描述"))
	})
})
