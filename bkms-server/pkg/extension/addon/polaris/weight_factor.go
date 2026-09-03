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

const (
	// 开启动态权重时，写入北极星服务的相关 metadata
	polarisMetaEnableDynamicWeight = "internal-enable-dynamic-weight"
	polarisMetaDynamicWeightConfig = "internal-dynamic-weight-config"
)

// 开启权重因子时写入北极星服务 metadata 的固定公式
const defaultDynamicWeightConfigJSON = `{"func":"linear","params":{"a":1,"b":1,"min":-1,"max":1.5}}`

func weightFactorMetadata() map[string]string {
	return map[string]string{
		polarisMetaEnableDynamicWeight: "true",
		polarisMetaDynamicWeightConfig: defaultDynamicWeightConfigJSON,
	}
}

func weightFactorMetadataKeys() []string {
	return []string{polarisMetaEnableDynamicWeight, polarisMetaDynamicWeightConfig}
}
