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

package appspec

import (
	"os"
	"reflect"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/stringx"
)

// ParseYAMLFile reads and parses a YAML file into the target struct.
func ParseYAMLFile(filePath string, target any) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.Errorf("file not found: %s", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", filePath)
	}

	if len(data) == 0 {
		return errors.Errorf("file is empty: %s", filePath)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return errors.Wrapf(err, "failed to parse YAML file %s", filePath)
	}

	// 递归 TrimSpace 所有 string 字段
	stringx.TrimSpaceRecursive(reflect.ValueOf(target))

	return nil
}

// ParseResourcesFile parses a resources YAML file.
func ParseResourcesFile(filePath string) (*ResourcesInput, error) {
	var input ResourcesInput
	if err := ParseYAMLFile(filePath, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ParseUpdateStrategyFile parses an update-strategy YAML file.
func ParseUpdateStrategyFile(filePath string) (*UpdateStrategyInput, error) {
	var input UpdateStrategyInput
	if err := ParseYAMLFile(filePath, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ParseLifecycleFile parses a lifecycle YAML file.
func ParseLifecycleFile(filePath string) (*LifecycleInput, error) {
	var input LifecycleInput
	if err := ParseYAMLFile(filePath, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ParseProbeFile parses a probe YAML file.
func ParseProbeFile(filePath string) (*ProbeInput, error) {
	var input ProbeInput
	if err := ParseYAMLFile(filePath, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ParseLabelsFile parses a labels YAML file.
func ParseLabelsFile(filePath string) (*LabelsInput, error) {
	var input LabelsInput
	if err := ParseYAMLFile(filePath, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ParseAnnotationsFile parses an annotations YAML file.
func ParseAnnotationsFile(filePath string) (*AnnotationsInput, error) {
	var input AnnotationsInput
	if err := ParseYAMLFile(filePath, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ParseStartCommandFile parses a start-command YAML file.
func ParseStartCommandFile(filePath string) (*StartCommandInput, error) {
	var input StartCommandInput
	if err := ParseYAMLFile(filePath, &input); err != nil {
		return nil, err
	}
	return &input, nil
}
