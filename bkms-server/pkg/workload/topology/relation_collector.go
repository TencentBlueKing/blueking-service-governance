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

package topology

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TencentBlueKing/gopkg/mapx"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// RelationCollector 从 Kubernetes 非结构化资源对象中收集扩展关系
type RelationCollector struct {
	resources map[string]*unstructured.Unstructured
}

// NewRelationCollector 创建一个新的关系收集器
// resources 是键为 "kind/namespace/name" 的资源对象映射
func NewRelationCollector(resources map[string]*unstructured.Unstructured) *RelationCollector {
	return &RelationCollector{resources: resources}
}

// Collect 执行所有关系收集，返回完整的扩展关系列表
func (c *RelationCollector) Collect() []ResourceRelation {
	var relations []ResourceRelation

	for _, obj := range c.resources {
		kind := obj.GetKind()
		namespace := obj.GetNamespace()
		name := obj.GetName()

		relations = append(relations, c.collectOwnerReferences(obj, kind, namespace, name)...)
		relations = append(relations, c.collectLabelSelectors(obj, kind, namespace, name)...)
		relations = append(relations, c.collectVolumeMounts(obj, kind, namespace, name)...)
		relations = append(relations, c.collectBackendRefs(obj, kind, namespace, name)...)
		relations = append(relations, c.collectEnvRefs(obj, kind, namespace, name)...)
		relations = append(relations, c.collectScaleTargetRefs(obj, kind, namespace, name)...)
		relations = append(relations, c.collectServiceAccountRefs(obj, kind, namespace, name)...)
	}

	return relations
}

// collectOwnerReferences 从资源的 metadata.ownerReferences 中收集所有者引用关系
func (c *RelationCollector) collectOwnerReferences(
	obj *unstructured.Unstructured,
	kind, namespace, name string,
) []ResourceRelation {
	ownerRefs := obj.GetOwnerReferences()
	var relations []ResourceRelation

	for _, ref := range ownerRefs {
		relations = append(relations, ResourceRelation{
			RelationType:    RelationTypeOwnerReference,
			SourceKind:      ref.Kind,
			SourceNamespace: namespace,
			SourceName:      ref.Name,
			TargetKind:      kind,
			TargetNamespace: namespace,
			TargetName:      name,
			SourceFieldPath: "metadata.ownerReferences",
			Summary:         fmt.Sprintf("%s/%s owns %s/%s", ref.Kind, ref.Name, kind, name),
		})
	}
	return relations
}

// collectLabelSelectors 从 Service/Deployment 的 spec.selector 中收集标签选择器关系
func (c *RelationCollector) collectLabelSelectors(
	obj *unstructured.Unstructured,
	kind, namespace, name string,
) []ResourceRelation {
	var relations []ResourceRelation

	switch kind {
	case k8skind.SVC:
		// Service: spec.selector
		selectorMap, found, _ := unstructured.NestedStringMap(obj.Object, "spec", "selector")
		if found && len(selectorMap) > 0 {
			relations = append(relations, ResourceRelation{
				RelationType:    RelationTypeLabelSelector,
				SourceKind:      kind,
				SourceNamespace: namespace,
				SourceName:      name,
				TargetKind:      k8skind.Po,
				TargetNamespace: namespace,
				TargetName:      TargetNameWildcard,
				SourceFieldPath: "spec.selector",
				Summary:         fmt.Sprintf("Service/%s selects Pods matching %s", name, formatLabels(selectorMap)),
				MatchedLabels:   selectorMap,
			})
		}
	case k8skind.GameDeploy, k8skind.GameSTS, k8skind.Deploy, k8skind.STS, k8skind.DS, k8skind.RS:
		// GameDeployment/GameStatefulSet/Deployment/StatefulSet/DaemonSet: spec.selector.matchLabels
		matchLabels, found, _ := unstructured.NestedStringMap(obj.Object, "spec", "selector", "matchLabels")
		if found && len(matchLabels) > 0 {
			relations = append(relations, ResourceRelation{
				RelationType:    RelationTypeLabelSelector,
				SourceKind:      kind,
				SourceNamespace: namespace,
				SourceName:      name,
				TargetKind:      k8skind.Po,
				TargetNamespace: namespace,
				TargetName:      TargetNameWildcard,
				SourceFieldPath: "spec.selector.matchLabels",
				Summary:         fmt.Sprintf("%s/%s selects Pods matching %s", kind, name, formatLabels(matchLabels)),
				MatchedLabels:   matchLabels,
			})
		}
	}
	return relations
}

