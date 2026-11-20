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

package audit_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

const mask = audit.SensitiveValueMask

// recorded 经 audit.WithDataAfter 写入操作记录，再反序列化为 T，用于断言实际落库的内容
func recorded[T any](data any) T {
	r := &audit.OperationRecord{}
	audit.WithDataAfter(data)(r)
	var out T
	Expect(json.Unmarshal(r.Data.After, &out)).To(Succeed())
	return out
}

var _ = Describe("Sanitize models", func() {
	Describe("env vars", func() {
		It("masks sensitive app env var for both value and pointer", func() {
			v := &appmodel.Variable{Key: "K", Value: "plain", IsSensitive: true}
			Expect(recorded[appmodel.Variable](v).Value).To(Equal(mask))
			Expect(recorded[appmodel.Variable](*v).Value).To(Equal(mask))
			Expect(v.Value).To(Equal("plain"))

			normal := appmodel.Variable{Key: "K", Value: "plain"}
			Expect(recorded[appmodel.Variable](normal).Value).To(Equal("plain"))
		})

		It("masks sensitive scoped env var", func() {
			v := &envvars.ScopedEnvVar{Key: "K", Value: "plain", IsSensitive: true}
			masked := recorded[envvars.ScopedEnvVar](v)
			Expect(masked.Key).To(Equal("K"))
			Expect(masked.Value).To(Equal(mask))
			Expect(v.Value).To(Equal("plain"))

			normal := &envvars.ScopedEnvVar{Key: "K", Value: "plain"}
			Expect(recorded[envvars.ScopedEnvVar](normal).Value).To(Equal("plain"))
		})

		It("keeps nil pointer as null", func() {
			var m *appmodel.AppModel
			r := &audit.OperationRecord{}
			audit.WithDataBefore(m)(r)
			Expect(string(r.Data.Before)).To(Equal("null"))
		})
	})

	Describe("app model", func() {
		It("masks env vars without modifying the original", func() {
			comp := &component.Component{Name: "polaris"}
			comp.Type = "ImportPolaris"
			comp.Properties = map[string]any{"polarisToken": "tk", "polarisName": "n"}
			m := &appmodel.AppModel{
				AppID: "app",
				Workload: appmodel.Workload{EnvVars: []appmodel.Variable{
					{Key: "SECRET", Value: "s", IsSensitive: true},
					{Key: "NORMAL", Value: "n"},
				}},
				Components: []*component.Component{comp, nil},
			}

			masked := recorded[appmodel.AppModel](m)

			Expect(masked.AppID).To(Equal("app"))
			Expect(masked.Workload.EnvVars).To(HaveLen(2))
			Expect(masked.Workload.EnvVars[0].Value).To(Equal(mask))
			Expect(masked.Workload.EnvVars[1].Value).To(Equal("n"))
			Expect(masked.Components).To(HaveLen(2))
			Expect(masked.Components[0].Name).To(Equal("polaris"))
			// 组件属性不脱敏
			Expect(masked.Components[0].Properties["polarisToken"]).To(Equal("tk"))
			Expect(masked.Components[1]).To(BeNil())

			Expect(m.Workload.EnvVars[0].Value).To(Equal("s"))
			Expect(comp.Properties["polarisToken"]).To(Equal("tk"))
		})
	})

	Describe("secret fields", func() {
		It("masks bscp token in workspace", func() {
			ws := &workspace.Workspace{ID: "ws"}
			ws.BkSystems.BscpToken = "tk"
			ws.BkSystems.BscpCredentialID = "1"

			masked := recorded[workspace.Workspace](ws)
			Expect(masked.ID).To(Equal("ws"))
			Expect(masked.BkSystems.BscpToken).To(Equal(mask))
			Expect(masked.BkSystems.BscpCredentialID).To(Equal("1"))
			Expect(ws.BkSystems.BscpToken).To(Equal("tk"))
		})

		It("masks helm repo password in application and helm source", func() {
			repo := &bkmsapp.HelmRepoConfig{RepoURL: "https://repo", Username: "u", Password: "pw"}
			source := &bkmsapp.HelmSource{HelmRepoConfig: repo}
			app := &bkmsapp.Application{Name: "app", HelmSpec: &bkmsapp.HelmSpec{HelmSource: source}}

			masked := recorded[bkmsapp.Application](app)
			Expect(masked.Name).To(Equal("app"))
			Expect(masked.HelmSpec.HelmSource.HelmRepoConfig.Username).To(Equal("u"))
			Expect(masked.HelmSpec.HelmSource.HelmRepoConfig.Password).To(Equal(mask))
			Expect(recorded[bkmsapp.HelmSource](source).HelmRepoConfig.Password).To(Equal(mask))
			Expect(repo.Password).To(Equal("pw"))

			Expect(recorded[bkmsapp.Application](&bkmsapp.Application{Name: "a"}).HelmSpec).To(BeNil())
		})

		It("masks image password in build config", func() {
			img := &build.ImageConfig{Name: "img", Username: "u", Password: "pw"}
			cfg := &build.Config{AppID: "app", Image: img}

			masked := recorded[build.Config](cfg)
			Expect(masked.AppID).To(Equal("app"))
			Expect(masked.Image.Username).To(Equal("u"))
			Expect(masked.Image.Password).To(Equal(mask))
			Expect(img.Password).To(Equal("pw"))

			Expect(recorded[build.Config](&build.Config{AppID: "app"}).Image).To(BeNil())
		})

		It("masks polaris tokens including applied fields", func() {
			applied := &polaris.RedeployRequiredFields{InstanceKey: "k", PolarisToken: "applied"}
			cfg := &polaris.PolarisConfig{
				Name:  "cfg",
				AppID: "app",
				EnvStates: map[string]polaris.PolarisEnvState{
					"prod": {AppliedFields: applied, LastError: "e"},
					"test": {},
				},
			}
			cfg.PolarisToken = "tk"

			masked := recorded[polaris.PolarisConfig](cfg)
			Expect(masked.Name).To(Equal("cfg"))
			Expect(masked.PolarisToken).To(Equal(mask))
			Expect(masked.EnvStates["prod"].AppliedFields.InstanceKey).To(Equal("k"))
			Expect(masked.EnvStates["prod"].AppliedFields.PolarisToken).To(Equal(mask))
			Expect(masked.EnvStates["prod"].LastError).To(Equal("e"))
			Expect(masked.EnvStates["test"].AppliedFields).To(BeNil())

			Expect(cfg.PolarisToken).To(Equal("tk"))
			Expect(applied.PolarisToken).To(Equal("applied"))
		})
	})
})
