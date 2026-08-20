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

package polaris

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

const (
	polarisConfigCRKind       = "PolarisConfig"
	polarisConfigCRAPIVersion = "tkex.tencent.com/v1"
)

// WorkloadResult contains the updated pod spec and Polaris extra resources.
type WorkloadResult struct {
	PodSpec      corev1.PodSpec
	ExtraObjects []unstructured.Unstructured
}

// WorkloadBuilder applies environment-specific Polaris configs to a workload payload.
type WorkloadBuilder struct {
	store PolarisConfigStore
}

// NewWorkloadBuilder creates a Polaris workload builder.
func NewWorkloadBuilder(store PolarisConfigStore) *WorkloadBuilder {
	return &WorkloadBuilder{store: store}
}

// Build builds polaris related extra resources.
// Only ServiceLabels participate in env-var rendering and undefined-reference collection.
func (b *WorkloadBuilder) Build(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	vars map[string]string,
	podSpec corev1.PodSpec,
	collector *envvarrefs.Collector,
) (*WorkloadResult, error) {
	configs, err := b.store.ListByEnv(ctx, app.ID, env.Name)
	if err != nil {
		return nil, fmt.Errorf("list polaris configs: %w", err)
	}

	result := &WorkloadResult{
		PodSpec:      *podSpec.DeepCopy(),
		ExtraObjects: make([]unstructured.Unstructured, 0, len(configs)*2),
	}
	for _, cfg := range configs {
		objects, buildErr := buildExtraResources(app, env, cfg, vars, collector)
		if buildErr != nil {
			return nil, fmt.Errorf("build resources for polaris config %s: %w", cfg.Name, buildErr)
		}
		result.ExtraObjects = append(result.ExtraObjects, objects...)
	}
	return result, nil
}

// BuildExtraResources 构造单个 PolarisConfig 对应的额外 K8s 资源（PolarisConfig CR + Service）
// Args:
// - app: 目标应用
// - env: 目标环境
// - cfg: 北极星配置
// - vars: 渲染 serviceLabels 时的变量上下文
// - collector: 收集未定义环境变量引用，可为 nil
// Returns:
// - []unstructured.Unstructured: 额外资源列表（PolarisConfig CR + Service）
// - error: 错误
func buildExtraResources(
	app *bkmsapp.Application,
	env *envmodel.Environment,
	cfg *PolarisConfig,
	vars map[string]string,
	collector *envvarrefs.Collector,
) ([]unstructured.Unstructured, error) {
	crName, serviceName := PolarisResourceNames(app.Name, cfg.Name)

	serviceSpec := map[string]any{
		"name":              serviceName,
		"namespace":         env.Cluster.Namespace,
		"port":              int64(cfg.ServicePort),
		"direct":            cfg.Direct,
		"keepNotReadyPod":   cfg.KeepNotReadyPod,
		"enableHealthCheck": cfg.EnableHealthCheck,
		"weight":            int64(cfg.GetEnvWeight(env.Name)),
	}
	if len(cfg.ServiceLabels) > 0 {
		extraMeta, err := renderServiceLabels(cfg.Name, cfg.ServiceLabels, vars, collector)
		if err != nil {
			return nil, fmt.Errorf("render service labels for polaris config %s: %w", cfg.Name, err)
		}
		serviceSpec["extraMeta"] = extraMeta
	}

	crMap := map[string]any{
		"apiVersion": polarisConfigCRAPIVersion,
		"kind":       polarisConfigCRKind,
		"metadata": map[string]any{
			"name": crName,
		},
		"spec": map[string]any{
			"polaris": map[string]any{
				"name":      cfg.PolarisName,
				"namespace": cfg.PolarisNamespace,
				"token":     cfg.PolarisToken,
			},
			"services": []any{serviceSpec},
		},
	}

	crConverted, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&crMap)
	if err != nil {
		return nil, fmt.Errorf("convert polaris config CR to unstructured: %w", err)
	}

	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       k8skind.SVC,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": app.Name},
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       cfg.ServicePort,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: cfg.ServicePort},
			}},
		},
	}
	serviceMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&service)
	if err != nil {
		return nil, fmt.Errorf("convert polaris service to unstructured: %w", err)
	}

	return []unstructured.Unstructured{
		{Object: crConverted},
		{Object: serviceMap},
	}, nil
}

// PolarisResourceNames 返回该配置在集群中的 PolarisConfig CR 名与配套 Service 名。
func PolarisResourceNames(appName, configName string) (crName, serviceName string) {
	baseName := strings.ToLower(fmt.Sprintf("%s-%s", appName, configName))
	return baseName + "-polaris", baseName + "-polaris-service"
}

// renderServiceLabels 渲染 serviceLabels 中的 ${{env.X}} 变量，并收集未定义的环境变量引用。
func renderServiceLabels(
	sourceName string,
	labels, vars map[string]string,
	collector *envvarrefs.Collector,
) (map[string]any, error) {
	renderer := render.New(render.SetEnvContext(vars))
	result := make(map[string]any, len(labels))
	for key, value := range labels {
		if err := collector.Collect(value, envvarrefs.Source{
			Type: envvarrefs.SourcePolaris,
			Name: sourceName,
		}); err != nil {
			return nil, fmt.Errorf("collect env vars from service label %s: %w", key, err)
		}
		rendered, err := renderer.Render(value)
		if err != nil {
			return nil, fmt.Errorf("render service label %s: %w", key, err)
		}
		result[key] = rendered
	}
	return result, nil
}