// collectVolumeMounts 从 Pod/Deployment 等的 volumes 中收集 ConfigMap/Secret 引用关系
func (c *RelationCollector) collectVolumeMounts(
	obj *unstructured.Unstructured,
	kind, namespace, name string,
) []ResourceRelation {
	var relations []ResourceRelation

	// 查找 volumes（直接在 spec.volumes 或嵌套在 spec.template.spec.volumes）
	volumePaths := [][]string{
		{"spec", "volumes"},
		{"spec", "template", "spec", "volumes"},
	}

	for _, path := range volumePaths {
		volumes, found, _ := unstructured.NestedSlice(obj.Object, path...)
		if !found {
			continue
		}

		fieldPath := strings.Join(path, ".")
		for _, v := range volumes {
			vol, ok := v.(map[string]any)
			if !ok {
				continue
			}

			// ConfigMap volume
			if cm, ok := vol["configMap"].(map[string]any); ok {
				if cmName, ok := cm["name"].(string); ok && cmName != "" {
					relations = append(relations, newVolumeMountRelation(
						kind, namespace, name, k8skind.CM, cmName, fieldPath,
					))
				}
			}

			// Secret volume
			if secret, ok := vol["secret"].(map[string]any); ok {
				if secretName, ok := secret["secretName"].(string); ok && secretName != "" {
					relations = append(relations, newVolumeMountRelation(
						kind, namespace, name, k8skind.Secret, secretName, fieldPath,
					))
				}
			}
		}
	}

	return relations
}

// collectBackendRefs 从 Ingress 的 rules 中收集 backend Service 引用关系
func (c *RelationCollector) collectBackendRefs(
	obj *unstructured.Unstructured,
	kind, namespace, name string,
) []ResourceRelation {
	if kind != k8skind.Ing {
		return nil
	}

	var relations []ResourceRelation
	rules, found, _ := unstructured.NestedSlice(obj.Object, "spec", "rules")
	if !found {
		return nil
	}

	for rIdx, rule := range rules {
		ruleMap, ok := rule.(map[string]any)
		if !ok {
			continue
		}

		httpBlock, ok := ruleMap["http"].(map[string]any)
		if !ok {
			continue
		}

		paths, ok := httpBlock["paths"].([]any)
		if !ok {
			continue
		}

		for pIdx, p := range paths {
			pathMap, ok := p.(map[string]any)
			if !ok {
				continue
			}

			backend, ok := pathMap["backend"].(map[string]any)
			if !ok {
				continue
			}

			// networking.k8s.io/v1 Ingress: backend.service.name
			if svc, ok := backend["service"].(map[string]any); ok {
				if svcName, ok := svc["name"].(string); ok && svcName != "" {
					relations = append(relations, ResourceRelation{
						RelationType:    RelationTypeBackendRef,
						SourceKind:      kind,
						SourceNamespace: namespace,
						SourceName:      name,
						TargetKind:      k8skind.SVC,
						TargetNamespace: namespace,
						TargetName:      svcName,
						SourceFieldPath: fmt.Sprintf("spec.rules[%d].http.paths[%d].backend.service.name", rIdx, pIdx),
						TargetFieldPath: "metadata.name",
						Summary:         fmt.Sprintf("Ingress/%s routes to Service/%s", name, svcName),
					})
				}
			}

			// legacy extensions/v1beta1 Ingress: backend.serviceName
			if svcName, ok := backend["serviceName"].(string); ok && svcName != "" {
				relations = append(relations, ResourceRelation{
					RelationType:    RelationTypeBackendRef,
					SourceKind:      kind,
					SourceNamespace: namespace,
					SourceName:      name,
					TargetKind:      k8skind.SVC,
					TargetNamespace: namespace,
					TargetName:      svcName,
					SourceFieldPath: fmt.Sprintf("spec.rules[%d].http.paths[%d].backend.serviceName", rIdx, pIdx),
					TargetFieldPath: "metadata.name",
					Summary:         fmt.Sprintf("Ingress/%s routes to Service/%s", name, svcName),
				})
			}
		}
	}

	return relations
}

// collectEnvRefs 从 Pod/Deployment 等的容器环境变量中收集 ConfigMap/Secret 引用关系
// 支持两种引用方式：
//   - env[].valueFrom.configMapKeyRef / secretKeyRef（单个环境变量引用）
//   - envFrom[].configMapRef / secretRef（批量环境变量引用）
func (c *RelationCollector) collectEnvRefs(
	obj *unstructured.Unstructured,
	kind, namespace, name string,
) []ResourceRelation {
	var relations []ResourceRelation

	// 容器所在的两种路径：Pod 直接在 spec.containers，Deployment 等在 spec.template.spec.containers
	containerPaths := [][]string{
		{"spec", "containers"},
		{"spec", "template", "spec", "containers"},
	}

	// 用 "Kind/Name" 作为 key 去重，避免同一资源对同一 ConfigMap/Secret 产生多条重复关系
	seen := make(map[string]bool)

	for _, basePath := range containerPaths {
		containers, found, _ := unstructured.NestedSlice(obj.Object, basePath...)
		if !found {
			continue
		}

		fieldPrefix := strings.Join(basePath, ".")

		for cIdx, container := range containers {
			ctrMap, ctrOK := container.(map[string]any)
			if !ctrOK {
				continue
			}

			relations = append(relations, c.collectEnvValueFromRefs(
				kind, namespace, name, seen, ctrMap, fieldPrefix, cIdx)...,
			)
			relations = append(relations, c.collectEnvFromRefs(
				kind, namespace, name, seen, ctrMap, fieldPrefix, cIdx)...,
			)
		}
	}

	return relations
}

