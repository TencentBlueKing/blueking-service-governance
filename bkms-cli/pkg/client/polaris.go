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
	AppID string `json:"appID" yaml:"appID" table:"-"`
	// 配置名称（应用内唯一）
	Name string `json:"name" yaml:"name"`
	// 关联的依赖服务实例 ID（平台创建时有值）
	DepSvcInstID string `json:"depSvcInstID" yaml:"depSvcInstID" table:"-"`
	// 组件实例标识，用于环境变量拼接
	InstanceKey string `json:"instanceKey" yaml:"instanceKey"`
	// 北极星服务名称
	PolarisName string `json:"polarisName" yaml:"polarisName"`
	// 北极星命名空间
	PolarisNamespace string `json:"polarisNamespace" yaml:"polarisNamespace"`
	// 北极星 Token（返回时脱敏）
	PolarisToken string `json:"polarisToken" yaml:"polarisToken"`
	// 服务端口
	ServicePort int32 `json:"servicePort" yaml:"servicePort"`
	// 是否为直连模式（注册 Pod IP 到北极星）
	Direct bool `json:"direct" yaml:"direct"`
	// 是否保留未就绪的 Pod 在北极星
	KeepNotReadyPod bool `json:"keepNotReadyPod" yaml:"keepNotReadyPod"`
	// 是否启用健康检查
	EnableHealthCheck bool `json:"enableHealthCheck" yaml:"enableHealthCheck"`
	// 服务标签
	ServiceLabels map[string]string `json:"serviceLabels" yaml:"serviceLabels" table:"-"`
	// 生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames" yaml:"scopeEnvNames"`
	// 负责人
	Operator string `json:"operator" yaml:"operator"`
	// 创建时间
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
	// 校验警告信息
	Warnings []string `json:"warnings" yaml:"warnings" table:"-"`
	// 各环境中已经生效的关键字段、下发错误和部署状态
	EnvStates map[string]PolarisEnvState `json:"envStates" yaml:"envStates" table:"-"`
	// 各环境的单实例权重，key 为环境名称；未设置时后端默认 100
	EnvWeights map[string]int32 `json:"envWeights" yaml:"envWeights" table:"-"`
}

// PolarisEnvState 单个环境的部署快照和最近一次动态下发错误
type PolarisEnvState struct {
	// 集群中已经生效的关键字段；nil 表示尚未完成首次应用部署
	AppliedFields *PolarisRedeployRequiredFields `json:"appliedFields" yaml:"appliedFields"`
	// 最近一次记录的动态下发错误
	LastError string `json:"lastError" yaml:"lastError"`
	// 环境信息最后更新时间
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
	// 当前配置期望 Token 是否不同于该环境最近一次部署快照
	PolarisTokenChanged bool `json:"polarisTokenChanged" yaml:"polarisTokenChanged"`
	// 部署状态: deployed / pendingCreate / pendingModify / pendingDelete
	Status string `json:"status" yaml:"status"`
}

// PolarisRedeployRequiredFields 需要重新部署才能生效的字段
type PolarisRedeployRequiredFields struct {
	InstanceKey  string `json:"instanceKey" yaml:"instanceKey"`
	PolarisToken string `json:"polarisToken" yaml:"polarisToken"`
	ServicePort  int32  `json:"servicePort" yaml:"servicePort"`
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
