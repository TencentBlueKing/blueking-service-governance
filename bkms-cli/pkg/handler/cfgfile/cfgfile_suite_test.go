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

package cfgfile

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

func TestCfgFile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/handler/cfgfile Suite")
}

const testAppID = "demo"

// expectDefs 让列表接口返回给定的配置文件定义。
func expectDefs(cli *mocks.MockClient, defs ...client.AppConfigFileDef) {
	cli.EXPECT().ListAppConfigFileDefs(mock.Anything, testAppID).Return(defs, nil)
}

// expectEnvs 让应用环境列表接口返回给定的环境名。
func expectEnvs(cli *mocks.MockClient, envNames ...string) {
	envs := make([]client.Env, 0, len(envNames))
	for _, name := range envNames {
		envs = append(envs, client.Env{Name: name})
	}
	cli.EXPECT().ListAppEnvs(mock.Anything, testAppID).Return(envs, nil)
}

// expectAppType 让应用详情接口返回给定的应用类型。
func expectAppType(cli *mocks.MockClient, appType string) {
	cli.EXPECT().GetApp(mock.Anything, testAppID).Return(&client.AppFull{ID: testAppID, Type: appType}, nil)
}

// expectEnvInstances 让环境实例接口返回默认文件和给定环境的实例。
func expectEnvInstances(cli *mocks.MockClient, defID string, envNames ...string) {
	items := []client.AppConfigFileEnvInstance{{FileID: defID + "-default"}}
	for _, name := range envNames {
		items = append(items, client.AppConfigFileEnvInstance{FileID: defID + "-" + name, EnvName: name})
	}
	cli.EXPECT().ListAppConfigFileEnvInstances(mock.Anything, testAppID, defID).Return(items, nil)
}

func plainDef(id, name, mountDir string) client.AppConfigFileDef {
	return client.AppConfigFileDef{
		ID:              id,
		FileID:          id + "-default",
		Name:            name,
		ConfigKind:      client.ConfigKindPlain,
		MountDir:        mountDir,
		IsUnifiedConfig: true,
		CurrentVersion:  1,
	}
}

var _ = Describe("ResolveDef", func() {
	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("selects a file by name", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"), plainDef("d2", "b.conf", "/etc"))

		def, err := ResolveDef(ctx, cli, testAppID, FileRef{Name: "b.conf"})

		Expect(err).NotTo(HaveOccurred())
		Expect(def.ID).To(Equal("d2"))
	})

	It("selects a file by ID", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"), plainDef("d2", "a.conf", "/data"))

		def, err := ResolveDef(ctx, cli, testAppID, FileRef{ID: "d2"})

		Expect(err).NotTo(HaveOccurred())
		Expect(def.MountDir).To(Equal("/data"))
	})

	It("disambiguates files with the same name by cleaned mount dir", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"), plainDef("d2", "a.conf", "/data/conf"))

		def, err := ResolveDef(ctx, cli, testAppID, FileRef{Name: "a.conf", MountDir: "/data/conf/"})

		Expect(err).NotTo(HaveOccurred())
		Expect(def.ID).To(Equal("d2"))
	})

	It("reports candidates when multiple files match", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"), plainDef("d2", "a.conf", "/data"))

		_, err := ResolveDef(ctx, cli, testAppID, FileRef{Name: "a.conf"})

		Expect(err).To(MatchError(ContainSubstring("multiple config files match")))
		Expect(err.Error()).To(ContainSubstring("d1"))
		Expect(err.Error()).To(ContainSubstring("d2"))
	})

	It("reports available files when nothing matches", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"))

		_, err := ResolveDef(ctx, cli, testAppID, FileRef{Name: "missing.conf"})

		Expect(err).To(MatchError(ContainSubstring("not found")))
		Expect(err.Error()).To(ContainSubstring("a.conf"))
	})
})

var _ = Describe("env validation", func() {
	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("rejects --env for helm apps", func() {
		expectDefs(cli, plainDef("d1", "values", ""))
		expectAppType(cli, constant.AppTypeHelm)

		_, err := View(ctx, cli, testAppID, FileRef{Name: "values"}, "prod")

		Expect(err).To(MatchError(ContainSubstring("no per-environment config files")))
	})

	It("rejects unknown environments", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"))
		expectAppType(cli, "trpc")
		expectEnvs(cli, "dev", "prod")

		_, err := View(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prd")

		Expect(err).To(MatchError(ContainSubstring("env prd not found")))
	})

	It("rejects environments a plain file is not mounted on", func() {
		def := plainDef("d1", "a.conf", "/etc")
		def.MountedEnvNames = []string{"dev"}
		expectDefs(cli, def)
		expectAppType(cli, "trpc")
		expectEnvs(cli, "dev", "prod")

		_, err := View(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod")

		Expect(err).To(MatchError(ContainSubstring("not mounted on env prod")))
	})
})
