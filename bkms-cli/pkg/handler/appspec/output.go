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

package appspec

import (
	"fmt"
	"strings"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// StartCommandOutput is the output structure for start command view.
type StartCommandOutput struct {
	Command []string `json:"command" yaml:"command"`
	Args    []string `json:"args" yaml:"args"`
}

// FormatTable returns a plain-text table representation of the start command.
func (o *StartCommandOutput) FormatTable() string {
	if o == nil || (len(o.Command) == 0 && len(o.Args) == 0) {
		return "  Not configured"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %-20s %s\n", "Command:", formatStringSlice(o.Command)))
	sb.WriteString(fmt.Sprintf("  %-20s %s", "Args:", formatStringSlice(o.Args)))
	return sb.String()
}

// FormatViewAllTable formats the ViewAllResult as a plain-text table string.
func FormatViewAllTable(result *ViewAllResult) string {
	var sb strings.Builder

	// 1. Start Command
	sb.WriteString("=== Start Command ===\n")
	if result.StartCommand != nil {
		sb.WriteString(result.StartCommand.FormatTable())
	} else {
		sb.WriteString("  Not configured")
	}
	sb.WriteString("\n\n")

	// 2. Lifecycle
	sb.WriteString("=== Lifecycle ===\n")
	sb.WriteString(FormatLifecycleTable(result.Lifecycle))
	sb.WriteString("\n\n")

	// 3. Probes
	sb.WriteString("=== Probes ===\n")
	sb.WriteString(FormatProbeTable(result.Probe))
	sb.WriteString("\n\n")

	// 4. Resources
	sb.WriteString("=== Resources ===\n")
	sb.WriteString(FormatResourcesTable(result.Resources))
	sb.WriteString("\n\n")

	// 5. Update Strategy
	sb.WriteString("=== Update Strategy ===\n")
	sb.WriteString(FormatUpdateStrategyTable(result.UpdateStrategy))
	sb.WriteString("\n\n")

	// 6. Metadata - Labels
	sb.WriteString("=== Labels ===\n")
	sb.WriteString(FormatLabelsTable(result.Labels))
	sb.WriteString("\n\n")

	// 7. Metadata - Annotations
	sb.WriteString("=== Annotations ===\n")
	sb.WriteString(FormatAnnotationsTable(result.Annotations))

	return sb.String()
}

// FormatSectionTable formats a single section's data as a plain-text table string.
func FormatSectionTable(section client.AppSpecSectionName, data any) string {
	switch section {
	case client.AppSpecSectionResources:
		if v, ok := data.(*client.ResourcesConfig); ok {
			return FormatResourcesTable(v)
		}
	case client.AppSpecSectionUpdateStrategy:
		if v, ok := data.(*client.UpdateStrategyConfig); ok {
			return FormatUpdateStrategyTable(v)
		}
	case client.AppSpecSectionLifecycle:
		if v, ok := data.(*client.LifecycleConfig); ok {
			return FormatLifecycleTable(v)
		}
	case client.AppSpecSectionProbe:
		if v, ok := data.(*client.ProbeConfig); ok {
			return FormatProbeTable(v)
		}
	case client.AppSpecSectionLabels:
		if v, ok := data.(*client.LabelsConfig); ok {
			return FormatLabelsTable(v)
		}
	case client.AppSpecSectionAnnotations:
		if v, ok := data.(*client.AnnotationsConfig); ok {
			return FormatAnnotationsTable(v)
		}
	}
	return "  Not configured"
}

// --- Section-specific table formatters ---

// FormatResourcesTable formats resources config as a table.
func FormatResourcesTable(r *client.ResourcesConfig) string {
	if r == nil {
		return "  Not configured"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %-20s %s\n", "Replicas:", ptrInt32Str(r.Replicas)))
	sb.WriteString(fmt.Sprintf("  %-20s %s\n", "CPU Requests:", ptrStr(r.CPURequests)))
	sb.WriteString(fmt.Sprintf("  %-20s %s\n", "CPU Limits:", ptrStr(r.CPULimits)))
	sb.WriteString(fmt.Sprintf("  %-20s %s\n", "Memory Requests:", ptrStr(r.MemoryRequests)))
	sb.WriteString(fmt.Sprintf("  %-20s %s", "Memory Limits:", ptrStr(r.MemoryLimits)))
	return sb.String()
}

// FormatUpdateStrategyTable formats update-strategy config as a table.
func FormatUpdateStrategyTable(s *client.UpdateStrategyConfig) string {
	if s == nil {
		return "  Not configured"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %-20s %s\n", "Max Surge:", ptrStr(s.MaxSurge)))
	sb.WriteString(fmt.Sprintf("  %-20s %s", "Max Unavailable:", ptrStr(s.MaxUnavailable)))
	return sb.String()
}

// FormatLifecycleTable formats lifecycle config as a table.
func FormatLifecycleTable(l *client.LifecycleConfig) string {
	if l == nil {
		return "  Not configured"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %-20s %s\n", "Grace Period (s):", ptrStr(l.TerminationGracePeriodSeconds)))
	sb.WriteString("\n  [PostStart Hook]\n")
	sb.WriteString(formatLifecycleHandlerTable(l.PostStart))
	sb.WriteString("\n  [PreStop Hook]\n")
	sb.WriteString(formatLifecycleHandlerTable(l.PreStop))
	return sb.String()
}

func formatLifecycleHandlerTable(handler *client.LifecycleHandlerConfig) string {
	if handler == nil {
		return "    Not configured"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("    %-18s %s\n", "Type:", handler.Type))
	switch handler.Type {
	case "EXEC":
		if handler.SleepSeconds != nil && *handler.SleepSeconds != "" {
			sb.WriteString(fmt.Sprintf("    %-18s %s\n", "Sleep Seconds:", *handler.SleepSeconds))
		}
		if handler.ShCommand != "" {
			sb.WriteString(fmt.Sprintf("    %-18s %s\n", "Shell Command:", handler.ShCommand))
		}
		if len(handler.Command) > 0 {
			sb.WriteString(fmt.Sprintf("    %-18s %s\n", "Command:", strings.Join(handler.Command, " ")))
		}
	case "HTTP":
		sb.WriteString(fmt.Sprintf("    %-18s %s\n", "URL:", handler.URL))
		if handler.Port > 0 {
			sb.WriteString(fmt.Sprintf("    %-18s %d\n", "Port:", handler.Port))
		}
		if len(handler.Headers) > 0 {
			sb.WriteString(fmt.Sprintf("    %-18s %s\n", "Headers:", formatMap(handler.Headers)))
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// FormatProbeTable formats probe config as a table.
func FormatProbeTable(p *client.ProbeConfig) string {
	if p == nil {
		return "  Not configured"
	}
	var sb strings.Builder
	sb.WriteString("\n  [Liveness Probe]\n")
	sb.WriteString(formatProbeItemTable(p.Liveness))
	sb.WriteString("\n  [Readiness Probe]\n")
	sb.WriteString(formatProbeItemTable(p.Readiness))
	sb.WriteString("\n  [Startup Probe]\n")
	sb.WriteString(formatProbeItemTable(p.Startup))
	return sb.String()
}

func formatProbeItemTable(item *client.ProbeItemConfig) string {
	if item == nil {
		return "    Not configured"
	}
	var sb strings.Builder
	if item.Handler != nil {
		sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Type:", item.Handler.Type))
		switch item.Handler.Type {
		case "HTTP":
			sb.WriteString(fmt.Sprintf("    %-20s %s\n", "URL:", item.Handler.URL))
			if item.Handler.Port > 0 {
				sb.WriteString(fmt.Sprintf("    %-20s %d\n", "Port:", item.Handler.Port))
			}
		case "TCP":
			if item.Handler.Port > 0 {
				sb.WriteString(fmt.Sprintf("    %-20s %d\n", "Port:", item.Handler.Port))
			}
		case "EXEC":
			if item.Handler.ShCommand != "" {
				sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Shell Command:", item.Handler.ShCommand))
			}
			if len(item.Handler.Command) > 0 {
				sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Command:", strings.Join(item.Handler.Command, " ")))
			}
		}
	}
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Initial Delay (s):", ptrInt32Str(item.InitialDelaySeconds)))
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Timeout (s):", ptrInt32Str(item.TimeoutSeconds)))
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Period (s):", ptrInt32Str(item.PeriodSeconds)))
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Success Threshold:", ptrInt32Str(item.SuccessThreshold)))
	sb.WriteString(fmt.Sprintf("    %-20s %s", "Failure Threshold:", ptrInt32Str(item.FailureThreshold)))
	return sb.String()
}

// FormatLabelsTable formats labels config as a table.
func FormatLabelsTable(l *client.LabelsConfig) string {
	if l == nil || len(l.Labels) == 0 {
		return "  Not configured"
	}
	var sb strings.Builder
	for k, v := range l.Labels {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", k+":", v))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// FormatAnnotationsTable formats annotations config as a table.
func FormatAnnotationsTable(a *client.AnnotationsConfig) string {
	if a == nil || len(a.Annotations) == 0 {
		return "  Not configured"
	}
	var sb strings.Builder
	for k, v := range a.Annotations {
		sb.WriteString(fmt.Sprintf("  %-30s %s\n", k+":", v))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// --- Helper functions ---

func ptrStr(p *string) string {
	if p == nil {
		return "-"
	}
	return *p
}

func ptrInt32Str(p *int32) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *p)
}

func formatStringSlice(s []string) string {
	if len(s) == 0 {
		return "-"
	}
	return strings.Join(s, " ")
}

func formatMap(m map[string]string) string {
	if len(m) == 0 {
		return "-"
	}
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ", ")
}
