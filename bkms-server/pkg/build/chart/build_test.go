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

package build

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/revision"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

var _ = Describe("ChartBuildService revision check", func() {
	It("rejects empty revision before touching any store", func() {
		// 零值服务的所有 store 都是 nil：只要校验晚于准备或版本号生成就会 panic，
		// 因此能返回哨兵错误即证明语义化版本号没有被消耗
		svc := &ChartBuildService{}
		app := &bkmsapp.Application{ID: "helm-app", WorkspaceID: "ws-1", Type: bkmsapp.AppTypeHelm}

		_, err := svc.ExecuteChartBuild(
			context.Background(), app, "patch", revision.CodeRevision{Type: revision.TypeTag},
		)

		Expect(errors.Is(err, ErrRevisionRequired)).To(BeTrue())
	})
})
