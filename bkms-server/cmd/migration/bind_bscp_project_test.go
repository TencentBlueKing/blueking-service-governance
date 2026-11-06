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

package migration

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
)

var _ = Describe("runBindBscpProject", func() {
	var originalCfg *config.Config

	BeforeEach(func() {
		originalCfg = config.G
	})

	AfterEach(func() {
		config.G = originalCfg
	})

	It("should fail fast without touching workspace when bscp is disabled", func() {
		raw, err := os.ReadFile("../../tests/configs/config.yaml")
		Expect(err).NotTo(HaveOccurred())
		content := strings.Replace(string(raw), "disableBSCP: false", "disableBSCP: true", 1)
		Expect(content).To(ContainSubstring("disableBSCP: true"))

		cfgFile := filepath.Join(GinkgoT().TempDir(), "config.yaml")
		Expect(os.WriteFile(cfgFile, []byte(content), 0o600)).To(Succeed())

		// 开关检查需早于交互式读取 token 与数据库访问，否则测试会阻塞或因无数据库而失败
		err = runBindBscpProject(context.Background(), cfgFile, "ws-1", "", "admin", true)
		Expect(err).To(MatchError(bscpapi.ErrDisabled))
	})
})
