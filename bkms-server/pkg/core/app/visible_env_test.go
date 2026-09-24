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

package app_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("Application.CheckVisibleEnv", func() {
	stdEnv := func(name string) *envmodel.Environment {
		return &envmodel.Environment{Name: name, Kind: envmodel.EnvironmentKindStandard}
	}
	featEnv := func(name, ownerAppID string) *envmodel.Environment {
		return &envmodel.Environment{Name: name, Kind: envmodel.EnvironmentKindFeature, OwnerAppID: ownerAppID}
	}

	DescribeTable("allow deploying to the target env",
		func(visibleEnvNames []string, env *envmodel.Environment) {
			app := &bkmsapp.Application{ID: "app-a", VisibleEnvNames: visibleEnvNames}
			Expect(app.CheckVisibleEnv(env)).To(Succeed())
		},
		Entry("unconfigured app may deploy to any standard env", nil, stdEnv("prod")),
		Entry("cleared list may deploy to any standard env", []string{}, stdEnv("prod")),
		Entry("configured app may deploy to a listed standard env", []string{"test", "stag"}, stdEnv("test")),
		Entry("configured app may deploy to its own feature env", []string{"test"}, featEnv("feat-a", "app-a")),
		// 历史环境记录可能缺失 kind，GetKind 回落到 standard，仍按名单判断
		Entry("legacy env without kind is treated as standard",
			[]string{"test"}, &envmodel.Environment{Name: "test"}),
	)

	DescribeTable("reject deploying to the target env",
		func(visibleEnvNames []string, env *envmodel.Environment) {
			app := &bkmsapp.Application{ID: "app-a", VisibleEnvNames: visibleEnvNames}

			err := app.CheckVisibleEnv(env)

			var bkErr *bkerrs.Error
			Expect(errors.As(err, &bkErr)).To(BeTrue())
			Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(err.Error()).To(ContainSubstring(env.Name))
			Expect(err.Error()).To(ContainSubstring("visible environment"))
		},
		Entry("configured app may not deploy to an unlisted standard env", []string{"test"}, stdEnv("prod")),
		Entry("configured app may not deploy to another app's feature env",
			[]string{"test"}, featEnv("feat-b", "app-b")),
	)
})
