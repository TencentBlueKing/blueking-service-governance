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

package dockerfile

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

const (
	// LanguageGo 指明 tRPC 应用使用 Go 语言
	LanguageGo = "go"
	// LanguageCpp 指明 tRPC 应用使用 C++ 语言
	LanguageCpp = "cpp"

	goTemplatePath  = "templates/go.Dockerfile.tmpl"
	cppTemplatePath = "templates/cpp.Dockerfile.tmpl"
)

const (
	defaultGoBuildCommandTemplate  = "CGO_ENABLED=0 go build -trimpath -o /out/%s ."
	defaultCppBuildCommandTemplate = "cmake -S . -B build -DCMAKE_BUILD_TYPE=Release" +
		" && cmake --build build --target %s -j$(nproc)" +
		" && cp build/%s /out/%s"
)

var safeAppNameRegexp = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var safeBuildArgNameRegexp = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

//go:embed templates/*.tmpl
var embeddedTemplates embed.FS

var templateFiles fs.FS = embeddedTemplates

var languageSpecs = map[string]languageSpec{
	LanguageGo: {
		templatePath:        goTemplatePath,
		defaultBuildCommand: defaultGoBuildCommand,
		prepareTemplateData: prepareGoTemplateData,
	},
	LanguageCpp: {
		templatePath:        cppTemplatePath,
		defaultBuildCommand: defaultCppBuildCommand,
	},
}

type languageSpec struct {
	templatePath        string
	defaultBuildCommand func(appName string) string
	// prepareTemplateData 用于在通用字段填充完成后，追加语言特定的模板数据
	//
	// 目前只有 Go 需要基于源码目录探测 go.sum 是否存在；C++ 分支暂时没有类似需求，
	// 因此保持为空。后续若 C++ 也需要检查 CMakeLists.txt 等文件，可复用同一钩子
	prepareTemplateData func(input Input, data *templateData) error
}

// Input 保存 Dockerfile 模板渲染所需的结构化输入
type Input struct {
	Language            string
	BuilderImage        string
	RunnerImage         string
	PreBuildCommands    string
	BuildCommands       string
	RuntimeEnvCommands  string
	StartCommand        string
	DockerBuildArgNames string
	DockerBuildDir      string
	ImageName           string
}

type templateData struct {
	BuilderImage        string
	RunnerImage         string
	DockerBuildArgNames []string
	PreBuildCommands    []string
	BuildCommands       []string
	RuntimeEnvCommands  []string
	StartCommand        string
	AppName             string
	DependencyFiles     []string
}

// Render 使用与语言类型匹配的模板渲染 BKMS 默认应用 Dockerfile
func Render(input Input) (string, error) {
	language := strings.TrimSpace(input.Language)
	if language == "" {
		return "", errors.Errorf("missing Dockerfile language")
	}

	spec, ok := languageSpecs[language]
	if !ok {
		return "", errors.Errorf("unsupported Dockerfile language %q", language)
	}

	appName, err := appNameFromImageName(input.ImageName)
	if err != nil {
		return "", err
	}

	templateContent, err := fs.ReadFile(templateFiles, spec.templatePath)
	if err != nil {
		return "", errors.Wrapf(
			err, "read Dockerfile template for language %q from %s", language, spec.templatePath,
		)
	}

	tmpl, err := template.New("dockerfile").Funcs(template.FuncMap{
		"quoteStartCommand": strconv.Quote,
	}).Parse(string(templateContent))
	if err != nil {
		return "", errors.Wrapf(err, "parse Dockerfile template for language %q", language)
	}

	data, err := buildTemplateData(input, spec, appName)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", errors.Wrapf(err, "render Dockerfile template for language %q", language)
	}
	return buf.String(), nil
}

func appNameFromImageName(imageName string) (string, error) {
	appName := strings.TrimSpace(imageName)
	if appName == "" {
		return "", errors.Errorf("missing image name")
	}
	if appName == "." || appName == ".." || !safeAppNameRegexp.MatchString(appName) {
		return "", errors.Errorf(
			"invalid image name %q: only letters, digits, dot, underscore and hyphen are allowed", imageName,
		)
	}
	return appName, nil
}

