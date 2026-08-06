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

package serializer

import (
	"strings"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// AppSpecAnnotationsInput is the input structure of the annotations section.
type AppSpecAnnotationsInput struct {
	// 自定义注解，key 需为合法的 Kubernetes annotation key（qualified name），
	// value 无格式与长度限制。
	Annotations map[string]string `json:"annotations"`
}

// ToModel converts input to an AppSpec annotations section.
func (i *AppSpecAnnotationsInput) ToModel() *appspec.AnnotationsSpec {
	if i == nil {
		return nil
	}
	cleaned := make(map[string]string, len(i.Annotations))
	for k, v := range i.Annotations {
		cleaned[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return &appspec.AnnotationsSpec{Annotations: cleaned}
}

// AppSpecAnnotationsOutput is the JSON representation of the annotations section.
type AppSpecAnnotationsOutput struct {
	// 自定义注解 key/value 映射
	Annotations map[string]string `json:"annotations"`
}

// FromModel fills output fields from an AppSpec annotations section.
func (o *AppSpecAnnotationsOutput) FromModel(spec *appspec.AnnotationsSpec) *AppSpecAnnotationsOutput {
	if spec == nil {
		return nil
	}
	*o = AppSpecAnnotationsOutput{Annotations: spec.Annotations}
	return o
}

// AppSpecAnnotationsSectionOutput is the JSON response for querying annotations.
type AppSpecAnnotationsSectionOutput struct {
	Data *AppSpecAnnotationsOutput `json:"data"`
}
