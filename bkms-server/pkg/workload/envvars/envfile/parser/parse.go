package parser

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	pkgerrors "github.com/pkg/errors"

	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// maxEnvFileContentBytes 是 `.env` 文本的最大字节数上限（1 MiB）。
const maxEnvFileContentBytes = 1 << 20

// metadataDirectiveLineRegexp 匹配 `.env` 文件中的元数据注释行，格式为 `# field_name: value`。
// 当前支持的 field_name 有 desc（描述）、scopeType 和 scopeValue。
// 其中 scopeType 在 env file 语义上仅支持 workspace / envType。
var metadataDirectiveLineRegexp = regexp.MustCompile(`^#\s*([A-Za-z_][A-Za-z0-9_-]*)\s*:\s*(.*)$`)

// ErrInvalidEnvFileContent 表示 `.env` 文件内容本身不合法。
// 解析流程会使用 `pkg/errors` 对该错误做包装，并附带具体行号与原因；
// 上层可通过 `errors.Is(err, ErrInvalidEnvFileContent)` 将其识别为输入错误。
var ErrInvalidEnvFileContent = pkgerrors.New("invalid env file content")

// ParsedEnvVarRecord 表示从 `.env` 文本中解析出的一条环境变量记录。
// 元数据指令（# desc: / # scopeType: / # scopeValue:）只作用于紧随其后的下一条变量记录。
type ParsedEnvVarRecord struct {
	// Line 该条记录在原始内容中的行号，仅用于错误消息定位。
	Line int
	// Key 环境变量名。
	Key string
	// Value 环境变量值（已去除引号、转义并截断行内注释）。
	Value string
	// Description 从 `# desc:` 指令解析出的描述，未声明则为空。
	Description string
	// DeclaredScopeType 从 `# scopeType:` 指令解析出的原始 scopeType 声明值；
	// nil 表示输入中未显式声明该指令。
	DeclaredScopeType *string
	// DeclaredScopeValue 从 `# scopeValue:` 指令解析出的原始 scopeValue 声明值；
	// nil 表示输入中未显式声明该指令。
	DeclaredScopeValue *string
}

// ScopeTypeSpecified 返回输入中是否显式声明了 scopeType 元数据。
func (r ParsedEnvVarRecord) ScopeTypeSpecified() bool {
	return r.DeclaredScopeType != nil
}

// ScopeValueSpecified 返回输入中是否显式声明了 scopeValue 元数据。
func (r ParsedEnvVarRecord) ScopeValueSpecified() bool {
	return r.DeclaredScopeValue != nil
}

// pendingMetadataDirective 暂存一个待消费的元数据指令（desc / scopeType / scopeValue）
type pendingMetadataDirective struct {
	line  int
	value string
}

// pendingRecordMetadata 保存“等待下一条 KEY=VALUE 消费”的元数据指令。
type pendingRecordMetadata struct {
	desc       *pendingMetadataDirective
	scopeType  *pendingMetadataDirective
	scopeValue *pendingMetadataDirective
}