func buildTemplateData(input Input, spec languageSpec, appName string) (templateData, error) {
	buildCommands, err := parseCommands("build commands", input.BuildCommands)
	if err != nil {
		return templateData{}, err
	}
	if len(buildCommands) == 0 {
		buildCommands = []string{spec.defaultBuildCommand(appName)}
	}
	preBuildCommands, err := parseCommands("pre-build commands", input.PreBuildCommands)
	if err != nil {
		return templateData{}, err
	}
	runtimeEnvCommands, err := parseCommands("runtime env commands", input.RuntimeEnvCommands)
	if err != nil {
		return templateData{}, err
	}
	dockerBuildArgs, err := parseDockerBuildArgNames(input.DockerBuildArgNames)
	if err != nil {
		return templateData{}, err
	}
	data := templateData{
		BuilderImage:        input.BuilderImage,
		RunnerImage:         input.RunnerImage,
		DockerBuildArgNames: dockerBuildArgs,
		PreBuildCommands:    preBuildCommands,
		BuildCommands:       buildCommands,
		RuntimeEnvCommands:  runtimeEnvCommands,
		StartCommand:        strings.TrimSpace(input.StartCommand),
		AppName:             appName,
	}
	if spec.prepareTemplateData != nil {
		if err = spec.prepareTemplateData(input, &data); err != nil {
			return templateData{}, err
		}
	}
	return data, nil
}

// defaultGoBuildCommand 构造未指定 BuildCommands 时使用的默认 go build 命令
//
// 这里的 appName 来自 ImageName 的安全转换结果：BKMS 平台约定应用名与镜像名、构建产物文件名保持一致，
// 因此可以直接用它作为 `-o /out/<appName>` 的产物路径
func defaultGoBuildCommand(appName string) string {
	return fmt.Sprintf(defaultGoBuildCommandTemplate, appName)
}

// defaultCppBuildCommand 构造未指定 BuildCommands 时使用的默认 C++ 构建命令
//
// 默认约定 CMake target、构建产物文件名与应用名保持一致，构建完成后统一复制到 `/out/<appName>`，
// 这样后续产物校验与 runner 阶段可以复用跨语言的固定路径规则
func defaultCppBuildCommand(appName string) string {
	return fmt.Sprintf(defaultCppBuildCommandTemplate, appName, appName, appName)
}

// parseDockerBuildArgNames 解析 BKMS_DOCKER_BUILD_ARG_NAMES 传入的 JSON 字符串数组
//
// 这里只声明 ARG 名称，不把 value 写入文件，避免将可能敏感的构建参数固化到生成产物中
func parseDockerBuildArgNames(argNames string) ([]string, error) {
	if strings.TrimSpace(argNames) == "" {
		return nil, nil
	}

	var names []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(argNames)), &names); err != nil {
		return nil, errors.Wrap(err, "unmarshal Docker build arg names as JSON string array")
	}
	return normalizeDockerBuildArgNames(names), nil
}

func normalizeDockerBuildArgNames(names []string) []string {
	seen := make(map[string]struct{})
	return lo.FilterMap(names, func(name string, _ int) (string, bool) {
		name = strings.TrimSpace(name)
		if name == "" || !safeBuildArgNameRegexp.MatchString(name) {
			return "", false
		}
		if _, ok := seen[name]; ok {
			return "", false
		}
		seen[name] = struct{}{}
		return name, true
	})
}

// parseCommands 解析流水线传入的 Dockerfile 命令参数
//
// 非空 commands 必须是 JSON 字符串数组，数组中每个元素代表一条 Dockerfile RUN 命令；
// 解析后会统一去除命令首尾空白并过滤空命令，避免模板渲染出空 RUN 指令
func parseCommands(field string, commands string) ([]string, error) {
	if strings.TrimSpace(commands) == "" {
		return nil, nil
	}

	trimmedPayload := strings.TrimSpace(commands)
	if !strings.HasPrefix(trimmedPayload, "[") {
		return nil, errors.Errorf("Dockerfile %s must be JSON string array", field)
	}

	var parsedCommands []string
	if err := json.Unmarshal([]byte(trimmedPayload), &parsedCommands); err != nil {
		return nil, errors.Wrapf(err, "unmarshal Dockerfile %s as JSON string array", field)
	}
	return normalizeCommandList(parsedCommands), nil
}

// normalizeCommandList 归一化解析后的命令列表
//
// 每条命令会先去除首尾空白，空字符串会被过滤，确保模板只渲染有效命令
func normalizeCommandList(commands []string) []string {
	return lo.FilterMap(commands, func(line string, _ int) (string, bool) {
		command := strings.TrimSpace(line)
		return command, command != ""
	})
}
