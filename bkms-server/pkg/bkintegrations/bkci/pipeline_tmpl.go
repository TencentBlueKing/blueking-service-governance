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
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

const (
	// 默认流水线构建镜像 Code
	defaultPipelineBuilderImageCode = "tlinux3_ci"
	// 默认流水线构建镜像版本
	defaultPipelineBuilderImageVersion = "2.*"

	pipelineTmplCtxKeyImageCode    = "builderImageCode"
	pipelineTmplCtxKeyImageVersion = "builderImageVersion"
)

// PipelineTemplatesReloader 负责将本地目录下的流水线模板重新加载到数据库
type PipelineTemplatesReloader struct {
	cfg   *config.Config
	store PipelineTemplateStore
}

// newPipelineTemplatesReloader 创建 PipelineTemplatesReloader
func newPipelineTemplatesReloader(cfg *config.Config, store PipelineTemplateStore) *PipelineTemplatesReloader {
	return &PipelineTemplatesReloader{cfg: cfg, store: store}
}

// Reload 重新加载流水线模板到数据库
// 注意：目前 Reload 不会删除已加载的场景模板
func (r *PipelineTemplatesReloader) Reload(ctx context.Context) error {
	if r.cfg == nil {
		return errors.New("config uninitialized, cannot reload pipeline templates")
	}
	tmplDir := r.cfg.BKCI.PipelineTmpl.BaseDir
	// 如果没有指定流水线模板目录，则跳过
	if tmplDir == "" {
		log.Warn(ctx, "bkci pipeline template directory is not set, skip reload pipeline templates")
		return nil
	}

	// 读取目录中的所有文件
	files, err := os.ReadDir(tmplDir)
	if err != nil {
		return errors.Wrapf(err, "read dir %s", tmplDir)
	}

	renderContext := r.buildRenderContext()
	for _, file := range files {
		// 如果不是 json 格式的则跳过
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		if err = r.reloadOne(ctx, tmplDir, file.Name(), renderContext); err != nil {
			return err
		}
	}
	return nil
}

// buildRenderContext 构造流水线模板的渲染上下文
func (r *PipelineTemplatesReloader) buildRenderContext() map[string]any {
	imageCode := defaultPipelineBuilderImageCode
	imageVersion := defaultPipelineBuilderImageVersion
	if r.cfg != nil {
		imageCode = cmp.Or(r.cfg.BKCI.PipelineTmpl.BuilderImageCode, defaultPipelineBuilderImageCode)
		imageVersion = cmp.Or(r.cfg.BKCI.PipelineTmpl.BuilderImageVersion, defaultPipelineBuilderImageVersion)
	}
	return map[string]any{
		pipelineTmplCtxKeyImageCode:    imageCode,
		pipelineTmplCtxKeyImageVersion: imageVersion,
	}
}

// reloadOne 加载单个流水线模板文件
func (r *PipelineTemplatesReloader) reloadOne(
	ctx context.Context, tmplDir, filename string, renderContext map[string]any,
) error {
	rawData, err := os.ReadFile(filepath.Join(tmplDir, filename))
	if err != nil {
		return errors.Wrapf(err, "read file %s", filename)
	}

	renderedData, err := r.renderTemplate(filename, rawData, renderContext)
	if err != nil {
		return err
	}

	var tmpl PipelineTemplate
	if err = json.Unmarshal(renderedData, &tmpl); err != nil {
		return errors.Wrapf(err, "unmarshal pipeline template file %s", filename)
	}
	if err = r.validateVersion(&tmpl); err != nil {
		return errors.Wrapf(err, "validate pipeline template file %s", filename)
	}

	pipelineTmplInfo := fmt.Sprintf("pipeline template %s (id: %s, type: %s)", tmpl.Name, tmpl.ID, tmpl.Type)
	if err = r.store.Upsert(ctx, &tmpl); err != nil {
		return errors.Wrapf(err, "upsert %s", pipelineTmplInfo)
	}
	log.Infof(ctx, "load %s success", pipelineTmplInfo)
	return nil
}

// validateVersion 校验流水线模板的 semver 版本是否合法
func (r *PipelineTemplatesReloader) validateVersion(tmpl *PipelineTemplate) error {
	if tmpl.Version == "" {
		return errors.Errorf("pipeline template type %s missing version", tmpl.Type)
	}
	if _, err := semver.StrictNewVersion(tmpl.Version); err != nil {
		return errors.Wrapf(err, "pipeline template type %s has invalid version %s", tmpl.Type, tmpl.Version)
	}
	return nil
}

// renderTemplate 渲染流水线模板
func (r *PipelineTemplatesReloader) renderTemplate(
	filename string, rawData []byte, renderContext map[string]any,
) ([]byte, error) {
	// 使用 [[ ]] Delims 避免与 json 括号冲突
	tmpl, err := template.New(filename).
		Option("missingkey=error").
		Delims("[[", "]]").
		Parse(string(rawData))
	if err != nil {
		return nil, errors.Wrapf(err, "parse pipeline template %s", filename)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, renderContext); err != nil {
		return nil, errors.Wrapf(err, "render pipeline template %s", filename)
	}
	return buf.Bytes(), nil
}

// ReloadPipelineTemplates 重新加载流水线模板到数据库
func ReloadPipelineTemplates(ctx context.Context) error {
	store, err := NewDBPipelineTemplateStore(database.Client(), database.Name())
	if err != nil {
		return errors.Wrapf(err, "new db pipeline template store")
	}
	return newPipelineTemplatesReloader(config.G, store).Reload(ctx)
}