// collectEnvValueFromRefs 从单个容器的 env[].valueFrom 中收集 configMapKeyRef/secretKeyRef 引用
func (c *RelationCollector) collectEnvValueFromRefs(
	kind, namespace, name string,
	seen map[string]bool,
	container map[string]any,
	fieldPrefix string,
	cIdx int,
) []ResourceRelation {
	envList, envOK := container["env"].([]any)
	if !envOK {
		return nil
	}

	var relations []ResourceRelation
	for eIdx, envItem := range envList {
		envMap, mapOK := envItem.(map[string]any)
		if !mapOK {
			continue
		}
		valueFrom, vfOK := envMap["valueFrom"].(map[string]any)
		if !vfOK {
			continue
		}

		// configMapKeyRef
		if cmRef, refOK := valueFrom["configMapKeyRef"].(map[string]any); refOK {
			if cmName := mapx.GetStr(cmRef, "name"); cmName != "" {
				key := fmt.Sprintf("%s/%s", k8skind.CM, cmName)
				if seen[key] {
					continue
				}
				seen[key] = true
				relations = append(relations, newEnvRefRelation(
					kind, namespace, name, k8skind.CM, cmName,
					fmt.Sprintf("%s[%d].env[%d].valueFrom.configMapKeyRef.name", fieldPrefix, cIdx, eIdx),
				))
			}
		}

		// secretKeyRef
		if secRef, refOK := valueFrom["secretKeyRef"].(map[string]any); refOK {
			if secName := mapx.GetStr(secRef, "name"); secName != "" {
				key := fmt.Sprintf("%s/%s", k8skind.Secret, secName)
				if seen[key] {
					continue
				}
				seen[key] = true
				relations = append(relations, newEnvRefRelation(
					kind, namespace, name, k8skind.Secret, secName,
					fmt.Sprintf("%s[%d].env[%d].valueFrom.secretKeyRef.name", fieldPrefix, cIdx, eIdx),
				))
			}
		}
	}

	return relations
}

// collectEnvFromRefs 从单个容器的 envFrom[] 中收集 configMapRef/secretRef 引用
func (c *RelationCollector) collectEnvFromRefs(
	kind, namespace, name string,
	seen map[string]bool,
	container map[string]any,
	fieldPrefix string,
	cIdx int,
) []ResourceRelation {
	envFromList, efOK := container["envFrom"].([]any)
	if !efOK {
		return nil
	}

	var relations []ResourceRelation
	for efIdx, envFromItem := range envFromList {
		efMap, mapOK := envFromItem.(map[string]any)
		if !mapOK {
			continue
		}

		// configMapRef
		if cmRef, refOK := efMap["configMapRef"].(map[string]any); refOK {
			if cmName := mapx.GetStr(cmRef, "name"); cmName != "" {
				key := fmt.Sprintf("%s/%s", k8skind.CM, cmName)
				if seen[key] {
					continue
				}
				seen[key] = true
				relations = append(relations, newEnvRefRelation(
					kind, namespace, name, k8skind.CM, cmName,
					fmt.Sprintf("%s[%d].envFrom[%d].configMapRef.name", fieldPrefix, cIdx, efIdx),
				))
			}
		}

		// secretRef
		if secRef, refOK := efMap["secretRef"].(map[string]any); refOK {
			if secName := mapx.GetStr(secRef, "name"); secName != "" {
				key := fmt.Sprintf("%s/%s", k8skind.Secret, secName)
				if seen[key] {
					continue
				}
				seen[key] = true
				relations = append(relations, newEnvRefRelation(
					kind, namespace, name, k8skind.Secret, secName,
					fmt.Sprintf("%s[%d].envFrom[%d].secretRef.name", fieldPrefix, cIdx, efIdx),
				))
			}
		}
	}

	return relations
}

// newVolumeMountRelation 构造一条 volume_mount 类型的扩展关系
func newVolumeMountRelation(srcKind, namespace, srcName, tgtKind, tgtName, srcFieldPath string) ResourceRelation {
	return ResourceRelation{
		RelationType:    RelationTypeVolumeMount,
		SourceKind:      srcKind,
		SourceNamespace: namespace,
		SourceName:      srcName,
		TargetKind:      tgtKind,
		TargetNamespace: namespace,
		TargetName:      tgtName,
		SourceFieldPath: srcFieldPath,
		TargetFieldPath: "metadata.name",
		Summary:         fmt.Sprintf("%s/%s mounts %s/%s", srcKind, srcName, tgtKind, tgtName),
	}
}

