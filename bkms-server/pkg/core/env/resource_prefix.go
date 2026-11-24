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

package env

import (
	"context"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/resname"
)

// resourcePrefixMaxAttempts 是共用命名空间特性环境分配 resourcePrefix 冲突时的最大重试次数。
const resourcePrefixMaxAttempts = 5

// validateShareSourceNamespace 校验 NsTypeShared 的前置条件：来源环境已绑定集群命名空间，
// 且带 resPrefix 前缀后，应用自身的资源名（主工作负载及其固定派生资源）不超过 K8s 长度上限。
func validateShareSourceNamespace(input CreateFeatureEnvInput) error {
	if input.SourceEnv.Cluster.ClusterID == "" || input.SourceEnv.Cluster.Namespace == "" {
		return errors.Wrapf(
			ErrInvalidFeatureEnvInput,
			"source environment %s has no cluster namespace to share", input.SourceEnv.Name,
		)
	}
	if err := resname.ValidateAppNameForSharedNS(input.App.Name); err != nil {
		return errors.Wrap(ErrInvalidFeatureEnvInput, err.Error())
	}
	return nil
}

// createEnvWithResourcePrefix 为复用来源环境 namespace 的特性环境分配 resourcePrefix 并落库。
// resourcePrefix 不能与来源环境的泳道名相同，否则 <lane>-<app> 与 <prefix>-<app> 会撞名；
// 与同 namespace 其他特性环境的 prefix 冲突时重新生成。
func (s *FeatureEnvService) createEnvWithResourcePrefix(
	ctx context.Context, env, sourceEnv *model.Environment,
) error {
	lanes, err := s.trafficManager.ListTrafficLanes(ctx, sourceEnv.WorkspaceID, sourceEnv.Name)
	if err != nil {
		return errors.Wrapf(err, "list traffic lanes of source environment %s", sourceEnv.Name)
	}
	laneNames := set.NewStringSet()
	for _, lane := range lanes {
		laneNames.Add(lane.LaneName)
	}

	for range resourcePrefixMaxAttempts {
		prefix, genErr := resname.GenEnvPrefix()
		if genErr != nil {
			return errors.Wrap(genErr, "generate feature environment resource prefix")
		}
		if laneNames.Has(prefix) {
			continue
		}
		env.NsSharing.ResPrefix = prefix
		err = s.createEnv(ctx, env)
		// 命名重复时重试
		if err == nil || !IsErrEnvClusterNamespaceOccupied(err) {
			return err
		}
	}
	return errors.Errorf(
		"allocate resource prefix for feature environment %s: no free prefix after %d attempts",
		env.Name, resourcePrefixMaxAttempts,
	)
}
