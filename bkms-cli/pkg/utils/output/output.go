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

// Package output 提供输出格式定义和工具函数
package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/itchyny/gojq"
	tw "github.com/olekukonko/tablewriter"
	twtypes "github.com/olekukonko/tablewriter/tw"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Format 定义支持的数据序列化输出格式
type Format string

const (
	// FormatDefault 默认输出格式
	FormatDefault Format = ""
	// FormatJson JSON 输出格式
	FormatJson Format = "json"
	// FormatYaml YAML 输出格式
	FormatYaml Format = "yaml"
	// FormatTable 表格输出格式
	FormatTable Format = "table"

	// FormatJq jq 表达式输出格式，格式为 jq=<...>
	FormatJq Format = "jq"
)

const (
	// FlagUsage 通用 --output 参数说明
	FlagUsage = "output format: json, yaml, table, jq=<expr>"
)

type formatKind string

const (
	// 简单格式，比如 YAML、Json 等
	simpleFormat formatKind = "simple"
	// 可自定义的格式，比如 jq=<...> 等
	customizableFormat formatKind = "customizable"
)

type parsedFormat struct {
	kind       formatKind
	formatType Format
	value      string
}

type formatter interface {
	Format(ctx context.Context, data any) (string, error)
}

// FormatData 智能格式化数据，格式缺省时，将根据数据类型自动选择表格或 JSON 格式。
func FormatData(ctx context.Context, data any, format string) (string, error) {
	parsed, err := parseFormat(format)
	if err != nil {
		return "", err
	}

	selectedFormatter, err := resolveFormatter(data, parsed)
	if err != nil {
		return "", err
	}
	return selectedFormatter.Format(ctx, data)
}

// parseFormat 解析用户提供的 format 字符串并输出解析后的结果，如果格式非法时会报错。
func parseFormat(format string) (parsedFormat, error) {
	format = strings.TrimSpace(format)

	if name, value, ok := strings.Cut(format, "="); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			return parsedFormat{}, errors.Errorf("output format type cannot be empty")
		}
		if Format(name) != FormatJq {
			return parsedFormat{}, errors.Errorf("unsupported customizable output format: %s", name)
		}
		return parsedFormat{
			kind:       customizableFormat,
			formatType: Format(name),
			value:      value,
		}, nil
	}

	switch Format(format) {
	case FormatDefault, FormatJson, FormatYaml, FormatTable:
		return parsedFormat{kind: simpleFormat, formatType: Format(format)}, nil
	default:
		return parsedFormat{}, errors.Errorf("unsupported output format: %s", format)
	}
}

// 根据解析后的 format 对象构建对应的 formatter 对象。
func resolveFormatter(data any, format parsedFormat) (formatter, error) {
	switch format.kind {
	case simpleFormat:
		switch format.formatType {
		case FormatJson:
			return jsonFormatter{}, nil
		case FormatYaml:
			return yamlFormatter{}, nil
		case FormatTable:
			return tableFormatter{}, nil
		case FormatDefault:
			// TODO: 根据数据类型自动在两种输出格式之间切换是否合理？是否应该调整？
			return inferDefaultFormatter(data), nil
		default:
			return nil, errors.Errorf("unsupported output format: %s", format.formatType)
		}
	case customizableFormat:
		switch format.formatType {
		case FormatJq:
			return jqFormatter{expr: format.value}, nil
		default:
			return nil, errors.Errorf("unsupported customizable output format: %s", format.formatType)
		}
	default:
		return nil, errors.Errorf("unsupported output format kind: %s", format.kind)
	}
}

// inferDefaultFormatter 在用户未指定 --output 时选择默认 formatter。
// 列表默认使用表格展示，单个对象默认使用 YAML 展示，以保持原有 CLI 输出习惯。
func inferDefaultFormatter(data any) formatter {
	if data == nil {
		return yamlFormatter{}
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return tableFormatter{}
	}
	return yamlFormatter{}
}

// jsonFormatter 将数据输出为 JSON 字符串。
type jsonFormatter struct{}

func (f jsonFormatter) Format(_ context.Context, data any) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "marshal json output")
	}
	return string(jsonBytes), nil
}

// yamlFormatter 将数据输出为 YAML 字符串。
type yamlFormatter struct{}

func (f yamlFormatter) Format(_ context.Context, data any) (string, error) {
	if data == nil {
		return "null", nil
	}
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "marshal yaml output")
	}
	return string(yamlBytes), nil
}

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

// asTable 将任意列表序列化为表格字符串
// 带有 `table:"-"` tag 的字段会被跳过，不在表格中展示
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
	headers := make([]string, len(fields))
	for i, field := range fields {
		headers[i] = field.header
	}

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

	builder := new(strings.Builder)
	table := tw.NewTable(builder, tw.WithRendition(twtypes.Rendition{
		Settings: twtypes.Settings{
			Separators: twtypes.Separators{
				BetweenRows: twtypes.On,
			},
		},
	}))
	table.Header(headers)
	_ = table.Bulk(rows)
	_ = table.Render()
	return builder.String()
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
		header := sf.Name
		if jsonTag := sf.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			if name, _, _ := strings.Cut(jsonTag, ","); name != "" {
				header = name
			}
		}
		fields = append(fields, tableField{index: i, header: header})
	}
	return fields
}

// formatFieldValue 格式化字段值用于表格展示
// 对 slice/array 类型的字段，将每个元素用换行符分隔，使表格中多行展示
func (f tableFormatter) formatFieldValue(v reflect.Value) string {
	// 解引用指针
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		if v.Len() == 0 {
			return ""
		}
		lines := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			lines[i] = fmt.Sprintf("%v", v.Index(i).Interface())
		}
		return strings.Join(lines, "\n")
	}

	return fmt.Sprintf("%v", v.Interface())
}

// jqFormatter 使用 jq 表达式从数据中提取并格式化输出。
type jqFormatter struct {
	expr string
}

func (f jqFormatter) Format(ctx context.Context, data any) (string, error) {
	if strings.TrimSpace(f.expr) == "" {
		return "", errors.Errorf("jq expression cannot be empty")
	}
	input, err := f.toInput(data)
	if err != nil {
		return "", errors.Wrap(err, "prepare jq input")
	}

	query, err := gojq.Parse(f.expr)
	if err != nil {
		return "", errors.Wrap(err, "parse jq expression")
	}

	iter := query.RunWithContext(ctx, input)
	outputs := make([]string, 0)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if haltErr, ok := err.(*gojq.HaltError); ok && haltErr.Value() == nil {
				break
			}
			return "", errors.Wrap(err, "run jq expression")
		}

		rendered, err := f.renderValue(v)
		if err != nil {
			return "", errors.Wrap(err, "render jq output")
		}
		outputs = append(outputs, rendered)
	}
	return strings.Join(outputs, "\n"), nil
}

// toInput converts custom Go structs to JSON-compatible values required by gojq.
func (f jqFormatter) toInput(data any) (any, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.UseNumber()
	var input any
	if err := decoder.Decode(&input); err != nil {
		return nil, err
	}
	return input, nil
}

func (f jqFormatter) renderValue(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}

	renderedBytes, err := gojq.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(renderedBytes), nil
}
