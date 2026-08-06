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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// NodeExtrasProvider 节点扩展字段提取函数类型
// 接收集群资源对象，返回该类型资源提供的 extras key-value 对
type NodeExtrasProvider func(obj *unstructured.Unstructured) map[string]string

// kindExtrasProviders 各资源类型的 extras 提取函数注册表
// 通过注册表模式，让每种资源能提供的 extras 字段一目了然
var kindExtrasProviders = map[string]NodeExtrasProvider{
	k8skind.Po:     extractPodExtras,
	k8skind.Deploy: extractDeploymentExtras,
	k8skind.STS:    extractWorkloadExtras,
	k8skind.DS:     extractWorkloadExtras,
	k8skind.RS:     extractReplicaSetExtras,
	k8skind.SVC:    extractServiceExtras,
	k8skind.Ing:    extractIngressExtras,
	k8skind.CM:     extractConfigMapExtras,
	k8skind.Secret: extractSecretExtras,
	k8skind.SA:     extractServiceAccountExtras,
}

// extractPodExtras 提取 Pod 的扩展字段：image, ip, nodeName, restartCount, ready, phase
func extractPodExtras(obj *unstructured.Unstructured) map[string]string {
	extras := make(map[string]string)
	if image := extractFirstContainerImage(obj, "spec", "containers"); image != "" {
		extras[ExtrasKeyImage] = image
	}
	// 提取 Pod IP
	podIP, found, _ := unstructured.NestedString(obj.Object, "status", "podIP")
	if found && podIP != "" {
		extras[ExtrasKeyPodIP] = podIP
	}
	// 提取 nodeName
	nodeName, found, _ := unstructured.NestedString(obj.Object, "spec", "nodeName")
	if found && nodeName != "" {
		extras[ExtrasKeyNodeName] = nodeName
	}
	// 提取 phase
	phase, found, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if found && phase != "" {
		extras[ExtrasKeyPhase] = phase
	}
	// 提取 restartCount 和 ready（从 containerStatuses 汇总）
	containerStatuses, found, _ := unstructured.NestedSlice(obj.Object, "status", "containerStatuses")
	if found {
		var totalRestarts int64
		allReady := true
		for _, cs := range containerStatuses {
			csMap, ok := cs.(map[string]any)
			if !ok {
				continue
			}
			restarts, _ := csMap["restartCount"].(int64)
			totalRestarts += restarts
			ready, _ := csMap["ready"].(bool)
			if !ready {
				allReady = false
			}
		}
		extras[ExtrasKeyRestartCount] = fmt.Sprintf("%d", totalRestarts)
		extras[ExtrasKeyReady] = fmt.Sprintf("%t", allReady)
	}
	return extras
}

// extractDeploymentExtras 提取 Deployment 的扩展字段：image, replicas, readyReplicas, availableReplicas, strategy
func extractDeploymentExtras(obj *unstructured.Unstructured) map[string]string {
	extras := extractWorkloadExtras(obj)
	// 提取 availableReplicas
	availableReplicas, found, _ := unstructured.NestedInt64(obj.Object, "status", "availableReplicas")
	if found {
		extras[ExtrasKeyAvailableReplicas] = fmt.Sprintf("%d", availableReplicas)
	}
	// 提取 strategy type
	strategy, found, _ := unstructured.NestedString(obj.Object, "spec", "strategy", "type")
	if found && strategy != "" {
		extras[ExtrasKeyStrategy] = strategy
	}
	return extras
}

// extractReplicaSetExtras 提取 ReplicaSet 的扩展字段：image, replicas, readyReplicas, ownerDeployment
func extractReplicaSetExtras(obj *unstructured.Unstructured) map[string]string {
	extras := extractWorkloadExtras(obj)
	// 提取 ownerDeployment（从 ownerReferences 中查找 Deployment）
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind == k8skind.Deploy {
			extras[ExtrasKeyOwnerDeployment] = ref.Name
			break
		}
	}
	return extras
}

// extractWorkloadExtras 提取工作负载（Deployment/StatefulSet/DaemonSet/ReplicaSet）的扩展字段：image, replicas, readyReplicas
func extractWorkloadExtras(obj *unstructured.Unstructured) map[string]string {
	extras := make(map[string]string)
	if image := extractFirstContainerImage(obj, "spec", "template", "spec", "containers"); image != "" {
		extras[ExtrasKeyImage] = image
	}
	extractReplicaInfo(obj, extras)
	return extras
}

