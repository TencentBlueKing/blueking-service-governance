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

// Package app 是 bkms-dockerfile-generator 的应用编排层
//
// 它把三件事串起来：
//  1. 从流水线注入的环境变量中加载配置（pkg/config）
//  2. 根据配置渲染 Dockerfile 文本（pkg/dockerfile）
//  3. 把渲染结果写入工作空间中的目标路径（writer.go）
//
// 命令行入口只负责收集 os.Args、os.Environ 和 stdout，把它们交给 cmd 包。
package app

import (
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-dockerfile-generator/pkg/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-dockerfile-generator/pkg/dockerfile"
)

// Run 是默认 Dockerfile 生成流程入口
//
// 参数：
//   - environ: 通常来自 os.Environ()，包含流水线注入的 BKMS_DOCKERFILE_* 变量
//   - out:     用于输出面向用户的进度日志（例如"生成了哪个文件"），一般为 os.Stdout
//
// 返回的 error 会由 cmd 层直接向上暴露，最终由 main 处理非零退出码。
func Run(environ []string, out io.Writer) error {
	cfg, err := config.LoadFromEnviron(environ)
	if err != nil {
		return err
	}
	// SourceType=repository 表示走用户自带 Dockerfile 的分支，CLI 不参与生成
	if cfg.SourceType == config.SourceTypeRepository {
		_, _ = fmt.Fprintln(out, "skip Dockerfile generation because BKMS_DOCKERFILE_SOURCE_TYPE=repository")
		return nil
	}

	content, err := dockerfile.Render(dockerfile.Input{
		Language:            cfg.Language,
		BuilderImage:        cfg.BuilderImage,
		RunnerImage:         cfg.RunnerImage,
		PreBuildCommands:    cfg.PreBuildCommands,
		BuildCommands:       cfg.BuildCommands,
		RuntimeEnvCommands:  cfg.RuntimeEnvCommands,
		StartCommand:        cfg.StartCommand,
		DockerBuildArgNames: cfg.DockerBuildArgNames,
		DockerBuildDir:      cfg.DockerBuildDir,
		ImageName:           cfg.ImageName,
	})
	if err != nil {
		return errors.Wrapf(err, "render Dockerfile (buildDir: %s)", cfg.DockerBuildDir)
	}
	if err = writeDockerfile(cfg.DockerfilePath, []byte(content)); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "generated Dockerfile: %s\n", cfg.DockerfilePath)
	return nil
}
