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

// Package yaml provides utilities for working with YAML.
package yaml

import (
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnmarshalMultipleDocuments 从单个 yaml 中加载多个文档（通过 --- 分割）
func UnmarshalMultipleDocuments(data string) ([]map[string]any, error) {
	reader := strings.NewReader(data)
	decoder := yaml.NewDecoder(reader)

	var documents []map[string]any
	// 循环解码每个文档
	for {
		var doc map[string]any
		err := decoder.Decode(&doc)
		if err != nil {
			// 到达文件末尾时，结束解码
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// 将解码后的文档添加到结果列表中
		documents = append(documents, doc)
	}
	return documents, nil
}