// extractServiceExtras 提取 Service 的扩展字段：ports, selector, clusterIP, type
func extractServiceExtras(obj *unstructured.Unstructured) map[string]string {
	extras := make(map[string]string)
	// 端口信息
	ports, found, _ := unstructured.NestedSlice(obj.Object, "spec", "ports")
	if found {
		var portStrs []string
		for _, p := range ports {
			portMap, ok := p.(map[string]any)
			if !ok {
				continue
			}
			portNum, _ := portMap["port"].(int64)
			protocol, _ := portMap["protocol"].(string)
			if protocol == "" {
				protocol = "TCP"
			}
			portStrs = append(portStrs, fmt.Sprintf("%d/%s", portNum, protocol))
		}
		if len(portStrs) > 0 {
			extras[ExtrasKeyPorts] = strings.Join(portStrs, ",")
		}
	}
	// 选择器
	selector, found, _ := unstructured.NestedStringMap(obj.Object, "spec", "selector")
	if found && len(selector) > 0 {
		extras[ExtrasKeySelector] = formatLabels(selector)
	}
	// ClusterIP
	clusterIP, found, _ := unstructured.NestedString(obj.Object, "spec", "clusterIP")
	if found && clusterIP != "" {
		extras[ExtrasKeyClusterIP] = clusterIP
	}
	// Service Type
	svcType, found, _ := unstructured.NestedString(obj.Object, "spec", "type")
	if found && svcType != "" {
		extras[ExtrasKeyServiceType] = svcType
	}
	return extras
}

// extractIngressExtras 提取 Ingress 的扩展字段：host, rules, tls
// 拆分为多个辅助函数以降低圈复杂度：规则遍历、后端服务名解析、TLS 提取分别处理
func extractIngressExtras(obj *unstructured.Unstructured) map[string]string {
	extras := make(map[string]string)
	extractIngressRules(obj, extras)
	extractIngressTLS(obj, extras)
	return extras
}

// extractIngressRules 从 spec.rules 中提取 hosts 与 rules（host+path->service）信息并写入 extras
func extractIngressRules(obj *unstructured.Unstructured, extras map[string]string) {
	rules, found, _ := unstructured.NestedSlice(obj.Object, "spec", "rules")
	if !found || len(rules) == 0 {
		return
	}
	var hosts []string
	var ruleStrs []string
	for _, r := range rules {
		ruleMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		host, _ := ruleMap["host"].(string)
		if host != "" {
			hosts = append(hosts, host)
		}
		ruleStrs = append(ruleStrs, buildIngressRuleStrs(host, ruleMap)...)
	}
	if len(hosts) > 0 {
		extras[ExtrasKeyHost] = strings.Join(hosts, ",")
	}
	if len(ruleStrs) > 0 {
		extras[ExtrasKeyRules] = strings.Join(ruleStrs, "; ")
	}
}

// buildIngressRuleStrs 根据单条 ingress rule 的 http.paths 生成 "host+path->service" 形式的规则字符串列表
func buildIngressRuleStrs(host string, ruleMap map[string]any) []string {
	httpRule, _ := ruleMap["http"].(map[string]any)
	if httpRule == nil {
		return nil
	}
	paths, _ := httpRule["paths"].([]any)
	var ruleStrs []string
	for _, p := range paths {
		pathMap, ok := p.(map[string]any)
		if !ok {
			continue
		}
		path, _ := pathMap["path"].(string)
		backend, _ := pathMap["backend"].(map[string]any)
		if backend == nil {
			continue
		}
		svcName := extractIngressBackendServiceName(backend)
		if svcName != "" {
			ruleStrs = append(ruleStrs, fmt.Sprintf("%s%s->%s", host, path, svcName))
		}
	}
	return ruleStrs
}

// extractIngressBackendServiceName 从 ingress backend 中解析后端 Service 名
// 兼容 networking.k8s.io/v1 (backend.service.name) 与 extensions/v1beta1 (backend.serviceName)
func extractIngressBackendServiceName(backend map[string]any) string {
	if svc, ok := backend["service"].(map[string]any); ok {
		if name, _ := svc["name"].(string); name != "" {
			return name
		}
	}
	name, _ := backend["serviceName"].(string)
	return name
}

// extractIngressTLS 从 spec.tls 中提取 TLS 配置（hosts+secret）并写入 extras
func extractIngressTLS(obj *unstructured.Unstructured, extras map[string]string) {
	tlsSlice, found, _ := unstructured.NestedSlice(obj.Object, "spec", "tls")
	if !found || len(tlsSlice) == 0 {
		return
	}
	var tlsStrs []string
	for _, t := range tlsSlice {
		tlsMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		secretName, _ := tlsMap["secretName"].(string)
		tlsHosts, _ := tlsMap["hosts"].([]any)
		var hostNames []string
		for _, h := range tlsHosts {
			if hs, hOk := h.(string); hOk {
				hostNames = append(hostNames, hs)
			}
		}
		tlsStrs = append(tlsStrs, fmt.Sprintf("%s(%s)", strings.Join(hostNames, ","), secretName))
	}
	if len(tlsStrs) > 0 {
		extras[ExtrasKeyTLS] = strings.Join(tlsStrs, "; ")
	}
}

