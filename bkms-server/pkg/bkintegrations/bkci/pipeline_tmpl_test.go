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

package bkci

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

const (
	testBuilderImageCode    = "custom_ci"
	testBuilderImageVersion = "3.*"
)

var _ = Describe("PipelineTemplatesReloader", func() {
	Describe("buildRenderContext", func() {
		It("should build context with configured builder image", func() {
			testConfig := &config.Config{}
			testConfig.BKCI.PipelineTmpl.BuilderImageCode = testBuilderImageCode
			testConfig.BKCI.PipelineTmpl.BuilderImageVersion = testBuilderImageVersion

			renderContext := newPipelineTemplatesReloader(testConfig, nil).buildRenderContext()

			Expect(renderContext[pipelineTmplCtxKeyImageCode]).To(Equal(testBuilderImageCode))
			Expect(renderContext[pipelineTmplCtxKeyImageVersion]).To(Equal(testBuilderImageVersion))
		})

		It("should build context with default builder image when config is nil", func() {
			renderContext := newPipelineTemplatesReloader(nil, nil).buildRenderContext()

			Expect(renderContext[pipelineTmplCtxKeyImageCode]).To(Equal(defaultPipelineBuilderImageCode))
			Expect(renderContext[pipelineTmplCtxKeyImageVersion]).To(Equal(defaultPipelineBuilderImageVersion))
		})

		It("should build context with default builder image when builder image config is empty", func() {
			renderContext := newPipelineTemplatesReloader(&config.Config{}, nil).buildRenderContext()

			Expect(renderContext[pipelineTmplCtxKeyImageCode]).To(Equal(defaultPipelineBuilderImageCode))
			Expect(renderContext[pipelineTmplCtxKeyImageVersion]).To(Equal(defaultPipelineBuilderImageVersion))
		})
	})

	Describe("renderTemplate", func() {
		It("should render dockerfile template with generator linux script", func() {
			templatePath := filepath.Join("assets", "pipeline_templates", "dockerfile.json")
			rawData, err := os.ReadFile(templatePath)
			Expect(err).NotTo(HaveOccurred())

			reloader := newPipelineTemplatesReloader(nil, nil)
			renderedData, err := reloader.renderTemplate("dockerfile.json", rawData, map[string]any{
				pipelineTmplCtxKeyImageCode:    testBuilderImageCode,
				pipelineTmplCtxKeyImageVersion: testBuilderImageVersion,
			})
			Expect(err).NotTo(HaveOccurred())

			var rendered map[string]any
			Expect(json.Unmarshal(renderedData, &rendered)).To(Succeed())
			Expect(rendered["version"]).To(Equal("1.2.0"))

			stages := rendered["stages"].([]any)
			triggerContainer := stages[0].(map[string]any)["containers"].([]any)[0].(map[string]any)
			params := triggerContainer["params"].([]any)
			paramIDs := make([]string, 0, len(params))
			for _, param := range params {
				paramIDs = append(paramIDs, param.(map[string]any)["id"].(string))
			}
			Expect(paramIDs).To(ContainElement("BKMS_DOCKER_BUILD_ARG_NAMES"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_SOURCE_TYPE"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_LANGUAGE"))
			Expect(paramIDs).To(ContainElement("BKMS_IMAGE_BUILD_TOOLCHAIN_BASE_URL"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_BUILDER_IMAGE"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_START_COMMAND"))
			Expect(paramIDs).NotTo(ContainElement("BKMS_DOCKERFILE_GENERATOR_BASE_URL"))

			buildContainer := stages[1].(map[string]any)["containers"].([]any)[0].(map[string]any)
			elements := buildContainer["elements"].([]any)
			var scriptElement map[string]any
			for _, element := range elements {
				candidate := element.(map[string]any)
				if candidate["atomCode"] == "linuxScript" {
					scriptElement = candidate
					break
				}
			}
			Expect(scriptElement).NotTo(BeNil())
			Expect(scriptElement["@type"]).To(Equal("linuxScript"))
			Expect(scriptElement["classType"]).To(Equal("linuxScript"))
			Expect(scriptElement["scriptType"]).To(Equal("SHELL"))
			Expect(scriptElement["continueNoneZero"]).To(BeFalse())
			Expect(scriptElement["enableArchiveFile"]).To(BeFalse())
			additionalOptions := scriptElement["additionalOptions"].(map[string]any)
			Expect(additionalOptions["enable"]).To(BeTrue())
			Expect(additionalOptions["enableCustomEnv"]).To(BeTrue())
			Expect(additionalOptions["runCondition"]).To(Equal("PRE_TASK_SUCCESS"))
			Expect(additionalOptions["timeout"]).To(BeNumerically("==", 900))
			script := scriptElement["script"].(string)
			Expect(script).To(ContainSubstring("BKMS_DOCKERFILE_SOURCE_TYPE"))
			Expect(script).To(ContainSubstring("BKMS_DOCKERFILE_LANGUAGE"))
			Expect(script).To(ContainSubstring("bkms_generated"))
			Expect(script).NotTo(ContainSubstring("BKMS_DOCKERFILE_GENERATOR_BASE_URL"))
			Expect(script).To(ContainSubstring(`if [ "$BKMS_DOCKERFILE_SOURCE_TYPE" != "bkms_generated" ]; then`))
			Expect(script).To(ContainSubstring(`if [ -z "$BKMS_IMAGE_BUILD_TOOLCHAIN_BASE_URL" ]; then`))
			Expect(script).To(ContainSubstring(`echo "BKMS_IMAGE_BUILD_TOOLCHAIN_BASE_URL is empty"`))
			Expect(script).To(ContainSubstring("bkms-dockerfile-generator"))
			Expect(
				script,
			).To(ContainSubstring(`generator_url="${BKMS_IMAGE_BUILD_TOOLCHAIN_BASE_URL%/}/bkms-dockerfile-generator"`))
			Expect(script).To(ContainSubstring(`curl -fsSL "$generator_url" -o "$generator_path"`))
			Expect(script).To(ContainSubstring(`echo "BKMS Dockerfile Generator version"`))
			Expect(script).To(ContainSubstring(`"$generator_path" version`))
			Expect(script).To(ContainSubstring("\n\"$generator_path\"\n"))
			// generator binary 应下载到 /tmp，避免进入仓库工作区或 Docker build context
			Expect(script).To(ContainSubstring(`dockerfile_dir="$(dirname "$BKMS_DOCKERFILE_PATH")"`))
			Expect(script).To(ContainSubstring(`generator_path="/tmp/bkms-dockerfile-generator"`))
			Expect(script).NotTo(ContainSubstring(`generator_path="$dockerfile_dir/bkms-dockerfile-generator"`))
			// 生成后需校验 Dockerfile 存在并输出内容，便于流水线日志排查
			Expect(script).To(ContainSubstring(`if [ ! -f "$BKMS_DOCKERFILE_PATH" ]; then`))
			Expect(script).To(ContainSubstring(`echo "generated Dockerfile not found at $BKMS_DOCKERFILE_PATH"`))
			Expect(script).To(ContainSubstring(`echo "===== generated Dockerfile ($BKMS_DOCKERFILE_PATH) ====="`))
			Expect(script).To(ContainSubstring(`cat "$BKMS_DOCKERFILE_PATH"`))
			Expect(script).To(ContainSubstring(`echo "===== end of generated Dockerfile ====="`))
		})

		It("should return error when template key is missing", func() {
			rawData := []byte(`{"imageCode": "[[ .missingBuilderImageCode ]]"}`)

			reloader := newPipelineTemplatesReloader(nil, nil)
			_, err := reloader.renderTemplate("test.json", rawData, map[string]any{
				pipelineTmplCtxKeyImageCode: testBuilderImageCode,
			})

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("validateVersion", func() {
		reloader := newPipelineTemplatesReloader(nil, nil)

		It("should allow valid semver version", func() {
			tmpl := &PipelineTemplate{Type: "dockerfile", Version: "1.0.0"}

			err := reloader.validateVersion(tmpl)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when version is missing", func() {
			tmpl := &PipelineTemplate{Type: "dockerfile"}

			err := reloader.validateVersion(tmpl)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing version"))
		})

		It("should return error when version is not strict semver", func() {
			tmpl := &PipelineTemplate{Type: "dockerfile", Version: "1.*"}

			err := reloader.validateVersion(tmpl)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid version"))
		})
	})
})
