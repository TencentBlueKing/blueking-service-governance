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

package testutil

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

// YAMLEqual compares two YAML contents by unmarshaling them and checking if the resulting maps are equal.
// It returns true if both YAML contents represent the same data structure, false otherwise.
func YAMLEqual(yaml1, yaml2 string) (bool, error) {
	var map1, map2 map[string]any

	// Unmarshal first YAML
	if err := yaml.Unmarshal([]byte(yaml1), &map1); err != nil {
		return false, err
	}

	// Unmarshal second YAML
	if err := yaml.Unmarshal([]byte(yaml2), &map2); err != nil {
		return false, err
	}

	// Compare the maps using deep equality
	return reflect.DeepEqual(map1, map2), nil
}

// YAMLValueAt returns a value from YAML content by walking explicit path segments.
// String segments select map keys, and int segments select list indexes.
// Expressions, wildcards, filters, and dotted path strings are intentionally unsupported.
func YAMLValueAt(yamlContent string, path ...any) (any, error) {
	var cfg any
	if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
		return nil, err
	}

	return valueAtPath(cfg, path)
}

func valueAtPath(root any, path []any) (any, error) {
	current := root
	for _, segment := range path {
		switch segment := segment.(type) {
		case string:
			node, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("yaml path %v cannot access map key %q on %T", path, segment, current)
			}
			value, ok := node[segment]
			if !ok {
				return nil, fmt.Errorf("yaml path %v not found", path)
			}
			current = value
		case int:
			node, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("yaml path %v cannot access list index %d on %T", path, segment, current)
			}
			if segment < 0 || segment >= len(node) {
				return nil, fmt.Errorf("yaml path %v index %d out of range", path, segment)
			}
			current = node[segment]
		default:
			return nil, fmt.Errorf("yaml path segment %v has unsupported type %T", segment, segment)
		}
	}
	return current, nil
}

// IsSuperMap checks if 'sup' map contains all key-value pairs of 'sub' map.
func IsSuperMap(sup, sub map[string]any) bool {
	for key, subValue := range sub {
		supValue, exists := sup[key]
		if !exists {
			return false
		}
		if !reflect.DeepEqual(subValue, supValue) {
			return false
		}
	}
	return true
}
