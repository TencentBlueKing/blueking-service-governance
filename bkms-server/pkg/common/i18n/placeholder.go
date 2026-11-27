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
	"maps"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

// checkPlaceholders 校验译文与原文引用的参数一致：同一位置的参数使用相同的 printf 动词。
// 译文可以用 %[n]d 这类显式下标调整参数顺序
func checkPlaceholders(msgid, msgstr string) error {
	want, err := placeholders(msgid)
	if err != nil {
		return errors.Wrap(err, "parse msgid")
	}
	got, err := placeholders(msgstr)
	if err != nil {
		return errors.Wrap(err, "parse msgstr")
	}
	if !maps.Equal(want, got) {
		return errors.Errorf("msgstr placeholders %v do not match msgid placeholders %v", got, want)
	}
	return nil
}

// placeholders 解析 printf 格式串，返回参数下标（从 1 开始）到动词的映射；宽度与精度中的 * 记为动词 '*'。
// 同一参数可以重复引用，但必须使用相同的动词，否则运行时格式化会出错
func placeholders(format string) (map[int]rune, error) {
	res := map[int]rune{}
	arg := 1
	use := func(verb rune) error {
		if prev, ok := res[arg]; ok && prev != verb {
			return errors.Errorf("argument %d is referenced by both %%%c and %%%c", arg, prev, verb)
		}
		res[arg] = verb
		arg++
		return nil
	}
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i < len(format) && format[i] == '%' {
			continue
		}
		for i < len(format) && strings.IndexByte("+-# 0", format[i]) >= 0 {
			i++
		}

		// argIndex 解析 [n] 显式下标，后续参数从该下标继续编号
		argIndex := func() error {
			if i >= len(format) || format[i] != '[' {
				return nil
			}
			end := strings.IndexByte(format[i:], ']')
			if end < 0 {
				return errors.Errorf("unclosed argument index at %d", i)
			}
			n, err := strconv.Atoi(format[i+1 : i+end])
			if err != nil || n < 1 {
				return errors.Errorf("invalid argument index %q", format[i:i+end+1])
			}
			arg = n
			i += end + 1
			return nil
		}
		// number 解析宽度或精度：* 占用一个参数，否则跳过数字
		number := func() error {
			if err := argIndex(); err != nil {
				return err
			}
			if i < len(format) && format[i] == '*' {
				i++
				return use('*')
			}
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
				i++
			}
			return nil
		}

		if err := number(); err != nil {
			return nil, err
		}
		if i < len(format) && format[i] == '.' {
			i++
			if err := number(); err != nil {
				return nil, err
			}
		}
		if err := argIndex(); err != nil {
			return nil, err
		}
		if i >= len(format) {
			return nil, errors.New("missing verb at end of format")
		}
		verb, size := utf8.DecodeRuneInString(format[i:])
		if err := use(verb); err != nil {
			return nil, err
		}
		i += size - 1
	}
	return res, nil
}
