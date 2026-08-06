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
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const (
	goModFile = "go.mod"
	goSumFile = "go.sum"
)

// Go 依赖文件处理背景
//
// Go Dockerfile 需要先复制依赖描述文件并执行 go mod download，然后再复制完整源码
// 这样 Docker 构建缓存可以把“依赖下载”和“业务代码编译”拆成两层：只改业务代码时，
// 已下载的 module 依赖不会被反复拉取，能明显减少镜像构建耗时
//
// 本文件只负责为 Go 模板准备这部分依赖文件列表，不参与模板渲染本身：
// 1. 根据 Input.DockerBuildDir 找到 Docker 构建上下文目录
// 2. 校验 go.mod 必须存在且是普通文件，因为 go mod download 依赖它描述 module
// 3. 仅在 go.sum 存在且是普通文件时才加入复制列表，兼容部分新项目暂未生成 go.sum 的场景
// 4. 将最终文件列表写入 templateData.DependencyFiles，由 go.Dockerfile.tmpl 渲染 COPY 指令
//
// 这里刻意不感知上层环境变量或业务配置，只返回 dockerfile 包内的语义化错误
// 上层如果需要提示具体环境变量，可以在调用 Render 后继续包装错误

// prepareGoTemplateData 为 Go Dockerfile 模板补充依赖文件复制所需的数据
//
// Go 模板会先复制 go.mod 和可选的 go.sum 再执行 go mod download，以便更好地复用 Docker 构建缓存
// 因此这里会在 Docker 构建目录中校验 go.mod 必须存在，并只在 go.sum 是普通文件时才加入模板数据
func prepareGoTemplateData(input Input, data *templateData) error {
	dependencyFiles, err := collectExistingFiles(fileCheckInput{
		directory:     input.DockerBuildDir,
		requiredFiles: []string{goModFile},
		optionalFiles: []string{goSumFile},
	})
	if err != nil {
		return err
	}
	data.DependencyFiles = dependencyFiles
	return nil
}

type fileCheckInput struct {
	directory     string
	requiredFiles []string
	optionalFiles []string
}

// collectExistingFiles 在 Docker 构建目录下检查依赖描述文件是否存在
//
// 错误信息只描述语义（"Docker build directory"），不引用具体环境变量名，
// 以避免 dockerfile 包与上层的环境变量契约耦合。上层调用者若希望在错误里
// 补上环境变量名，可再通过 errors.Wrap 包装一层
func collectExistingFiles(input fileCheckInput) ([]string, error) {
	directory := strings.TrimSpace(input.directory)
	if directory == "" {
		return nil, errors.Errorf("missing Docker build directory")
	}
	info, err := os.Stat(directory)
	if err != nil {
		return nil, errors.Wrapf(err, "Docker build directory %q is not accessible", directory)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("Docker build directory %q is not a directory", directory)
	}

	files := make([]string, 0, len(input.requiredFiles)+len(input.optionalFiles))
	for _, name := range input.requiredFiles {
		if err = requireRegularFile(directory, name); err != nil {
			return nil, err
		}
		files = append(files, name)
	}
	for _, name := range input.optionalFiles {
		exists, checkErr := regularFileExists(directory, name)
		if checkErr != nil {
			return nil, checkErr
		}
		if exists {
			files = append(files, name)
		}
	}
	return files, nil
}

// requireRegularFile 校验指定文件必须存在且必须是普通文件
//
// 该函数用于必需依赖文件的检查：缺失、目录、符号异常或权限错误都会阻止继续渲染，
// 避免生成的 Dockerfile 在后续 docker build 阶段才暴露更难定位的问题
func requireRegularFile(directory string, name string) error {
	exists, err := regularFileExists(directory, name)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Errorf("required file %s is missing in %s", name, directory)
	}
	return nil
}

// regularFileExists 判断指定路径是否存在且是普通文件
//
// 文件不存在会被视为正常结果并返回 false；其他 os.Stat 错误会原样包装返回，
// 以便调用方区分可选文件缺失和目录不可访问等异常场景
func regularFileExists(directory string, name string) (bool, error) {
	path := filepath.Join(directory, name)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "check file %s in %s", name, directory)
	}
	return info.Mode().IsRegular(), nil
}