// ParseEnvFileRecords 扫描原始 `.env` 文本内容并返回解析后的记录列表。
// 遇到第一个校验错误（格式错误、key 非法、文件内 key 重复、不支持的元数据字段）
// 立即返回包装了 ErrInvalidEnvFileContent 的错误，包含行号和具体原因描述。
func ParseEnvFileRecords(content string) ([]ParsedEnvVarRecord, error) {
	if len([]byte(content)) > maxEnvFileContentBytes {
		return nil, pkgerrors.Wrapf(
			ErrInvalidEnvFileContent,
			"content must not exceed %d bytes",
			maxEnvFileContentBytes,
		)
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0), maxEnvFileContentBytes)

	var (
		records  []ParsedEnvVarRecord
		pending  pendingRecordMetadata
		seenKeys = make(map[string]int)
	)

	for lineNo := 1; scanner.Scan(); lineNo++ {
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" {
			continue
		}

		// 元数据注释和普通注释都在这里被消费；只有非注释行才继续按 KEY=VALUE 解析。
		if handled, err := collectMetadataDirective(trimmed, lineNo, &pending); err != nil {
			return nil, err
		} else if handled {
			continue
		}

		// 到这里说明当前行必须是一条真正的赋值记录。
		key, value, err := parseEnvAssignmentLine(trimmed)
		if err != nil {
			return nil, wrapLineError(lineNo, "%s", err.Error())
		}

		// 先做记录本身的基础合法性校验，再消费上一段注释积累的元数据。
		if err := validateParsedRecord(lineNo, key, value, seenKeys, pending); err != nil {
			return nil, err
		}

		record := ParsedEnvVarRecord{Line: lineNo, Key: key, Value: value}
		// 元数据只作用于紧随其后的这一条记录，消费后立即清空。
		pending.applyToRecord(&record)
		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, pkgerrors.Wrapf(ErrInvalidEnvFileContent, "failed to read env file content: %s", err.Error())
	}
	// 文件结尾若还残留未消费的元数据，说明用户写了悬空指令。
	if danglingLine := firstDanglingDirectiveLine(
		pending.desc,
		pending.scopeType,
		pending.scopeValue,
	); danglingLine > 0 {
		return nil, wrapLineError(danglingLine, "metadata directive must be followed by KEY=VALUE")
	}

	return records, nil
}

const (
	// parseModeNormal 读取未加引号的普通内容。
	parseModeNormal = iota
	// parseModeDoubleQuote 读取双引号包裹的内容，支持转义。
	parseModeDoubleQuote
	// parseModeSingleQuote 读取单引号包裹的内容，不处理转义。
	parseModeSingleQuote
	// parseModeEscape 处理双引号内的转义序列。
	parseModeEscape
)

// parseEnvAssignmentLine 解析一行已经过空白/注释分流后的 `KEY=VALUE` 赋值语句。
func parseEnvAssignmentLine(line string) (string, string, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", pkgerrors.New("invalid format, expected KEY=VALUE")
	}

	key := strings.TrimSpace(parts[0])
	// value 侧保留原始语法解析语义，不在这里做 key/value 约束校验。
	value, err := parseEnvValue(parts[1])
	if err != nil {
		return "", "", err
	}
	return key, value, nil
}

