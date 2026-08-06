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

package postrenderer

import (
	"bytes"
	"context"
	"slices"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/postrender"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	wlbscpcfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

// shouldInjectBscpKinds 待注入 BSCP sidecar 的 k8s 资源类型
// DaemonSet 也需要注入
var shouldInjectBscpKinds = []string{
	k8skind.DS,
	k8skind.STS,
	k8skind.Deploy,
	k8skind.GameSTS,
	k8skind.GameDeploy,
}

// 编译期接口实现检查
var _ postrender.PostRenderer = (*BscpPostRenderer)(nil)

// BscpPostRenderer 实现 Helm PostRenderer 接口
// 在 Helm 渲染 Manifest 后、提交到集群前，拦截目标工作负载并注入 BSCP sidecar
type BscpPostRenderer struct {
	fragment *wlbscpcfg.PodFragment
}

// NewBscpPostRenderer 创建 BSCP PostRenderer
// fragment 为 nil 时返回 nil（表示不需要 PostRenderer）
func NewBscpPostRenderer(fragment *wlbscpcfg.PodFragment) *BscpPostRenderer {
	if fragment == nil {
		return nil
	}
	return &BscpPostRenderer{fragment: fragment}
}

// NewBscpPostRendererFromStore 从 Store 获取 PodFragment 并创建 PostRenderer
// 未配置时返回 nil, nil
func NewBscpPostRendererFromStore(
	ctx context.Context,
	store bscpcfg.Store,
	appID, envName string,
) (*BscpPostRenderer, error) {
	fragment, err := wlbscpcfg.BuildFromStore(ctx, store, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "build bscp post renderer from store")
	}
	if fragment == nil {
		return nil, nil
	}
	return NewBscpPostRenderer(fragment), nil
}

// Run 实现 PostRenderer 接口
// 解析 multi-doc YAML，对目标工作负载注入 BSCP sidecar，其他资源原样返回
func (r *BscpPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	if r == nil || r.fragment == nil {
		return renderedManifests, nil
	}

	// 一次性解析所有 YAML 文档
	docs, err := parseMultiDocYAML(renderedManifests)
	if err != nil {
		return nil, errors.Wrap(err, "parse multi-doc YAML for bscp post renderer")
	}

	// 对目标工作负载执行注入
	injected := false
	for i, doc := range docs {
		kind, _ := doc["kind"].(string)
		if !slices.Contains(shouldInjectBscpKinds, kind) {
			continue
		}
		// 提取 metadata.name 进行匹配
		name := extractMetadataName(doc)
		if name != r.fragment.WorkloadName {
			continue
		}
		// 当 WorkloadKind 非空时，同时匹配资源的 kind（精确定位）
		if r.fragment.WorkloadKind != "" && kind != r.fragment.WorkloadKind {
			continue
		}
		if err := r.injectBscpIntoDoc(doc, kind); err != nil {
			return nil, errors.Wrapf(err, "inject bscp into %s/%s", kind, name)
		}
		docs[i] = doc
		injected = true
		// 提前退出，避免不必要的遍历
		break
	}

	if !injected {
		return nil, errors.Errorf(
			"bscp postrenderer: target workload %q not found in rendered manifests",
			r.fragment.WorkloadName,
		)
	}

	// 统一组装输出
	return assembleMultiDocYAML(docs)
}

// injectBscpIntoDoc 将 BSCP PodFragment 注入到单个工作负载文档中。
func (r *BscpPostRenderer) injectBscpIntoDoc(doc map[string]any, kind string) error {
	// 提取 spec.template.spec
	podSpecMap := extractPodSpecMap(doc)
	if podSpecMap == nil {
		return errors.Errorf("bscp postrenderer: workload %s has no spec.template.spec", kind)
	}

	// 默认以第一个容器，作为主容器名称
	containers, _ := podSpecMap["containers"].([]any)
	if len(containers) == 0 {
		return errors.Errorf("bscp postrenderer: workload %s has no containers", kind)
	}
	firstContainer, _ := containers[0].(map[string]any)
	mainContainerName, _ := firstContainer["name"].(string)

	// 使用原生 map 操作完成注入
	if err := wlbscpcfg.MergePodSpecMap(podSpecMap, r.fragment, mainContainerName); err != nil {
		return errors.Wrapf(err, "merge pod spec map for %s", kind)
	}

	return nil
}

// extractPodSpecMap 从 doc 中提取 spec.template.spec 对应的 map
// 如果路径不存在返回 nil
func extractPodSpecMap(doc map[string]any) map[string]any {
	spec, ok := doc["spec"].(map[string]any)
	if !ok {
		return nil
	}
	template, ok := spec["template"].(map[string]any)
	if !ok {
		return nil
	}
	podSpec, ok := template["spec"].(map[string]any)
	if !ok {
		return nil
	}
	return podSpec
}

// extractMetadataName 从 doc 中提取 metadata.name 字段值
func extractMetadataName(doc map[string]any) string {
	metadata, ok := doc["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	name, _ := metadata["name"].(string)
	return name
}
