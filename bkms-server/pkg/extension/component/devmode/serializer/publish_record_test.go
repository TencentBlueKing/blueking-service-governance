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
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode/serializer"
)

var _ = Describe("ListPublishRecordsByScopeInput", func() {
	validate := func(envNames, envTypes []string) error {
		input := slz.ListPublishRecordsByScopeInput{
			EnvNames: envNames, EnvTypes: envTypes, Page: 1, PageSize: 10,
		}
		if err := binding.Validator.ValidateStruct(input); err != nil {
			return err
		}
		return input.Validate()
	}

	It("binds repeated query values", func() {
		req := httptest.NewRequest(
			http.MethodGet,
			"/?envNames=stag&envNames=test&statuses=pending&statuses=running&page=1&pageSize=10", nil,
		)
		var input slz.ListPublishRecordsByScopeInput
		Expect(binding.Query.Bind(req, &input)).To(Succeed())
		Expect(input.EnvNames).To(Equal([]string{"stag", "test"}))
		Expect(input.Statuses).To(Equal([]string{"pending", "running"}))
	})

	It("accepts env names", func() {
		Expect(validate([]string{"stag", "test"}, nil)).To(Succeed())
	})

	It("accepts supported env types", func() {
		Expect(validate(nil, []string{"development", "test", "staging"})).To(Succeed())
	})

	DescribeTable("rejects invalid input",
		func(envNames, envTypes []string, expectedDetail, msg string) {
			err := validate(envNames, envTypes)
			Expect(err).To(HaveOccurred())
			if expectedDetail != "" {
				bkErr := bkerrs.FromError(err)
				Expect(bkErr).NotTo(BeNil())
				Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
				Expect(bkErr.DetailCode()).To(Equal(bkerrs.ErrDetailCode(expectedDetail)))
				return
			}
			Expect(err.Error()).To(ContainSubstring(msg))
		},
		Entry("no scope", nil, nil, "EXTENSION_DEVMODE_RECORD_QUERY_SCOPE_CONFLICT", ""),
		Entry("both scopes", []string{"stag"}, []string{"test"}, "EXTENSION_DEVMODE_RECORD_QUERY_SCOPE_CONFLICT", ""),
		Entry("production", nil, []string{"production"}, "", "'oneof' tag"),
		Entry("unknown type", nil, []string{"dev"}, "", "'oneof' tag"),
		Entry("invalid env name", []string{"stag/1"}, nil, "", "'uri_slug' tag"),
	)
})
