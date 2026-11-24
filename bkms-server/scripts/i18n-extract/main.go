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

// i18n-extract 扫描源码中需要翻译的英文原文，把各语言 .po 文件中缺失的原文以待翻译条目（msgstr 为空）追加到文件末尾，
// 并列出 .po 中已不再被源码引用的条目，由开发者确认后删除。
//
// 用法（在 bkms-server 目录下执行）：
//
//	make i18n
//
// 留空的条目运行时回退到英文原文，不影响服务启动；只追加不改写，不会打乱文件中已有的内容与注释
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chai2010/gettext-go/po"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n/extract"
)

const localesDir = "pkg/common/i18n/locales"

var sourceRoots = []string{"pkg", "cmd"}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "i18n-extract:", err)
		os.Exit(1)
	}
}

func run() error {
	msgs, err := extract.Messages(".", sourceRoots...)
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(localesDir, "*.po"))
	if err != nil {
		return errors.Wrap(err, "glob locale files")
	}
	for _, path := range paths {
		if err = update(path, msgs); err != nil {
			return err
		}
	}
	return nil
}

// update 向 path 追加缺失的原文，并打印不再被引用的条目
func update(path string, msgs []extract.Message) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "read %s", path)
	}
	file, err := po.Load(data)
	if err != nil {
		return errors.Wrapf(err, "parse %s", path)
	}

	existing := map[string]bool{}
	for _, m := range file.Messages {
		existing[m.MsgId] = true
	}
	referenced := map[string]bool{}
	var sb strings.Builder
	added := 0
	for _, m := range msgs {
		referenced[m.ID] = true
		if existing[m.ID] {
			continue
		}
		fmt.Fprintf(&sb, "\n#: %s\nmsgid %s\nmsgstr \"\"\n", m.Pos, strconv.Quote(m.ID))
		added++
	}
	for _, m := range file.Messages {
		if !referenced[m.MsgId] {
			fmt.Printf("%s: msgid %q is no longer referenced in source, remove it\n", path, m.MsgId)
		}
	}
	if added == 0 {
		fmt.Printf("%s: no untranslated messages to add\n", path)
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return errors.Wrapf(err, "open %s", path)
	}
	defer f.Close()
	if _, err = f.WriteString(sb.String()); err != nil {
		return errors.Wrapf(err, "write %s", path)
	}
	fmt.Printf("%s: added %d untranslated messages, fill in msgstr to translate them\n", path, added)
	return nil
}
