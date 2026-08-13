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

package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("buildExportFilename", func() {
	It("keeps original parts without replacing characters", func() {
		filename := buildExportFilename("workspace-a", "app", "foo/bar", "with space", "effective-env-vars.env")
		Expect(filename).To(Equal("workspace-a-app-foo/bar-with space-effective-env-vars.env"))
	})

	It("prefixes workspace ID for business exports", func() {
		filename := buildExportFilename("workspace-a", "app", "foo/bar", "effective-env-vars.env")
		Expect(filename).To(Equal("workspace-a-app-foo/bar-effective-env-vars.env"))
	})

	It("skips empty workspace ID after trimming", func() {
		filename := buildExportFilename("  ", "env", "test", "scoped-env-vars.env")
		Expect(filename).To(Equal("env-test-scoped-env-vars.env"))
	})
})
