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

package networking

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking"
)

const (
	ServiceAppLabelKey          = "io.tencent.bkms.app"
	ServiceControllerLabelKey   = "io.tencent.bkms.controller"
	ServiceControllerLabelValue = "bkms-syncer"

	ServiceTrafficlaneEnabledAnnoKey = "io.tencent.bkms.traffic-lane-enabled"
)

// NewServiceSyncer 创建 serviceSyncer 实例
func NewServiceSyncer(env *envmodel.Environment) *serviceSyncer {
	return &serviceSyncer{
		client:    k8sclient.NewWithGVR(cluster.NewConfig(env.Cluster.ClusterID), gvr.SVC),
		namespace: env.Cluster.Namespace,
	}
}

// serviceSyncer service 同步器. 负责将应用的 services 部署(同步)到目标环境中.
// - 对于 Helm 应用: 仅处理用于额外声明的独立 Service, 不会处理 Helm Chart 内部的 Service
type serviceSyncer struct {
	client    *k8sclient.Client
	namespace string
}

// Sync 同步 services 到目标环境中
// NOTE: Sync 方法不会去检查 namespace 是否已存在, 由调用方保证, 否则会同步失败
func (s *serviceSyncer) Sync(ctx context.Context, appID string, services []networking.Service) error {
	if err := s.validate(appID, services); err != nil {
		return err
	}

	// 1. Upsert new services
	manifests := make(map[string]map[string]any)
	for _, svc := range services {
		manifest, err := s.genManifest(svc)
		if err != nil {
			return errors.Wrapf(err, "gen service: %s", svc.Name)
		}
		manifests[svc.Name] = manifest
	}

	for svcName, manifest := range manifests {
		_, err := s.client.Upsert(ctx, s.namespace, manifest, metav1.PatchOptions{})
		if err != nil {
			return errors.Wrapf(err, "create service: %s", svcName)
		}
	}

	// 2. 查询需要删除的 services, 并删除
	namesToDelete, err := s.listServicesToDelete(ctx, appID, services)
	if err != nil {
		return errors.Wrapf(err, "list services to delete")
	}
	if len(namesToDelete) == 0 {
		return nil
	}
	for _, name := range namesToDelete {
		err = s.client.Delete(ctx, s.namespace, name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "delete service: %s", name)
		}
	}

	return nil
}

func (s *serviceSyncer) genManifest(service networking.Service) (map[string]any, error) {
	ports := lo.Map(service.Ports, func(port networking.ServicePort, _ int) corev1.ServicePort {
		return corev1.ServicePort{
			Name:       port.Name,
			Port:       port.Port,
			Protocol:   corev1.Protocol(port.Protocol),
			TargetPort: intstr.Parse(port.TargetPort),
		}
	})

	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: service.Name,
			Annotations: map[string]string{
				ServiceTrafficlaneEnabledAnnoKey: fmt.Sprintf("%t", service.TrafficLaneEnabled),
			},
			Labels: map[string]string{
				ServiceAppLabelKey:        service.AppID,
				ServiceControllerLabelKey: ServiceControllerLabelValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: service.Selector,
			Ports:    ports,
		},
	}

	svcObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&svc)
	if err != nil {
		return nil, err
	}

	return svcObj, nil
}

func (s *serviceSyncer) listServicesToDelete(
	ctx context.Context,
	appID string,
	services []networking.Service,
) ([]string, error) {
	// 根据标签过滤出相关的 services(忽略非 bkms-syncer 创建的 service, 防止误删)
	selector := labels.SelectorFromSet(labels.Set{
		ServiceAppLabelKey:        appID,
		ServiceControllerLabelKey: ServiceControllerLabelValue,
	})

	listOptions := metav1.ListOptions{
		LabelSelector: selector.String(),
	}

	existingServices, err := s.client.List(ctx, s.namespace, listOptions)
	if err != nil {
		return nil, err
	}

	if len(existingServices.Items) == 0 {
		return nil, nil
	}

	existingNames := lo.Map(existingServices.Items, func(item unstructured.Unstructured, _ int) string {
		return item.GetName()
	})

	namesToDelete, _ := lo.Difference(existingNames, lo.Map(services, func(item networking.Service, _ int) string {
		return item.Name
	}))

	return namesToDelete, nil
}

func (s *serviceSyncer) validate(appID string, services []networking.Service) error {
	if len(services) == 0 {
		return nil
	}

	appIDs := lo.UniqMap(services, func(u networking.Service, _ int) string {
		return u.AppID
	})

	if len(appIDs) > 1 {
		return errors.Errorf("multiple appIDs found: %v", appIDs)
	}
	if appIDs[0] != appID {
		return errors.Errorf("appID mismatch: %s", appID)
	}

	return nil
}
