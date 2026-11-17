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

// i18n-extract 扫描源码中需要翻译的 key（bkerrs.Define 的详情码、i18n.T 的字面量 key），
// 把 locales 中缺失的 key 以待翻译骨架追加到所属模块的 YAML 文件。
//
// 用法（在 bkms-server 目录下执行）：
//
//	make i18n
//
// 目标文件取同目录（含子目录）已翻译 key 最多落在的文件，没有时逐级向上查找到模块目录，都找不到时写入 misc.yaml。
// 骨架的英文预填 Define 的默认描述，中文留空；i18n 加载时拒绝空文案，补全翻译之前服务无法启动、测试也会失败。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n/extract"
)

const (
	localesDir   = "pkg/common/i18n/locales"
	fallbackFile = "misc.yaml"
)

var sourceRoots = []string{"pkg", "cmd"}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "i18n-extract:", err)
		os.Exit(1)
	}
}

// run 找出源码引用了、但 locale 文件中没有的 key，按所属模块分组追加骨架并打印位置
func run() error {
	// 1. 收集源码引用的全部 key，以及 locale 中已有的 key 各自所在的文件
	keys, err := extract.Keys(sourceRoots...)
	if err != nil {
		return err
	}
	keyFiles, err := loadKeyFiles()
	if err != nil {
		return err
	}

	// 2. 一次遍历同时完成两件事：已翻译的 key 用来统计各目录的翻译习惯落在哪个文件，未翻译的 key 记为待补全
	//    dirFiles: 目录 -> 文件 -> 该目录及其子目录中已翻译 key 落在此文件的数量（不统计到源码根目录，避免跨模块）
	dirFiles := map[string]map[string]int{}
	var missing []extract.Key
	for _, k := range keys {
		file, ok := keyFiles[k.Name]
		if !ok {
			missing = append(missing, k)
			continue
		}
		// key 计入其所在目录及每一级祖先目录，使只有子目录有翻译的父目录也能推断出目标文件
		for _, d := range moduleDirs(k.Dir) {
			if dirFiles[d] == nil {
				dirFiles[d] = map[string]int{}
			}
			dirFiles[d][file]++
		}
	}
	if len(missing) == 0 {
		fmt.Println("all keys have translations")
		return nil
	}

	// 3. 为每个缺失的 key 推断目标文件并分组，同一文件只打开写入一次
	byFile := map[string][]extract.Key{}
	for _, k := range missing {
		file := targetFile(dirFiles, k.Dir)
		byFile[file] = append(byFile[file], k)
	}
	// map 遍历顺序随机，按文件名排序使多次运行的写入顺序与输出一致
	files := make([]string, 0, len(byFile))
	for file := range byFile {
		files = append(files, file)
	}
	sort.Strings(files)

	// 4. 逐个文件追加骨架，并打印 文件:行号，便于在 IDE 中直接跳转补全
	fmt.Printf(
		"added %d keys, fill in zh/en at the locations below before running the server or tests:\n",
		len(missing),
	)
	for _, file := range files {
		path := filepath.Join(localesDir, file)
		lines, err := appendSkeleton(path, byFile[file])
		if err != nil {
			return err
		}
		for i, k := range byFile[file] {
			fmt.Printf("  %s:%d  %s  (used at %s)\n", path, lines[i], k.Name, k.Pos)
		}
	}
	return nil
}

// loadKeyFiles 返回 key -> 所在的 locale 文件名；只解析 YAML 结构，不经过 i18n 的模板与完整性校验
func loadKeyFiles() (map[string]string, error) {
	paths, err := filepath.Glob(filepath.Join(localesDir, "*.yaml"))
	if err != nil {
		return nil, errors.Wrap(err, "glob locale files")
	}
	keyFiles := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "read %s", path)
		}
		var entries map[string]any
		if err = yaml.Unmarshal(data, &entries); err != nil {
			return nil, errors.Wrapf(err, "parse %s", path)
		}
		for key := range entries {
			keyFiles[key] = filepath.Base(path)
		}
	}
	return keyFiles, nil
}

// moduleDirs 返回 dir 及其祖先目录，到源码根目录（不含）为止
func moduleDirs(dir string) []string {
	var dirs []string
	for d := dir; d != "." && d != string(filepath.Separator) && !slices.Contains(sourceRoots, d); d = filepath.Dir(d) {
		dirs = append(dirs, d)
	}
	return dirs
}

// targetFile 从 dir 开始逐级向上，取已翻译 key 最多落在的文件；数量相同时取文件名较小者，保证结果稳定
func targetFile(dirFiles map[string]map[string]int, dir string) string {
	// moduleDirs 由近及远返回目录，越近的目录越能代表 key 所属模块，因此命中即返回
	for _, d := range moduleDirs(dir) {
		best, bestCount := "", 0
		for file, count := range dirFiles[d] {
			if count > bestCount || (count == bestCount && file < best) {
				best, bestCount = file, count
			}
		}
		if best != "" {
			return best
		}
	}
	return fallbackFile
}

// appendSkeleton 在文件末尾追加待翻译骨架，返回每个 key 所在的行号
func appendSkeleton(path string, keys []extract.Key) ([]int, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "read %s", path)
	}

	// line 记录已写内容的最后一行行号：以换行符结尾的文件，换行符个数即行数；
	// 不以换行符结尾时先补一个换行，避免骨架接在最后一行后面
	var sb strings.Builder
	line := strings.Count(string(data), "\n")
	if len(data) > 0 && data[len(data)-1] != '\n' {
		sb.WriteString("\n")
		line++
	}

	// 每个 key 占 5 行：空行、TODO 注释、key、zh、en，key 位于第 3 行
	lines := make([]int, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, line+3)
		fmt.Fprintf(&sb, "\n# TODO(make i18n): 补全翻译后删除本行，引用位置 %s%s\n", sourceLine(k.Pos), paramsHint(k))
		fmt.Fprintf(&sb, "%s:\n  zh: \"\"\n  en: %s\n", k.Name, strconv.Quote(k.Default))
		line += 5
	}

	// 只追加不改写，不会打乱文件中已有的内容与注释
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, errors.Wrapf(err, "open %s", path)
	}
	defer f.Close()
	if _, err = f.WriteString(sb.String()); err != nil {
		return nil, errors.Wrapf(err, "write %s", path)
	}
	return lines, nil
}

// paramsHint 列出调用处传入的模板参数，提示以 {{.name}} 引用
func paramsHint(k extract.Key) string {
	names := make([]string, 0, len(k.Params))
	for _, p := range k.Params {
		names = append(names, "{{."+p+"}}")
	}
	// 参数无法完全静态解析时仍列出已知部分，并提示去源码确认
	switch {
	case k.ParamsUnknown:
		names = append(names, "部分参数无法静态解析，请查看源码")
	case len(names) == 0:
		return ""
	}
	return "，可用参数 " + strings.Join(names, " ")
}

// sourceLine 去掉 file:line:col 中的列号
func sourceLine(pos string) string {
	if i := strings.LastIndex(pos, ":"); i > 0 {
		return pos[:i]
	}
	return pos
}
