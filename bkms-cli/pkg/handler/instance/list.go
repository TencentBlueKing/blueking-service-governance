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

package instance

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// ListInstancesOptions 查询实例列表参数。
type ListInstancesOptions struct {
	// Status 按实例状态过滤，如 Running。
	Status string
}

// ListInstances 获取应用在指定环境下的全部实例，并自动处理分页。
func ListInstances(
	ctx context.Context,
	cli client.Client,
	appID, envName string,
	opts ListInstancesOptions,
) ([]client.Instance, error) {
	var instances []client.Instance

	for page := 1; ; page++ {
		listOpts := client.ListAppInstancesOptions{
			Page:     page,
			PageSize: client.DefaultListAppInstancesPageSize,
		}

		resp, err := cli.ListAppInstances(ctx, appID, envName, listOpts)
		if err != nil {
			return nil, errors.Wrap(err, "list app instances")
		}
		if resp == nil {
			return nil, errors.Errorf("empty app instances for app %s env %s", appID, envName)
		}

		slog.Debug(
			"fetched app instances page",
			"appID", appID,
			"envName", envName,
			"page", listOpts.Page,
			"pageSize", listOpts.PageSize,
			"statusFilter", opts.Status,
			"resultsCount", len(resp.Results),
			"totalCount", resp.Count,
		)

		instances = append(instances, resp.Results...)
		if len(resp.Results) == 0 || len(instances) >= lo.Must(strconv.Atoi(resp.Count)) {
			break
		}
	}

	if opts.Status != "" {
		instances = lo.Filter(instances, func(instance client.Instance, _ int) bool {
			return instance.Status == opts.Status
		})
	}

	return instances, nil
}
