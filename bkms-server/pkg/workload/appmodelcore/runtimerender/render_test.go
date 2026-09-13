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

package runtimerender_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/runtimerender"
)

var _ = Describe("RenderConfigContents", func() {
	It("should render env var templates and produce ConfigFileParams", func() {
		items := []appcfg.MountableFile{
			{Name: "app.yaml", MountDir: "/etc/app", Content: "host: ${{ env.DB_HOST }}\nport: 3306"},
		}
		envVars := map[string]string{"DB_HOST": "127.0.0.1"}
		collector := envvarrefs.NewCollector(envVars)

		result, err := runtimerender.RenderConfigContents(items, envVars, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].FileName).To(Equal("app.yaml"))
		Expect(result[0].FilePath).To(Equal("/etc/app"))
		Expect(result[0].FileContent).To(Equal("host: 127.0.0.1\nport: 3306"))
		Expect(collector.UndefinedEnvVars()).To(BeEmpty())
	})

	It("should render multiple files", func() {
		items := []appcfg.MountableFile{
			{Name: "a.conf", MountDir: "/etc/a", Content: "key=${{ env.A }}"},
			{Name: "b.conf", MountDir: "/etc/b", Content: "key=${{ env.B }}"},
		}
		envVars := map[string]string{"A": "1", "B": "2"}
		collector := envvarrefs.NewCollector(envVars)

		result, err := runtimerender.RenderConfigContents(items, envVars, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(2))
		Expect(result[0].FileContent).To(Equal("key=1"))
		Expect(result[1].FileContent).To(Equal("key=2"))
	})

	It("should collect undefined env var references", func() {
		items := []appcfg.MountableFile{
			{Name: "app.yaml", MountDir: "/etc/app", Content: "host: ${{ env.UNDEFINED_VAR }}\n"},
		}
		envVars := map[string]string{"OTHER_VAR": "val"}
		collector := envvarrefs.NewCollector(envVars)

		result, err := runtimerender.RenderConfigContents(items, envVars, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))

		undefined := collector.UndefinedEnvVars()
		Expect(undefined).To(HaveLen(1))
		Expect(undefined[0].Key).To(Equal("UNDEFINED_VAR"))
		Expect(undefined[0].Sources).To(HaveLen(1))
		Expect(undefined[0].Sources[0].Name).To(Equal("app.yaml"))
	})

	It("should pass through content without templates unchanged", func() {
		items := []appcfg.MountableFile{
			{Name: "static.conf", MountDir: "/etc", Content: "no_template_here"},
		}
		collector := envvarrefs.NewCollector(nil)

		result, err := runtimerender.RenderConfigContents(items, nil, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].FileContent).To(Equal("no_template_here"))
	})

	It("should return empty result for empty input", func() {
		collector := envvarrefs.NewCollector(nil)

		result, err := runtimerender.RenderConfigContents(nil, nil, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("should return error when env var collection fails due to invalid template syntax", func() {
		items := []appcfg.MountableFile{
			{Name: "bad.yaml", MountDir: "/etc/app", Content: "${{ env. }}"},
		}
		collector := envvarrefs.NewCollector(nil)

		_, err := runtimerender.RenderConfigContents(items, nil, collector)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("collecting env vars from config bad.yaml"))
	})

	It("should return error for the failing file in a multi-file batch", func() {
		items := []appcfg.MountableFile{
			{Name: "good.yaml", MountDir: "/etc/a", Content: "key: value"},
			{Name: "bad.yaml", MountDir: "/etc/b", Content: "${{ env. }}"},
		}
		collector := envvarrefs.NewCollector(nil)

		_, err := runtimerender.RenderConfigContents(items, nil, collector)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("bad.yaml"))
	})

	It("should handle empty content string without error", func() {
		items := []appcfg.MountableFile{
			{Name: "empty.conf", MountDir: "/etc/app", Content: ""},
		}
		collector := envvarrefs.NewCollector(nil)

		result, err := runtimerender.RenderConfigContents(items, nil, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].FileContent).To(BeEmpty())
	})
})
