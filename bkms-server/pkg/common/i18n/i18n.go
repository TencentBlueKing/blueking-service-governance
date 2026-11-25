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

// Package i18n 以 gettext 的方式提供面向用户文案的国际化：源码中的英文原文即翻译 key（msgid），
// 译文放在 locales/<lang>.po 中，原文中的 printf 动词即参数占位符
package i18n

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"slices"

	"github.com/chai2010/gettext-go/po"
	"github.com/pkg/errors"
)

//go:embed locales/*.po
var localesFS embed.FS

// SourceLang 源码中文案使用的语言，原文即该语言的文案，没有翻译文件
const SourceLang = LangEN

// catalog 语言 -> 原文 -> 译文
type catalog map[Lang]map[string]string

// messages 在启动时加载，翻译文件有误时直接启动失败
var messages = mustLoad(localesFS)

func mustLoad(fsys fs.FS) catalog {
	c, err := load(fsys)
	if err != nil {
		panic(err)
	}
	return c
}

// load 加载除源语言外每种支持语言的 locales/<lang>.po。
// 未翻译（msgstr 为空）或标记为 fuzzy 的条目跳过，运行时回退到原文；
// 原文重复、译文与原文的 printf 参数不一致时返回错误
func load(fsys fs.FS) (catalog, error) {
	c := catalog{}
	for _, lang := range supportedLangs {
		if lang == SourceLang {
			continue
		}
		path := "locales/" + string(lang) + ".po"
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, errors.Wrapf(err, "read locale file %s", path)
		}
		file, err := po.Load(data)
		if err != nil {
			return nil, errors.Wrapf(err, "parse locale file %s", path)
		}

		entries := make(map[string]string, len(file.Messages))
		seen := make(map[string]bool, len(file.Messages))
		for _, msg := range file.Messages {
			if seen[msg.MsgId] {
				return nil, errors.Errorf("msgid %q is duplicated in %s", msg.MsgId, path)
			}
			seen[msg.MsgId] = true
			if msg.MsgStr == "" || slices.Contains(msg.Flags, "fuzzy") {
				continue
			}
			if err = checkPlaceholders(msg.MsgId, msg.MsgStr); err != nil {
				return nil, errors.Wrapf(err, "msgid %q in %s", msg.MsgId, path)
			}
			entries[msg.MsgId] = msg.MsgStr
		}
		c[lang] = entries
	}
	return c, nil
}

func (c catalog) translate(lang Lang, msgid string) (string, bool) {
	if lang == SourceLang {
		return msgid, c.has(msgid)
	}
	msgstr, ok := c[lang][msgid]
	return msgstr, ok
}

// has 判断原文是否在任一语言中有译文，源语言据此判断原文是否属于需要翻译的文案
func (c catalog) has(msgid string) bool {
	for _, entries := range c {
		if _, ok := entries[msgid]; ok {
			return true
		}
	}
	return false
}

// T 按 ctx 中的语言翻译原文 format 并以 args 格式化，没有译文时使用原文；用于后端直接生成的非错误文案。
// format 必须是字符串字面量，`make i18n` 才能提取到
func T(ctx context.Context, format string, args ...any) string {
	msg, _ := Translate(ctx, format)
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

// Translate 返回原文 msgid 在 ctx 语言下的译文（未格式化）；原文没有任何译文时返回原文与 false，
// 由调用方决定是否兜底（如错误出口对未翻译的存量文案沿用完整的错误信息）
func Translate(ctx context.Context, msgid string) (string, bool) {
	msgstr, ok := messages.translate(LangFromContext(ctx), msgid)
	if !ok {
		return msgid, false
	}
	return msgstr, true
}

// Msg 标记面向用户、需要翻译的原文（对应 gettext 的 N_()），`make i18n` 只提取被标记的原文：
//
//	bkerrs.On(model.ErrUnknownSwitch, bkerrs.ErrCodeInvalidArgument, i18n.Msg("unknown cfg switch key"))
//
// 定义为类型别名，i18n.Msg("...") 仍是字符串常量，go vet 可以照常检查作为格式串时的参数
type Msg = string
