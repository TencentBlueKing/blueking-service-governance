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

package polaris_test

import (
	"errors"

	"github.com/bytedance/mockey"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

func mockPolarisDiscoveryFailure() {
	mockey.Mock(cluster.NewConfig).Return(&cluster.Config{ClusterID: "test-cluster"}).Build()
	mockey.Mock(discovery.GetGroupVersionResource).Return(nil, errors.New("test discovery error")).Build()
}

// k8sServiceClient 返回访问 core/v1 Service 的客户端
func k8sServiceClient(clusterCfg *cluster.Config) (*k8sclient.Client, error) {
	gvr, err := discovery.GetGroupVersionResource(
		clusterCfg, k8skind.SVC, corev1.SchemeGroupVersion.String(),
	)
	if err != nil {
		return nil, err
	}
	return k8sclient.NewWithGVR(clusterCfg, *gvr), nil
}

func newTestConfig(
	appID, name string,
	scopeEnvNames []string,
	envStates map[string]polaris.PolarisEnvState,
) *polaris.PolarisConfig {
	return &polaris.PolarisConfig{
		AppID: appID,
		Name:  name,
		Properties: polaris.Properties{
			InstanceKey:      "k1",
			PolarisName:      "polaris-service",
			PolarisNamespace: "Test",
			PolarisToken:     "t1",
			ServicePort:      8080,
		},
		ScopeEnvNames: scopeEnvNames,
		EnvStates:     envStates,
	}
}

func redeployFields(instanceKey, token string, servicePort int32) *polaris.RedeployRequiredFields {
	return &polaris.RedeployRequiredFields{
		InstanceKey:  instanceKey,
		PolarisToken: token,
		ServicePort:  servicePort,
	}
}

func envState(appliedFields *polaris.RedeployRequiredFields) polaris.PolarisEnvState {
	return polaris.PolarisEnvState{
		AppliedFields: appliedFields,
	}
}