// parseEnvValue 解析 `.env` 中的 value 子集语法：
// 支持单/双引号、双引号转义以及非引号状态下的行内注释截断。
// 实现参考 `https://github.com/hashicorp/go-envparse/blob/2f9b4989/envparse.go#L180`
// 转义子集贴近 JSON string escapes：
// https://www.json.org/json-en.html
// 注释与引号的整体交互语义参考常见 dotenv 约定：
// https://dotenvx.com/docs/env-file.html
func parseEnvValue(raw string) (string, error) {
	value := bytes.TrimSpace([]byte(raw))
	if len(value) == 0 {
		return "", nil
	}

	newv := make([]byte, len(value))
	newi := 0
	lastSig := 0
	mode := parseModeNormal

	for i := 0; i < len(value); i++ {
		v := value[i]
		if v < 32 {
			return "", pkgerrors.Errorf("0x%0.2x is an invalid value character", v)
		}
		if v > 127 {
			// 多字节 UTF-8 字符在普通/引号模式下允许原样透传，但不能出现在转义序列中。
			if mode == parseModeEscape {
				return "", pkgerrors.New("multibyte characters disallowed in escape sequences")
			}
			lastSig = newi
			newv[newi] = v
			newi++
			continue
		}

		switch mode {
		case parseModeNormal:
			switch v {
			case '"':
				mode = parseModeDoubleQuote
			case '\'':
				mode = parseModeSingleQuote
			case '#':
				// 非引号态遇到 # 表示后续为行内注释，返回注释前最后一个有效字符为止。
				return string(newv[:lastSig]), nil
			case ' ', '\t':
				// 空白先保留，只有在后续遇到 # 或行尾时才根据 lastSig 决定是否裁掉尾随空白。
				newv[newi] = v
				newi++
			default:
				newv[newi] = v
				newi++
				lastSig = newi
			}
		case parseModeDoubleQuote:
			switch v {
			case '"':
				mode = parseModeNormal
			case '\\':
				mode = parseModeEscape
			default:
				newv[newi] = v
				newi++
				lastSig = newi
			}
		case parseModeEscape:
			switch v {
			case '"', '\\', '/':
				newv[newi] = v
			case 'b':
				newv[newi] = '\b'
			case 'f':
				newv[newi] = '\f'
			case 'r':
				newv[newi] = '\r'
			case 'n':
				newv[newi] = '\n'
			case 't':
				newv[newi] = '\t'
			case 'u':
				// \uXXXX 可能继续携带第二段 surrogate pair，因此会返回实际消费宽度。
				r, width, err := parseUnicodeEscape(value[i+1:])
				if err != nil {
					return "", err
				}
				i += width
				n := utf8.EncodeRune(newv[newi:], r)
				newi += n - 1
			default:
				return "", pkgerrors.Errorf("invalid escape sequence: %q", string(v))
			}
			newi++
			lastSig = newi
			mode = parseModeDoubleQuote
		case parseModeSingleQuote:
			switch v {
			case '\'':
				mode = parseModeNormal
			default:
				newv[newi] = v
				newi++
				lastSig = newi
			}
		}
	}

	switch mode {
	case parseModeNormal:
		// 正常结束时返回全部已写入内容；尾随空白在无注释场景下保留。
		return string(newv[:newi]), nil
	case parseModeDoubleQuote:
		return "", pkgerrors.New(`unmatched "`)
	case parseModeSingleQuote:
		return "", pkgerrors.New("unmatched '")
	case parseModeEscape:
		return "", pkgerrors.New("incomplete escape sequence")
	default:
		return "", pkgerrors.New("invalid parser state")
	}
}

// parseUnicodeEscape 解析 `\uXXXX`，并在需要时继续消费一对 surrogate pair。
// 实现参考 `https://github.com/hashicorp/go-envparse/blob/2f9b4989/envparse.go#L180`
func parseUnicodeEscape(buf []byte) (rune, int, error) {
	r, err := h2r(buf)
	if err != nil {
		return 0, 0, err
	}
	width := 4
	if utf16.IsSurrogate(r) {
		if len(buf) < 10 {
			return 0, 0, pkgerrors.New("incomplete Unicode surrogate pair")
		}
		if buf[4] != '\\' || buf[5] != 'u' {
			return 0, 0, pkgerrors.New("incomplete Unicode surrogate pair")
		}
		r2, err := h2r(buf[6:])
		if err != nil {
			return 0, 0, err
		}
		r = utf16.DecodeRune(r, r2)
		width += 6
	}
	return r, width, nil
}

// h2r 将 4 个十六进制字符转换为一个 rune。
// 该实现参考 `https://github.com/hashicorp/go-envparse/blob/2f9b4989/envparse.go#L394`
func h2r(buf []byte) (rune, error) {
	if len(buf) < 4 {
		return 0, pkgerrors.New("incomplete hex sequence")
	}
	var r rune
	for i := 0; i < 4; i++ {
		d := buf[i]
		switch {
		case '0' <= d && d <= '9':
			d = d - '0'
		case 'a' <= d && d <= 'f':
			d = d - 'a' + 10
		case 'A' <= d && d <= 'F':
			d = d - 'A' + 10
		default:
			return 0, pkgerrors.Errorf("invalid hex character: %q", string(d))
		}
		r *= 16
		r += rune(d)
	}
	return r, nil
}

// applyToRecord 将 pending 元数据写入当前记录，并清空等待队列。
func (p *pendingRecordMetadata) applyToRecord(record *ParsedEnvVarRecord) {
	if p.desc != nil {
		record.Description = p.desc.value
		p.desc = nil
	}
	if p.scopeType != nil {
		scopeType := p.scopeType.value
		record.DeclaredScopeType = &scopeType
		p.scopeType = nil
	}
	if p.scopeValue != nil {
		scopeValue := p.scopeValue.value
		record.DeclaredScopeValue = &scopeValue
		p.scopeValue = nil
	}
}

