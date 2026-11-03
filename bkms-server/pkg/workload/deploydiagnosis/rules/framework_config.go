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

package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/file"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

var (
	// 命中「打开某路径失败」，并抽出日志里的读取路径
	openConfigPathRE = regexp.MustCompile(`open\s+(\S+):\s+no such file or directory`)
	// 行内必须有文件缺失的字样，避免语法错误、权限或超时被判成这条规则
	configReadFailRE = regexp.MustCompile(`(?i)no such file or directory|file not found`)
	// 未配置框架配置文件名时，靠常见扩展名认出配置文件
	configFileExts = map[string]struct{}{
		".yaml": {},
		".yml":  {},
		".json": {},
		".conf": {},
	}
)

// FrameworkConfigRule 框架配置文件读取失败：日志里出现读不到配置文件的报错
type FrameworkConfigRule struct{}

// ID 返回 IDFrameworkConfigNotFound
func (FrameworkConfigRule) ID() string { return IDFrameworkConfigNotFound }

// Diagnose 只对 tRPC / TAF 应用生效；处理步骤按日志里读取路径的形态区分
func (r FrameworkConfigRule) Diagnose(snap *Snapshot, inst *Instance) *Group {
	if !bkmsapp.IsAppModelType(snap.AppType) {
		return nil
	}
	evidence, logPath := r.findEvidence(inst.LogLines, snap.FrameworkConfig.FileName)
	if len(evidence) == 0 {
		return nil
	}

	configs, mismatch := r.buildRelatedConfigs(snap.FrameworkConfig, logPath)
	s := frameworkConfigSpec.withSolutions(frameworkConfigSolutions.unknownPath)
	// 相对路径也会被判为不一致，但根因是工作目录，先认它才能给出对的处理步骤
	switch {
	case logPath != "" && !filepath.IsAbs(logPath):
		s = s.withSolutions(frameworkConfigSolutions.relativePath)
	case mismatch:
		s = s.withSolutions(frameworkConfigSolutions.pathMismatch)
	}
	return newInstanceGroup(s, snap, inst, evidence, configs)
}

// findEvidence 返回最多 MaxEvidenceLines 条报错日志，以及第一条能抽出的读取路径（抽不到为空）
func (r FrameworkConfigRule) findEvidence(logLines []string, configuredName string) ([]string, string) {
	evidence := make([]string, 0, MaxEvidenceLines)
	logPath := ""
	for _, line := range logLines {
		if !r.lineReportsMissingFile(line, configuredName) {
			continue
		}
		if matches := openConfigPathRE.FindStringSubmatch(line); len(matches) == 2 && logPath == "" {
			logPath = matches[1]
		}
		evidence = append(evidence, line)
		if len(evidence) >= MaxEvidenceLines {
			break
		}
	}
	return evidence, logPath
}

// lineReportsMissingFile 行内要有文件缺失；已知配置文件名时只认这个文件
func (r FrameworkConfigRule) lineReportsMissingFile(line, configuredName string) bool {
	if !configReadFailRE.MatchString(line) {
		return false
	}
	if matches := openConfigPathRE.FindStringSubmatch(line); len(matches) == 2 {
		return r.isConfigPath(matches[1], configuredName)
	}
	return configuredName != "" && strings.Contains(line, configuredName)
}

// isConfigPath 已知文件名时只认该文件，否则才用常见配置扩展名兜底
func (FrameworkConfigRule) isConfigPath(path, configuredName string) bool {
	base := filepath.Base(path)
	if configuredName != "" {
		return base == configuredName
	}
	_, ok := configFileExts[strings.ToLower(filepath.Ext(base))]
	return ok
}

// buildRelatedConfigs 生成相关配置表格，第二个返回值表示日志路径与配置路径是否不一致
// 只有配置路径是绝对路径时才比对；日志是相对路径时视为不一致，因为单凭字符串无法确认指向同一文件
func (FrameworkConfigRule) buildRelatedConfigs(fc FrameworkConfig, logPath string) ([]ConfigRow, bool) {
	configured := filepath.Join(fc.FilePath, fc.FileName)
	mismatch := logPath != "" && filepath.IsAbs(configured) && !file.IsSameAbsPath(logPath, configured)
	hint := ""
	if mismatch {
		hint = hintPathMismatch
	}
	return []ConfigRow{
		{Label: labelFrameworkConfigFile, Value: fc.FileName},
		{Label: labelFrameworkConfigPath, Value: fc.FilePath},
		{Label: labelLogReadPath, Value: logPath, Danger: mismatch, Hint: hint},
		{Label: labelStartCommand, Value: fc.Command},
	}, mismatch
}
