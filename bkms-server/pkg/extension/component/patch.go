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

package component

import (
	"encoding/json"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// ApplyGameDeploymentPatchers applies root patchers in array order using Strategic Merge Patch.
func ApplyGameDeploymentPatchers(
	gameDeployment tkex.GameDeployment,
	patchers []map[string]any,
) (tkex.GameDeployment, error) {
	current, err := json.Marshal(gameDeployment)
	if err != nil {
		return gameDeployment, errors.Wrap(err, "marshaling GameDeployment")
	}
	for index, patcher := range patchers {
		patch, marshalErr := json.Marshal(patcher)
		if marshalErr != nil {
			return gameDeployment, errors.Wrapf(marshalErr, "marshaling patcher[%d]", index)
		}
		current, err = strategicpatch.StrategicMergePatch(current, patch, gameDeployment)
		if err != nil {
			return gameDeployment, errors.Wrapf(err, "applying patcher[%d]", index)
		}
	}

	var result tkex.GameDeployment
	if err = json.Unmarshal(current, &result); err != nil {
		return gameDeployment, errors.Wrap(err, "unmarshaling patched GameDeployment")
	}
	return result, nil
}
