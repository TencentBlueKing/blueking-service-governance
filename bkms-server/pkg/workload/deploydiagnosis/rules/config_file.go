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

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
)

var (
	// 命中「打开某路径失败」，并抽出日志里的读取路径
	openConfigPathRE = regexp.MustCompile(`open\s+(\S+):\s+no such file or directory`)
	// 行内必须同时出现配置文件名和文件缺失，避免语法错误、权限或超时被判成这条规则
	configReadFailRE = regexp.MustCompile(`(?i)no such file or directory|file not found`)
	configFileExts   = map[string]struct{}{
		".yaml": {},
		".yml":  {},
		".json": {},
		".conf": {},
	}
)

type configFileHit struct {
	lines   []string
	logPath string
}

func (h configFileHit) evidenceLines() []string { return h.lines }

type configFileRule struct{}

func (configFileRule) ID() string { return IDConfigFileNotFound }

func (r configFileRule) Match(snap *Snapshot, inst *Instance) (Hit, bool) {
	if !bkmsapp.IsAppModelType(snap.AppType) {
		return nil, false
	}
	lines, logPath := r.findEvidence(inst.LogLines, snap.Framework.FileName)
	if len(lines) == 0 {
		return nil, false
	}
	return configFileHit{lines: lines, logPath: logPath}, true
}

func (r configFileRule) Render(snap *Snapshot, inst *Instance, hit Hit) *serializer.DiagnosisGroupObj {
	found := hit.(configFileHit)
	return buildGroup(r.spec(), snap, inst, found.lines, r.relatedConfigs(snap.Framework, found.logPath))
}

func (configFileRule) spec() spec {
	return spec{
		id:          IDConfigFileNotFound,
		name:        "框架配置文件读取失败",
		category:    "配置下发",
		confidence:  "high",
		cause:       "框架读不到配置文件，容器启动即退出",
		causeDetail: "平台会把框架配置文件按你设置的路径写入容器。日志里的读取路径必须与「应用配置 → 框架配置文件」一致，否则进程会在启动时退出。",
		solutions: []string{
			"对比下方日志里的读取路径与「应用配置 → 框架配置文件」中的配置文件路径，两者必须完全一致",
			"代码使用相对路径读取时，需与启动命令的工作目录匹配，建议统一改为绝对路径",
		},
		actions: []serializer.DiagnosisActionObj{
			{ID: serializer.ActionFrameworkConfig, Label: "前往框架配置文件"},
			{ID: serializer.ActionRedeploy, Label: "重新部署"},
		},
	}
}

func (r configFileRule) findEvidence(logLines []string, configuredName string) ([]string, string) {
	evidence := make([]string, 0, MaxEvidenceLines)
	logPath := ""
	for _, line := range logLines {
		if !r.lineReportsMissingFile(line, configuredName) {
			continue
		}
		if matches := openConfigPathRE.FindStringSubmatch(line); len(matches) == 2 {
			if logPath == "" {
				logPath = matches[1]
			}
		}
		evidence = append(evidence, line)
		if len(evidence) >= MaxEvidenceLines {
			break
		}
	}
	return evidence, logPath
}

// lineReportsMissingFile 已知配置文件名时只认这个文件，并且行内要有文件缺失
func (r configFileRule) lineReportsMissingFile(line, configuredName string) bool {
	if !configReadFailRE.MatchString(line) {
		return false
	}
	if matches := openConfigPathRE.FindStringSubmatch(line); len(matches) == 2 {
		return r.isConfigPath(matches[1], configuredName)
	}
	return configuredName != "" && strings.Contains(line, configuredName)
}

// isConfigPath 已知文件名时只认该文件，否则才用常见配置扩展名兜底
func (configFileRule) isConfigPath(path, configuredName string) bool {
	base := filepath.Base(path)
	if configuredName != "" {
		return base == configuredName
	}
	_, ok := configFileExts[strings.ToLower(filepath.Ext(base))]
	return ok
}

func (r configFileRule) relatedConfigs(fw Framework, logPath string) []*serializer.RelatedConfigRowObj {
	configured := r.joinPath(fw.FilePath, fw.FileName)
	// 只有两边都是绝对路径且不完全一致才标红；相对路径对不上工作目录，不能当成同一路径
	danger := logPath != "" && strings.HasPrefix(configured, "/") && !r.samePath(logPath, configured)
	hint := ""
	if danger {
		hint = PathMismatchHint
	}
	return []*serializer.RelatedConfigRowObj{
		{Label: "框架配置文件", Value: fw.FileName},
		{Label: "配置文件路径", Value: fw.FilePath},
		{Label: "日志中的读取路径", Value: logPath, Danger: danger, Hint: hint},
		{Label: "启动命令", Value: fw.Command},
	}
}

func (configFileRule) joinPath(dir, name string) string {
	if dir == "" {
		return name
	}
	if name == "" {
		return dir
	}
	return filepath.Join(dir, name)
}

func (configFileRule) samePath(logPath, configured string) bool {
	if !strings.HasPrefix(logPath, "/") || !strings.HasPrefix(configured, "/") {
		return false
	}
	normalize := func(p string) string { return filepath.Clean(strings.TrimRight(p, "/")) }
	return normalize(logPath) == normalize(configured)
}