// collectMetadataDirective 识别并缓存注释行中的 BKMS 扩展元数据。
// 返回 handled=true 表示当前行已作为注释消费，无需继续按赋值行处理。
func collectMetadataDirective(trimmed string, lineNo int, pending *pendingRecordMetadata) (bool, error) {
	if !strings.HasPrefix(trimmed, "#") {
		return false, nil
	}

	matches := metadataDirectiveLineRegexp.FindStringSubmatch(trimmed)
	if len(matches) == 0 {
		// 普通注释不参与语义解析，但仍视为已消费，避免再走赋值分支。
		return true, nil
	}

	fieldName := strings.ToLower(matches[1])
	fieldValue := strings.TrimSpace(matches[2])
	switch fieldName {
	case "desc":
		desc, err := parseMetadataDirectiveValue(fieldValue)
		if err != nil {
			return true, wrapLineError(lineNo, "invalid desc metadata: %s", err.Error())
		}
		pending.desc = &pendingMetadataDirective{line: lineNo, value: desc}
	case "scopetype":
		switch fieldValue {
		case string(envvartypes.ScopeTypeWorkspace), string(envvartypes.ScopeTypeEnvType):
		default:
			return true, wrapLineError(lineNo, `scopeType %q is not supported`, fieldValue)
		}
		pending.scopeType = &pendingMetadataDirective{line: lineNo, value: fieldValue}
	case "scopevalue":
		pending.scopeValue = &pendingMetadataDirective{line: lineNo, value: fieldValue}
	default:
		// 对形如 `# foo: bar` 的未知字段直接报错，避免用户误以为系统会忽略它。
		return true, wrapLineError(lineNo, `unsupported metadata directive %q`, matches[1])
	}
	return true, nil
}

func parseMetadataDirectiveValue(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, `"`) || strings.HasPrefix(trimmed, `'`) {
		return parseEnvValue(trimmed)
	}
	return trimmed, nil
}

// validateParsedRecord 校验解析出的单条变量记录及其与上下文相关的约束。
func validateParsedRecord(
	lineNo int,
	key string,
	value string,
	seenKeys map[string]int,
	pending pendingRecordMetadata,
) error {
	if err := envvartypes.ValidateEnvVarKey(key); err != nil {
		return wrapLineError(lineNo, "%s", err.Error())
	}
	if err := envvartypes.ValidateEnvVarValue(value); err != nil {
		return wrapLineError(lineNo, `env var value for key %q: %s`, key, err.Error())
	}
	if firstLine, dup := seenKeys[key]; dup {
		return wrapLineError(lineNo, `duplicate key %q (first declared at line %d)`, key, firstLine)
	}
	if pending.scopeValue != nil && pending.scopeType == nil {
		// scopeValue 语义依赖 scopeType，因此必须在真正落地到记录前就拒绝。
		return wrapLineError(lineNo, "scopeValue requires scopeType")
	}

	// 只有确认当前记录合法后，才把 key 标记为已出现。
	seenKeys[key] = lineNo
	return nil
}

// wrapLineError 为 parser 错误统一补充行号和 ErrInvalidEnvFileContent 哨兵。
func wrapLineError(lineNo int, format string, args ...any) error {
	formatArgs := append([]any{lineNo}, args...)
	return pkgerrors.Wrapf(ErrInvalidEnvFileContent, "line %d: "+format, formatArgs...)
}

// firstDanglingDirectiveLine 检测是否有悬空（未被消费）的元数据指令，并返回其中最早出现的行号
func firstDanglingDirectiveLine(directives ...*pendingMetadataDirective) int {
	firstLine := 0
	for _, directive := range directives {
		if directive == nil {
			continue
		}
		if firstLine == 0 || directive.line < firstLine {
			firstLine = directive.line
		}
	}
	return firstLine
}
