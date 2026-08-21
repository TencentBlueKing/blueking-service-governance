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

// 本模块主要输出类似 kubectl 命令的极简表格样式。

package output

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/stringx"
)

// 列间距。表格不绘制任何边框，仅靠列右侧的空白区分列，与 kubectl 的输出保持一致
const columnGap = 3

// 单元格样式。Lip Gloss 默认不带任何 padding，列间距需要显式设置
var cellStyle = lipgloss.NewStyle().PaddingRight(columnGap)

// tableFormatter 将数据输出为表格字符串。
type tableFormatter struct{}

// tableField 表格列的元信息
type tableField struct {
	// 结构体字段索引
	index int
	// 表头名称
	header string
}

func (f tableFormatter) Format(_ context.Context, data any) (string, error) {
	if data == nil {
		return "null", nil
	}

	// 传入的数据可能是包含多个元素的切片，也可能是单个元素。
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		if v.Len() == 0 {
			return "null", nil
		}
		items := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			items[i] = v.Index(i).Interface()
		}
		return f.asTable(items), nil
	}
	return f.asTable([]any{data}), nil
}

// asTable 将任意列表序列化为表格字符串，一些特性行为：
//
// - 字段名优先取 json tag 并被转为 FIELD NAME 样式；
// - 带有 `table:"-"` tag 的字段会被跳过。
func (f tableFormatter) asTable(items []any) string {
	if len(items) == 0 {
		return "null"
	}

	// 解析字段元信息（基于第一个元素的类型）
	fields := f.parseTableFields(reflect.TypeOf(items[0]))
	if len(fields) == 0 {
		return "null"
	}

	// 提取表头
	headers := lo.Map(fields, func(item tableField, _ int) string { return item.header })

	// 提取行数据
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		v := reflect.ValueOf(item)
		for v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		row := make([]string, len(fields))
		for i, field := range fields {
			fieldVal := v.Field(field.index)
			row[i] = f.formatFieldValue(fieldVal)
		}
		rows = append(rows, row)
	}

	return renderTable(headers, rows)
}

// parseTableFields 解析结构体类型，返回需要在表格中展示的字段列表
func (f tableFormatter) parseTableFields(t reflect.Type) []tableField {
	// 解引用指针类型
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	fields := make([]tableField, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Tag.Get("table") == "-" {
			continue
		}
		// 优先使用 json tag 作为表头（更贴近 API 语义），fallback 到字段名
		name := sf.Name
		if jsonTag := sf.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			if tagName, _, _ := strings.Cut(jsonTag, ","); tagName != "" {
				name = tagName
			}
		}
		fields = append(fields, tableField{index: i, header: toHeader(name)})
	}
	return fields
}

// formatFieldValue 格式化字段值用于表格展示
func (f tableFormatter) formatFieldValue(v reflect.Value) string {
	// 解引用指针
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	// 对 slice/array 类型的字段，将每个元素用逗号分隔，使单元格内容保持在一行
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		if v.Len() == 0 {
			return ""
		}
		elems := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			elems[i] = fmt.Sprintf("%v", v.Index(i).Interface())
		}
		return strings.Join(elems, ", ")
	}

	return fmt.Sprintf("%v", v.Interface())
}

// toHeader 将字段名（或 json tag）转换为表头文本，如 appName -> APP NAME
func toHeader(name string) string {
	return strings.ToUpper(stringx.ToWords(name))
}

// renderTable 以 kubectl 风格渲染表格：不绘制边框与行列分隔线，列之间靠空白对齐。
func renderTable(headers []string, rows [][]string) string {
	t := table.New().
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderHeader(false).
		BorderColumn(false).
		BorderRow(false).
		StyleFunc(func(_, _ int) lipgloss.Style { return cellStyle }).
		Headers(headers...).
		Rows(rows...)

	return trimTrailingSpaces(t.Render())
}

// trimTrailingSpaces 去掉每行末尾的空白。最后一列的单元格同样会被补齐到列宽，
// 直接输出会留下无意义的尾部空格，重定向到文件时尤为明显。
func trimTrailingSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
