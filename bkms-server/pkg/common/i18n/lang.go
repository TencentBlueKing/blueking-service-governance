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

package i18n

import (
	"context"

	"github.com/samber/lo"
	"golang.org/x/text/language"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/ctxkey"
)

// Lang 支持的语言
type Lang string

const (
	// LangZH 中文
	LangZH Lang = "zh-cn"
	// LangEN 英文
	LangEN Lang = "en"

	// DefaultLang 默认语言，获取不到语言信息时 fallback
	DefaultLang = LangZH
)

// supportedLanguages 支持的语言列表，语言协商与资源加载校验均由此派生；
// 新增语言需：在上方声明 Lang 常量、在此追加一项、新增 locales/<lang>.po（缺失则启动失败），未翻译的条目回退到英文原文
var supportedLanguages = []supportedLanguage{
	{LangZH, language.Chinese},
	{LangEN, language.English},
}

var (
	supportedLangs = lo.Map(supportedLanguages, func(s supportedLanguage, _ int) Lang { return s.lang })
	matcher        = language.NewMatcher(
		lo.Map(supportedLanguages, func(s supportedLanguage, _ int) language.Tag { return s.tag }),
	)
)

type supportedLanguage struct {
	lang Lang
	tag  language.Tag
}

// Negotiate 按 blueking_language cookie、Accept-Language 的优先级协商语言，均无法识别时返回默认语言
func Negotiate(cookie, acceptLanguage string) Lang {
	if tag, err := language.Parse(cookie); cookie != "" && err == nil {
		return match(tag)
	}
	if tags, _, err := language.ParseAcceptLanguage(acceptLanguage); err == nil && len(tags) > 0 {
		return match(tags...)
	}
	return DefaultLang
}

func match(tags ...language.Tag) Lang {
	_, idx, confidence := matcher.Match(tags...)
	if confidence == language.No {
		return DefaultLang
	}
	return supportedLangs[idx]
}

// WithLang 将语言写入 ctx
func WithLang(ctx context.Context, lang Lang) context.Context {
	return context.WithValue(ctx, ctxkey.Lang, lang)
}

// LangFromContext 读取 ctx 中的语言，不存在时返回默认语言
func LangFromContext(ctx context.Context) Lang {
	if lang, ok := ctx.Value(ctxkey.Lang).(Lang); ok {
		return lang
	}
	return DefaultLang
}
