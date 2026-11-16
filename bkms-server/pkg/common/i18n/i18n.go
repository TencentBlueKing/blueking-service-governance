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

// Package i18n 提供面向用户文案的国际化：按 key 查找内嵌的多语言模板，并以命名参数渲染
package i18n

import (
	"context"
	"embed"
	"io/fs"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localesFS embed.FS

// catalog key -> 语言 -> 文案模板
type catalog map[string]map[Lang]*template.Template

// messages 在启动时加载，资源文件有误时直接启动失败
var messages = mustLoad(localesFS)

func mustLoad(fsys fs.FS) catalog {
	c, err := load(fsys)
	if err != nil {
		panic(err)
	}
	return c
}

// load 加载 locales 目录下的全部 YAML 文件，要求每个 key 都提供所有支持语言的文案，且 key 不跨文件重复
func load(fsys fs.FS) (catalog, error) {
	files, err := fs.Glob(fsys, "locales/*.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "glob locale files")
	}

	c := catalog{}
	for _, file := range files {
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, errors.Wrapf(err, "read locale file %s", file)
		}
		var entries map[string]map[Lang]string
		if err = yaml.Unmarshal(data, &entries); err != nil {
			return nil, errors.Wrapf(err, "parse locale file %s", file)
		}

		for key, texts := range entries {
			if _, exists := c[key]; exists {
				return nil, errors.Errorf("locale key %s is duplicated in %s", key, file)
			}
			c[key] = make(map[Lang]*template.Template, len(supportedLangs))
			for _, lang := range supportedLangs {
				text := strings.TrimSpace(texts[lang])
				if text == "" {
					return nil, errors.Errorf("locale key %s in %s lacks %s text", key, file, lang)
				}
				tmpl, err := template.New(key).Option("missingkey=error").Parse(text)
				if err != nil {
					return nil, errors.Wrapf(err, "parse template of locale key %s (%s)", key, lang)
				}
				c[key][lang] = tmpl
			}
		}
	}
	return c, nil
}

func (c catalog) localize(lang Lang, key string, params map[string]string) (string, bool) {
	tmpl, ok := c[key][lang]
	if !ok {
		return "", false
	}
	if params == nil {
		params = map[string]string{}
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, params); err != nil {
		return "", false
	}
	return sb.String(), true
}

// Localize 按 ctx 中的语言渲染 key 对应的文案；key 不存在或缺少模板所需参数时返回 false，由调用方兜底
func Localize(ctx context.Context, key string, params map[string]string) (string, bool) {
	return messages.localize(LangFromContext(ctx), key, params)
}

// LocalizeOr 按 ctx 中的语言渲染 key 对应的文案，失败时返回 fallback，用于枚举名称等固定展示文本
func LocalizeOr(ctx context.Context, key string, params map[string]string, fallback string) string {
	if msg, ok := Localize(ctx, key, params); ok {
		return msg
	}
	return fallback
}

// LocalizeStored 翻译系统写入存储的默认文案（如默认环境名）：
// stored 与 key 的中文文案（写入时使用的语言）一致时，说明未被用户修改，按 ctx 中的语言渲染；否则原样返回 stored
func LocalizeStored(ctx context.Context, key string, params map[string]string, stored string) string {
	if original, ok := messages.localize(LangZH, key, params); !ok || original != stored {
		return stored
	}
	return LocalizeOr(ctx, key, params, stored)
}

// Has 判断 key 是否有对应的文案（不渲染、不校验参数），用于翻译完整性检查
func Has(key string) bool {
	_, ok := messages[key]
	return ok
}
