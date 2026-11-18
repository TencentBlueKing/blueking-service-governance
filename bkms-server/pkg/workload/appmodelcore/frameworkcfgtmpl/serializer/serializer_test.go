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
	"strings"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/frameworkcfgtmpl/serializer"
)

var _ = Describe("framework config template serializers", func() {
	DescribeTable("validates file names with the app config file name rule",
		func(fileName string, valid bool) {
			create := serializer.CreateTemplateInput{
				Framework: "trpc", Language: "go", Name: "t", FileName: fileName, Content: "a: 1",
			}
			update := serializer.UpdateTemplateInput{Name: "t", FileName: fileName, Content: "a: 1"}
			for _, input := range []any{&create, &update} {
				err := binding.Validator.ValidateStruct(input)
				if valid {
					Expect(err).NotTo(HaveOccurred())
				} else {
					Expect(err).To(MatchError(ContainSubstring("FileName")))
				}
			}
		},
		Entry("default tRPC file name", "trpc_go.yaml", true),
		Entry("default TAF file name", "config.conf", true),
		Entry("empty", "", false),
		Entry("path separator", "conf/a.yaml", false),
		Entry("whitespace", " a.yaml", false),
		Entry("too long", strings.Repeat("a", 65), false),
	)
})