// extractConfigMapExtras 提取 ConfigMap 的扩展字段：keys, dataSize, binaryDataSize
func extractConfigMapExtras(obj *unstructured.Unstructured) map[string]string {
	extras := make(map[string]string)
	// 提取 data 的 keys 和 size
	data, found, _ := unstructured.NestedMap(obj.Object, "data")
	if found && len(data) > 0 {
		keys := make([]string, 0, len(data))
		var totalSize int
		for k, v := range data {
			keys = append(keys, k)
			if s, ok := v.(string); ok {
				totalSize += len(s)
			}
		}
		sort.Strings(keys)
		extras[ExtrasKeyKeys] = strings.Join(keys, ",")
		extras[ExtrasKeyDataSize] = fmt.Sprintf("%d", totalSize)
	}
	// 提取 binaryData 的 size
	binaryData, found, _ := unstructured.NestedMap(obj.Object, "binaryData")
	if found && len(binaryData) > 0 {
		var totalSize int
		for _, v := range binaryData {
			if s, ok := v.(string); ok {
				totalSize += len(s)
			}
		}
		extras[ExtrasKeyBinaryDataSize] = fmt.Sprintf("%d", totalSize)
		// 合并 binaryData 的 keys 到 keys 列表（无论 data 是否存在都需要设置）
		bKeys := make([]string, 0, len(binaryData))
		for k := range binaryData {
			bKeys = append(bKeys, k)
		}
		sort.Strings(bKeys)
		if existing := extras[ExtrasKeyKeys]; existing != "" {
			extras[ExtrasKeyKeys] = existing + "," + strings.Join(bKeys, ",")
		} else {
			extras[ExtrasKeyKeys] = strings.Join(bKeys, ",")
		}
	}
	return extras
}

// extractSecretExtras 提取 Secret 的扩展字段：secretType, keys
// 注意：值不返回，仅返回键名
func extractSecretExtras(obj *unstructured.Unstructured) map[string]string {
	extras := make(map[string]string)
	// 提取 Secret 类型
	secretType, found, _ := unstructured.NestedString(obj.Object, "type")
	if found && secretType != "" {
		extras[ExtrasKeySecretType] = secretType
	}
	// 提取 data 的 keys（不暴露值）
	data, found, _ := unstructured.NestedMap(obj.Object, "data")
	if found && len(data) > 0 {
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		extras[ExtrasKeyKeys] = strings.Join(keys, ",")
	}
	return extras
}

// extractServiceAccountExtras 提取 ServiceAccount 的扩展字段：secrets, automountToken
func extractServiceAccountExtras(obj *unstructured.Unstructured) map[string]string {
	extras := make(map[string]string)
	// 提取 secrets 引用列表
	secrets, found, _ := unstructured.NestedSlice(obj.Object, "secrets")
	if found && len(secrets) > 0 {
		var secretNames []string
		for _, s := range secrets {
			sMap, ok := s.(map[string]any)
			if !ok {
				continue
			}
			name, _ := sMap["name"].(string)
			if name != "" {
				secretNames = append(secretNames, name)
			}
		}
		if len(secretNames) > 0 {
			extras[ExtrasKeySecrets] = strings.Join(secretNames, ",")
		}
	}
	// 提取 automountServiceAccountToken
	automount, found, _ := unstructured.NestedBool(obj.Object, "automountServiceAccountToken")
	if found {
		extras[ExtrasKeyAutomountToken] = fmt.Sprintf("%t", automount)
	}
	return extras
}

// extractFirstContainerImage 从指定路径的 containers 列表中提取第一个容器的镜像名
// containerPath 为 containers 字段在对象中的嵌套路径，如 "spec", "containers" 或 "spec", "template", "spec", "containers"
func extractFirstContainerImage(obj *unstructured.Unstructured, containerPath ...string) string {
	containers, found, _ := unstructured.NestedSlice(obj.Object, containerPath...)
	if !found || len(containers) == 0 {
		return ""
	}
	firstContainer, ok := containers[0].(map[string]any)
	if !ok {
		return ""
	}
	image, _ := firstContainer["image"].(string)
	return image
}

// extractReplicaInfo 从资源的 spec.replicas 和 status.readyReplicas 提取副本数信息并写入 extras
func extractReplicaInfo(obj *unstructured.Unstructured, extras map[string]string) {
	replicas, found, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	if found {
		extras[ExtrasKeyReplicas] = fmt.Sprintf("%d", replicas)
	}
	readyReplicas, found, _ := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")
	if found {
		extras[ExtrasKeyReadyReplicas] = fmt.Sprintf("%d", readyReplicas)
	}
}
