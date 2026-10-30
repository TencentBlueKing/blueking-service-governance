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

var _ = Describe("buildWeTermURL", func() {
	It("builds url in the order agreed with BCS", func() {
		url := buildWeTermURL(weTermParams{
			EnvID:         "env-id",
			ProjectID:     "project-id",
			ProjectCode:   "project-code",
			ClusterID:     "BCS-K8S-00000",
			Namespace:     "ns",
			PodName:       "pod-0",
			ContainerName: "main",
		})
		Expect(url).To(Equal(
			"weterm://session/open/bcs?ns=ns&pod=pod-0&container=main&type=login" +
				"&clusterId=BCS-K8S-00000&envId=env-id&projectID=project-id&projectCode=project-code",
		))
	})

	It("escapes query values", func() {
		url := buildWeTermURL(weTermParams{PodName: "a&b=c"})
		Expect(url).To(ContainSubstring("pod=a%26b%3Dc&"))
	})
})
