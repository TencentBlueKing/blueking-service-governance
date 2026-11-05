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

package image

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// RefreshImages 刷新应用镜像快照。服务端返回 failed 或未知状态（含空响应）时返回错误，
// 只有 success 与 refreshing（已有刷新在进行中）视为正常结果。
func RefreshImages(ctx context.Context, cli client.Client, appID string) (*client.RefreshAppImagesResult, error) {
	result, err := cli.RefreshAppImages(ctx, appID)
	if err != nil {
		return nil, err
	}

	switch result.Status {
	case client.RefreshAppImagesStatusSuccess, client.RefreshAppImagesStatusRefreshing:
		return result, nil
	case client.RefreshAppImagesStatusFailed:
		return nil, errors.Errorf("refresh app images failed: %s", result.Message)
	default:
		return nil, errors.Errorf(
			"refresh app images returned unexpected status %q: %s", result.Status, result.Message,
		)
	}
}