// newEnvRefRelation 构造一条 env_ref 类型的扩展关系
func newEnvRefRelation(srcKind, namespace, srcName, tgtKind, tgtName, srcFieldPath string) ResourceRelation {
	return ResourceRelation{
		RelationType:    RelationTypeEnvRef,
		SourceKind:      srcKind,
		SourceNamespace: namespace,
		SourceName:      srcName,
		TargetKind:      tgtKind,
		TargetNamespace: namespace,
		TargetName:      tgtName,
		SourceFieldPath: srcFieldPath,
		TargetFieldPath: "metadata.name",
		Summary:         fmt.Sprintf("%s/%s refers env from %s/%s", srcKind, srcName, tgtKind, tgtName),
	}
}

// collectScaleTargetRefs 从 HPA / GPA 的 spec.scaleTargetRef 中收集扩缩目标引用关系
func (c *RelationCollector) collectScaleTargetRefs(
	obj *unstructured.Unstructured,
	kind, namespace, name string,
) []ResourceRelation {
	if kind != k8skind.HPA && kind != k8skind.GPA {
		return nil
	}

	scaleTargetRef, found, _ := unstructured.NestedMap(obj.Object, "spec", "scaleTargetRef")
	if !found || len(scaleTargetRef) == 0 {
		return nil
	}

	targetKind, _ := scaleTargetRef["kind"].(string)
	targetName, _ := scaleTargetRef["name"].(string)
	if targetKind == "" || targetName == "" {
		return nil
	}

	return []ResourceRelation{
		{
			RelationType:    RelationTypeScaleTargetRef,
			SourceKind:      kind,
			SourceNamespace: namespace,
			SourceName:      name,
			TargetKind:      targetKind,
			TargetNamespace: namespace,
			TargetName:      targetName,
			SourceFieldPath: "spec.scaleTargetRef",
			TargetFieldPath: "metadata.name",
			Summary:         fmt.Sprintf("%s/%s scales %s/%s", kind, name, targetKind, targetName),
		},
	}
}

// serviceAccountRefKinds 支持从 spec.template.spec.serviceAccountName 提取 SA 引用的工作负载类型
var serviceAccountRefKinds = map[string]bool{
	k8skind.Deploy:     true,
	k8skind.STS:        true,
	k8skind.DS:         true,
	k8skind.GameDeploy: true,
	k8skind.GameSTS:    true,
	k8skind.Job:        true,
}

// defaultServiceAccountName 默认的 ServiceAccount 名称，不需要生成关联边
const defaultServiceAccountName = "default"

// collectServiceAccountRefs 从工作负载的 serviceAccountName 字段中收集 ServiceAccount 引用关系
func (c *RelationCollector) collectServiceAccountRefs(
	obj *unstructured.Unstructured,
	kind, namespace, name string,
) []ResourceRelation {
	var saName string
	var fieldPath string

	switch {
	case serviceAccountRefKinds[kind]:
		// Deployment/StatefulSet/DaemonSet/GameDeployment/GameStatefulSet/Job:
		// spec.template.spec.serviceAccountName
		val, found, _ := unstructured.NestedString(
			obj.Object, "spec", "template", "spec", "serviceAccountName",
		)
		if !found || val == "" {
			return nil
		}
		saName = val
		fieldPath = "spec.template.spec.serviceAccountName"

	case kind == k8skind.CJ:
		// CronJob: spec.jobTemplate.spec.template.spec.serviceAccountName
		val, found, _ := unstructured.NestedString(
			obj.Object, "spec", "jobTemplate", "spec", "template", "spec", "serviceAccountName",
		)
		if !found || val == "" {
			return nil
		}
		saName = val
		fieldPath = "spec.jobTemplate.spec.template.spec.serviceAccountName"

	default:
		return nil
	}

	// 跳过默认的 ServiceAccount
	if saName == defaultServiceAccountName {
		return nil
	}

	return []ResourceRelation{
		{
			RelationType:    RelationTypeServiceAccountRef,
			SourceKind:      kind,
			SourceNamespace: namespace,
			SourceName:      name,
			TargetKind:      k8skind.SA,
			TargetNamespace: namespace,
			TargetName:      saName,
			SourceFieldPath: fieldPath,
			TargetFieldPath: "metadata.name",
			Summary:         fmt.Sprintf("%s/%s references ServiceAccount/%s", kind, name, saName),
		},
	}
}

// formatLabels 将标签映射格式化为可读字符串
func formatLabels(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(labels))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
	}
	return strings.Join(parts, ",")
}
