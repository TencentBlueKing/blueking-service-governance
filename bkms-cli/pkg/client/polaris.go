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

package client

// PolarisConfig 北极星配置
type PolarisConfig struct {
	// 所属应用 ID
	AppID string `json:"appID" yaml:"appID"`
	// 组件名称
	Name string `json:"name" yaml:"name"`
	// 关联的依赖服务实例 ID
	DepSvcInstID string `json:"depSvcInstID" yaml:"depSvcInstID"`
	// 组件实例标识，用于环境变量拼接
	InstanceKey string `json:"instanceKey" yaml:"instanceKey"`
	// 北极星实例名称
	PolarisName string `json:"polarisName" yaml:"polarisName"`
	// 北极星环境（命名空间）
	PolarisNamespace string `json:"polarisNamespace" yaml:"polarisNamespace"`
	// 北极星 Token
	PolarisToken string `json:"polarisToken" yaml:"polarisToken"`
	// 服务端口
	ServicePort int32 `json:"servicePort" yaml:"servicePort"`
	// 是否为直连模式
	Direct bool `json:"direct" yaml:"direct"`
	// 是否保留未就绪的 Pod 在北极星
	KeepNotReadyPod bool `json:"keepNotReadyPod" yaml:"keepNotReadyPod"`
	// 是否启用健康检查
	EnableHealthCheck bool `json:"enableHealthCheck" yaml:"enableHealthCheck"`
	// 服务权重
	Weight int32 `json:"weight" yaml:"weight"`
	// 服务标签
	ServiceLabels map[string]string `json:"serviceLabels" yaml:"serviceLabels"`
	// 组件生效范围类型
	ScopeType string `json:"scopeType" yaml:"scopeType"`
	// 组件生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames" yaml:"scopeEnvNames"`
	// 负责人
	Operator string `json:"operator" yaml:"operator"`
	// 创建时间
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
}

// ListPolarisConfigsRespData 获取北极星配置列表返回数据
type ListPolarisConfigsRespData struct {
	Data []PolarisConfig `json:"data"`
}

// CreatePolarisConfigRespData 创建北极星配置返回数据
type CreatePolarisConfigRespData struct {
	Data struct {
		Name string `json:"name"`
	} `json:"data"`
}
