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

package helm

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
)

// CheckIfTrafficLaneDeployed 检查指定的流量泳道是否已部署
func CheckIfTrafficLaneDeployed(ctx context.Context, appID, envName, laneName string) error {
	store, err := NewRecordStoreMongo(database.Client(), database.Name())
	if err != nil {
		return errors.Wrapf(err, "get helm record store")
	}

	// 获取指定泳道的部署情况
	record, err := store.GetLatest(ctx, appID, envName, laneName)
	if err != nil {
		return errors.Wrapf(err, "get helm deploy record (app: %s, env: %s, lane: %s)", appID, envName, laneName)
	}
	// 如果指定泳道未部署，返回错误
	if record.Status != helm.StatusDeployed {
		return errors.Errorf("app %s env %s lane: %s not deployed", appID, envName, laneName)
	}
	return nil
}
