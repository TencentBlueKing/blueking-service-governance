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

package appruntime

import (
	"errors"
	"fmt"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/datatype"
)

type namedValue struct {
	source string
	value  string
}

// Inspect classifies a persisted snapshot without guessing missing TAF language.
func Inspect(s Snapshot) Report {
	report := Report{
		AppID:         s.AppID,
		PersistedType: s.AppType,
	}
	switch s.AppType {
	case AppTypeHelm, AppTypeAgones:
		inspectHelmLike(s, &report)
	case AppTypeTRPC, AppTypeTAF, AppTypeBKMSApp:
		inspectAppModel(s, &report)
	case "":
		report.Issues = append(report.Issues, newIssue(
			IssueUnknownAppType, true, "app type is empty", nil,
		))
	default:
		report.Issues = append(report.Issues, newIssue(
			IssueUnknownAppType, true, fmt.Sprintf("unknown app type %q", s.AppType),
			map[string]string{"type": s.AppType},
		))
	}
	return report
}

// Normalize returns the unified stack when Inspect has no blocking issues.
func Normalize(s Snapshot) (Stack, error) {
	report := Inspect(s)
	if err := report.Err(); err != nil {
		return Stack{}, err
	}
	return report.Stack, nil
}

func inspectHelmLike(s Snapshot, report *Report) {
	report.Stack = Stack{Type: s.AppType}
	if s.Language != "" || s.Framework != "" || s.TrpcSpecLanguage != "" {
		report.Issues = append(report.Issues, newIssue(
			IssueUnexpectedStackFields,
			true,
			"helm/agones application carries language or framework fields",
			map[string]string{
				"language":  s.Language,
				"framework": s.Framework,
			},
		))
	}
}

func inspectAppModel(s Snapshot, report *Report) {
	if s.AppModelMissing {
		report.Issues = append(report.Issues, newIssue(
			IssueOrphanApplication, true, "appmodel-managed application has no AppModel",
			map[string]string{"type": s.AppType},
		))
	}
	framework, frameworkFrom := resolveFramework(s, report)
	language, languageFrom := resolveLanguage(s, framework, report)
	report.Sources = Sources{FrameworkFrom: frameworkFrom, LanguageFrom: languageFrom}
	report.Stack = Stack{
		Type:      AppTypeBKMSApp,
		Framework: framework,
		Language:  language,
	}
	if framework != "" && language != "" {
		if err := ValidateStack(report.Stack); err != nil {
			code := IssueInvalidCombination
			if errors.Is(err, ErrUnknownFramework) {
				code = IssueUnknownFramework
			} else if errors.Is(err, ErrUnknownLanguage) {
				code = IssueUnknownLanguage
			}
			report.Issues = append(report.Issues, newIssue(
				code, true, err.Error(),
				map[string]string{"framework": framework, "language": language},
			))
		}
	}
	inspectFrameworkConfig(s, framework, report)
}

func resolveFramework(s Snapshot, report *Report) (framework, source string) {
	candidates := make([]namedValue, 0, 3)
	if s.Framework != "" {
		candidates = append(candidates, namedValue{source: "application.framework", value: s.Framework})
	}
	if mapped := FrameworkFromLegacyType(s.AppType); mapped != "" {
		candidates = append(candidates, namedValue{source: "application.type", value: mapped})
	}
	if productFramework := productFrameworkFromWorkloadType(s.WorkloadType); productFramework != "" {
		candidates = append(candidates, namedValue{source: "workload.type", value: productFramework})
	}

	uniq := uniqueNamedValues(candidates)
	switch len(uniq) {
	case 0:
		if s.AppType == AppTypeBKMSApp {
			report.Issues = append(report.Issues, newIssue(
				IssueMissingFramework, true, "bkmsApp is missing framework", nil,
			))
		}
		return "", ""
	case 1:
		if _, ok := catalogByName[uniq[0].value]; !ok {
			report.Issues = append(report.Issues, newIssue(
				IssueUnknownFramework, true, fmt.Sprintf("unknown framework %q", uniq[0].value),
				map[string]string{"framework": uniq[0].value, "source": uniq[0].source},
			))
			return "", uniq[0].source
		}
		return uniq[0].value, uniq[0].source
	default:
		fields := map[string]string{}
		for _, item := range uniq {
			fields[item.source] = item.value
		}
		report.Issues = append(report.Issues, newIssue(
			IssueTypeFrameworkConflict, true, "type/framework values conflict", fields,
		))
		return "", ""
	}
}

func productFrameworkFromWorkloadType(workloadType string) string {
	switch workloadType {
	case FrameworkTRPC, FrameworkTAF, FrameworkBlank:
		return workloadType
	default:
		// Empty or internal types such as "standard" are not product frameworks.
		return ""
	}
}

