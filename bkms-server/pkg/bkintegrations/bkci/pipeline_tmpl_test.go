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
	"github.com/samber/lo"

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
			Expect(renderContext).NotTo(HaveKey(pipelineTmplCtxKeyCallbackURL))
			Expect(renderContext).NotTo(HaveKey(pipelineTmplCtxKeyCredentialID))
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

	Describe("renderPipelineTemplate", func() {
		It("should render dockerfile template with generator linux script", func() {
			templatePath := filepath.Join("assets", "pipeline_templates", "dockerfile.json")
			rawData, err := os.ReadFile(templatePath)
			Expect(err).NotTo(HaveOccurred())

			renderedData, err := renderPipelineTemplate("dockerfile.json", rawData, map[string]any{
				pipelineTmplCtxKeyImageCode:    testBuilderImageCode,
				pipelineTmplCtxKeyImageVersion: testBuilderImageVersion,
			})
			Expect(err).NotTo(HaveOccurred())

			var rendered map[string]any
			Expect(json.Unmarshal(renderedData, &rendered)).To(Succeed())
			Expect(rendered["version"]).To(Equal("1.6.0"))
			// 社区版没有构建并推送镜像插件，渲染结果中不得再出现
			Expect(string(renderedData)).NotTo(ContainSubstring("DockerBuildAndPushImage"))

			stages := rendered["stages"].([]any)
			triggerContainer := stages[0].(map[string]any)["containers"].([]any)[0].(map[string]any)
			params := triggerContainer["params"].([]any)
			paramIDs := make([]string, 0, len(params))
			for _, param := range params {
				paramIDs = append(paramIDs, param.(map[string]any)["id"].(string))
			}
			Expect(paramIDs).To(ContainElement("BKMS_DOCKER_BUILD_ARG_NAMES"))
			Expect(paramIDs).To(ContainElement("BKMS_IMAGE_REGISTRY_HOST"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_SOURCE_TYPE"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_LANGUAGE"))
			Expect(paramIDs).To(ContainElement("BKMS_IMAGE_BUILD_TOOLCHAIN_BASE_URL"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_BUILDER_IMAGE"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_START_COMMAND"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_EXTRA_FILES"))
			Expect(paramIDs).To(ContainElement("BKMS_DOCKERFILE_ARTIFACT_DIR"))
			Expect(paramIDs).NotTo(ContainElement("BKMS_DOCKERFILE_GENERATOR_BASE_URL"))

			// 产物目录允许为空：未配置时由生成器套用语言默认值
			artifactDirParam := lo.FindOrElse(params, nil, func(param any) bool {
				return param.(map[string]any)["id"] == "BKMS_DOCKERFILE_ARTIFACT_DIR"
			}).(map[string]any)
			Expect(artifactDirParam["valueNotEmpty"]).To(BeFalse())

			buildContainer := stages[1].(map[string]any)["containers"].([]any)[0].(map[string]any)
			elements := buildContainer["elements"].([]any)
			Expect(elements).To(HaveLen(3))
			for _, element := range elements {
				Expect(element.(map[string]any)["atomCode"]).NotTo(Equal("DockerBuildAndPushImage"))
			}
			scriptElement := elements[1].(map[string]any)
			Expect(scriptElement["name"]).To(Equal("生成 Dockerfile"))
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
			Expect(script).To(ContainSubstring("BKMS_DOCKERFILE_EXTRA_FILES"))
			Expect(script).To(ContainSubstring(`echo "BKMS_DOCKERFILE_ARTIFACT_DIR" "$BKMS_DOCKERFILE_ARTIFACT_DIR"`))
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

		It("should render dockerfile template with docker build and push linux script", func() {
			templatePath := filepath.Join("assets", "pipeline_templates", "dockerfile.json")
			rawData, err := os.ReadFile(templatePath)
			Expect(err).NotTo(HaveOccurred())

			renderedData, err := renderPipelineTemplate("dockerfile.json", rawData, map[string]any{
				pipelineTmplCtxKeyImageCode:    testBuilderImageCode,
				pipelineTmplCtxKeyImageVersion: testBuilderImageVersion,
			})
			Expect(err).NotTo(HaveOccurred())

			var rendered map[string]any
			Expect(json.Unmarshal(renderedData, &rendered)).To(Succeed())
			buildContainer := rendered["stages"].([]any)[1].(map[string]any)["containers"].([]any)[0].(map[string]any)

			// 账密经 Job 环境变量从凭证 ID 兑换，模板中只有 settings 引用，不含明文
			Expect(buildContainer["customEnv"]).To(ConsistOf(
				map[string]any{
					"key":   "BKMS_IMAGE_REGISTRY_USERNAME",
					"value": "${{settings.${BKMS_IMAGE_CREDENTIAL}.username}}",
				},
				map[string]any{
					"key":   "BKMS_IMAGE_REGISTRY_PASSWORD",
					"value": "${{settings.${BKMS_IMAGE_CREDENTIAL}.password}}",
				},
			))

			elements := buildContainer["elements"].([]any)
			buildElement := elements[len(elements)-1].(map[string]any)
			Expect(buildElement["name"]).To(Equal("构建并推送 Docker 镜像"))
			Expect(buildElement["atomCode"]).To(Equal("linuxScript"))
			Expect(buildElement["@type"]).To(Equal("linuxScript"))
			Expect(buildElement["classType"]).To(Equal("linuxScript"))
			Expect(buildElement["scriptType"]).To(Equal("SHELL"))
			// 失败必须让步骤失败，不能被 continueNoneZero 吞掉
			Expect(buildElement["continueNoneZero"]).To(BeFalse())
			additionalOptions := buildElement["additionalOptions"].(map[string]any)
			Expect(additionalOptions["enable"]).To(BeTrue())
			Expect(additionalOptions["continueWhenFailed"]).To(BeFalse())
			Expect(additionalOptions["timeout"]).To(BeNumerically("==", 900))

			script := buildElement["script"].(string)
			Expect(script).To(HavePrefix("set -euo pipefail\n"))
			Expect(script).To(ContainSubstring(
				"for var in BKMS_DOCKERFILE_PATH BKMS_DOCKER_BUILD_DIR BKMS_IMAGE_REGISTRY " +
					"BKMS_IMAGE_NAME BKMS_IMAGE_TAG BKMS_IMAGE_CREDENTIAL; do",
			))
			Expect(script).To(ContainSubstring(`if ! command -v docker >/dev/null 2>&1; then`))
			// 账密通过带引号的 heredoc 承接，由蓝盾在执行前替换为明文
			Expect(script).To(ContainSubstring(
				"IFS= read -r registry_username <<'BKMS_CREDENTIAL_EOF'\n${BKMS_IMAGE_REGISTRY_USERNAME}\n",
			))
			Expect(script).To(ContainSubstring(
				"IFS= read -r registry_password <<'BKMS_CREDENTIAL_EOF'\n${BKMS_IMAGE_REGISTRY_PASSWORD}\n",
			))
			// settings 引用残留、Job 环境变量未生效两种兑换失败都要拦下
			Expect(script).To(ContainSubstring(`'' | *'{{settings.'* | *'{BKMS_IMAGE_REGISTRY_'*)`))
			// 登录态写入临时 DOCKER_CONFIG，取消或超时也要清理；密码只走 stdin，登录后不再保留
			Expect(script).To(ContainSubstring(`DOCKER_CONFIG="$(mktemp -d)"`))
			Expect(script).To(ContainSubstring("trap cleanup_docker_config EXIT\n"))
			Expect(script).To(ContainSubstring("trap 'exit 130' INT\n"))
			Expect(script).To(ContainSubstring("trap 'exit 143' TERM\n"))
			Expect(script).To(ContainSubstring(
				`printf '%s' "$registry_password" | docker login --username "$registry_username" --password-stdin "$registry_host"`,
			))
			Expect(script).To(ContainSubstring("unset registry_username registry_password\n"))
			Expect(script).NotTo(ContainSubstring(`echo "$registry_password"`))
			Expect(script).NotTo(ContainSubstring("--password \""))
			// 构建参数不能从环境变量读：蓝盾导出时会删掉换行和单引号
			Expect(
				script,
			).To(ContainSubstring("done <<'BKMS_BUILD_ARGS_EOF'\n${BKMS_DOCKER_BUILD_ARGS}\nBKMS_BUILD_ARGS_EOF\n"))
			Expect(script).NotTo(ContainSubstring(`"${BKMS_DOCKER_BUILD_ARGS:-}"`))
			Expect(script).To(ContainSubstring(`*'{BKMS_DOCKER_BUILD_ARGS}'*)`))
			// 构建参数为空时不追加 --build-arg
			Expect(script).To(ContainSubstring(
				`docker build -f "$BKMS_DOCKERFILE_PATH" -t "$target_image" ` +
					`${build_args[@]+"${build_args[@]}"} "$BKMS_DOCKER_BUILD_DIR"`,
			))
			// 镜像源地址兜底去掉 scheme 和末尾斜杠后再拼 host 与目标镜像
			Expect(script).To(ContainSubstring(`registry="${BKMS_IMAGE_REGISTRY#http://}"`))
			Expect(script).To(ContainSubstring(`registry="${registry#https://}"`))
			Expect(script).To(ContainSubstring(`registry_host="${registry%%/*}"`))
			Expect(script).To(ContainSubstring(`target_image="$registry/$BKMS_IMAGE_NAME:$BKMS_IMAGE_TAG"`))
			Expect(script).To(ContainSubstring(`if ! docker push "$target_image"; then`))
			// 各阶段失败都要在日志中标明阶段
			Expect(script).To(ContainSubstring(`echo "[ERROR] docker login to $registry_host failed"`))
			Expect(script).To(ContainSubstring(`echo "[ERROR] docker build $target_image failed"`))
			Expect(script).To(ContainSubstring(`echo "[ERROR] docker push $target_image failed"`))
		})

		It("should return error when template key is missing", func() {
			rawData := []byte(`{"imageCode": "[[ .missingBuilderImageCode ]]"}`)

			_, err := renderPipelineTemplate("test.json", rawData, map[string]any{
				pipelineTmplCtxKeyImageCode: testBuilderImageCode,
			})
			Expect(err).To(HaveOccurred())
		})

		It("should keep instance-time placeholders after reload via template self-escape", func() {
			templatePath := filepath.Join("assets", "pipeline_templates", "build_trigger.json")
			rawData, err := os.ReadFile(templatePath)
			Expect(err).NotTo(HaveOccurred())
			// 资产原文用反引号自逃逸，避免 JSON 字符串内再嵌 \" 导致模板解析失败
			Expect(string(rawData)).To(ContainSubstring("[[ `[[ .appID ]]` ]]"))
			Expect(string(rawData)).To(ContainSubstring("[[ `[[ .callbackURL ]]` ]]"))
			Expect(string(rawData)).To(ContainSubstring("[[ `[[ .credentialID ]]` ]]"))

			reloader := newPipelineTemplatesReloader(nil, nil)
			renderedData, err := renderPipelineTemplate(
				"build_trigger.json", rawData, reloader.buildRenderContext(),
			)
			Expect(err).NotTo(HaveOccurred())

			var rendered map[string]any
			Expect(json.Unmarshal(renderedData, &rendered)).To(Succeed())
			Expect(rendered["version"]).To(Equal("1.0.0"))
			Expect(rendered["name"]).To(Equal("[bkms] 自动构建触发（[[ .appID ]]）"))
			// 自逃逸经 Reload 后留下二次渲染占位；全局镜像已落地
			Expect(string(renderedData)).To(ContainSubstring("[[ .callbackURL ]]"))
			Expect(string(renderedData)).To(ContainSubstring("[[ .credentialID ]]"))
			Expect(string(renderedData)).NotTo(ContainSubstring("[[ `[[ .callbackURL ]]` ]]"))
			Expect(string(renderedData)).To(ContainSubstring(defaultPipelineBuilderImageCode))
			Expect(string(renderedData)).NotTo(ContainSubstring("[[ .builderImageCode ]]"))
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