func resolveLanguage(s Snapshot, framework string, report *Report) (language, source string) {
	candidates := make([]namedValue, 0, 3)
	if s.Language != "" {
		candidates = append(candidates, namedValue{source: "application.language", value: s.Language})
	}
	if s.TrpcSpecLanguage != "" {
		candidates = append(candidates, namedValue{source: "application.trpcSpec.language", value: s.TrpcSpecLanguage})
	}
	if s.TrpcConfigLanguage != "" {
		candidates = append(candidates, namedValue{
			source: "workload.trpcConfig.language",
			value:  s.TrpcConfigLanguage,
		})
	}

	uniq := uniqueNamedValues(candidates)
	switch len(uniq) {
	case 0:
		blocking := languageRequired(s.AppType, framework)
		report.Issues = append(report.Issues, newIssue(
			IssueIncompleteLanguage,
			blocking,
			"no valid language found",
			map[string]string{"framework": framework, "type": s.AppType},
		))
		return "", ""
	case 1:
		if !KnownLanguages.Contains(uniq[0].value) {
			report.Issues = append(report.Issues, newIssue(
				IssueUnknownLanguage, true, fmt.Sprintf("unknown language %q", uniq[0].value),
				map[string]string{"language": uniq[0].value, "source": uniq[0].source},
			))
			return "", uniq[0].source
		}
		return uniq[0].value, uniq[0].source
	default:
		fields := map[string]string{}
		for _, item := range uniq {
			fields[item.source] = item.value
		}
		report.Issues = append(report.Issues, newIssue(
			IssueLanguageConflict, true, "language values conflict", fields,
		))
		return "", ""
	}
}

func languageRequired(appType, framework string) bool {
	if appType == AppTypeBKMSApp {
		return true
	}
	if framework == FrameworkTAF || appType == AppTypeTAF {
		// TAF language was never persisted; do not guess cpp.
		return false
	}
	return framework == FrameworkTRPC || framework == FrameworkBlank || appType == AppTypeTRPC
}

func inspectFrameworkConfig(s Snapshot, framework string, report *Report) {
	if s.HasTrpcConfig && s.HasTafConfig {
		report.Issues = append(report.Issues, newIssue(
			IssueDualFrameworkConfig, true, "both trpcConfig and tafConfig are present", nil,
		))
	}

	hasMap := s.FrameworkConfig != nil
	hasVersion := s.FrameworkConfigVersion != 0
	legacy := legacyFileMeta(framework, s)

	if hasMap && !hasVersion {
		report.Issues = append(report.Issues, newIssue(
			IssueMissingConfigVersion, true, "frameworkConfig is set without frameworkConfigVersion", nil,
		))
	}
	if hasVersion && s.FrameworkConfigVersion != FrameworkConfigVersionV1 {
		report.Issues = append(report.Issues, newIssue(
			IssueUnknownConfigVersion, true,
			fmt.Sprintf("unknown frameworkConfigVersion %d", s.FrameworkConfigVersion),
			map[string]string{"frameworkConfigVersion": fmt.Sprint(s.FrameworkConfigVersion)},
		))
	}

	if hasMap || hasVersion {
		cloned := datatype.CloneMap(s.FrameworkConfig)
		if cloned == nil {
			cloned = map[string]any{}
		}
		report.FrameworkConfig = cloned
		report.FrameworkConfigVersion = s.FrameworkConfigVersion
		if !hasVersion {
			report.FrameworkConfigVersion = FrameworkConfigVersionV1
		}
		newName, newPath := fileMetaFromMap(cloned)
		if legacy.hasMeta && len(cloned) > 0 && (legacy.fileName != newName || legacy.filePath != newPath) {
			report.Issues = append(report.Issues, newIssue(
				IssueFrameworkConfigConflict, true,
				"frameworkConfig file metadata conflicts with legacy config",
				map[string]string{
					"legacyFileName": legacy.fileName,
					"legacyFilePath": legacy.filePath,
					"mapFileName":    newName,
					"mapFilePath":    newPath,
				},
			))
		}
		if len(cloned) == 0 && legacy.hasMeta {
			report.FrameworkConfig = FileMetaToMap(legacy.fileName, legacy.filePath)
			report.FrameworkConfigVersion = FrameworkConfigVersionV1
		}
		return
	}

	report.FrameworkConfig = FileMetaToMap(legacy.fileName, legacy.filePath)
	report.FrameworkConfigVersion = FrameworkConfigVersionV1
}

type legacyMeta struct {
	fileName string
	filePath string
	hasMeta  bool
}

func legacyFileMeta(framework string, s Snapshot) legacyMeta {
	switch framework {
	case FrameworkTRPC:
		if s.TrpcFileName != "" || s.TrpcFilePath != "" {
			return legacyMeta{fileName: s.TrpcFileName, filePath: s.TrpcFilePath, hasMeta: true}
		}
	case FrameworkTAF:
		if s.TafFileName != "" || s.TafFilePath != "" {
			return legacyMeta{fileName: s.TafFileName, filePath: s.TafFilePath, hasMeta: true}
		}
	}
	return legacyMeta{}
}

func uniqueNamedValues(items []namedValue) []namedValue {
	out := make([]namedValue, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		if _, ok := seen[item.value]; ok {
			continue
		}
		seen[item.value] = struct{}{}
		out = append(out, item)
	}
	return out
}
