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
	"crypto/rand"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// 背景：通过增加 resourceKey 前缀，确保特性环境与来源环境共存于同一命名空间时，资源命名不冲突。
// 目前仅用于特性环境，后续考虑拓展到所有标准环境，即标准环境也可以复用命名空间。
// 共用命名空间的特性环境 resourceKey 规则：固定前缀 + 随机串，生成冲突时有限次重试。
const (
	// 共用命名空间时，附加命名以字母作前缀，确保满足 k8s 资源命名规则
	// Q: 为什么不使用后缀
	// A: 目前大部分派生资源名是在主工作负载名后面拼固定后缀得到的,
	//    如 <workload>-plain-cfg、<workload>-plain-direct、<workload>-devmode-scripts
	//    如果再加后缀，容易导致后缀拼接顺序不一致。
	//    同时外部配日志采集、监控时通常以前缀过滤，这里也用前缀能避免规则误伤到不同特性环境。
	resourceKeyPrefix      = "f"
	resourceKeyRandLen     = 4
	resourceKeyCharset     = "abcdefghijklmnopqrstuvwxyz0123456789"
	resourceKeyMaxAttempts = 5
)

// longestDerivedNameSuffix 是从工作负载名派生的资源名中最长的固定后缀（开发模式 ConfigMap）。
// 北极星资源名更长，但取决于后续加入生效范围的配置，由北极星配置保存时另行校验。
const longestDerivedNameSuffix = "-devmode-scripts"

// validateShareSourceNamespace 校验复用来源环境 namespace 的前置条件：来源环境已绑定集群命名空间，
// 且带 resourceKey 前缀后，应用自身的资源名（主工作负载及其派生资源）不超过 K8s 长度上限。
func validateShareSourceNamespace(input CreateFeatureEnvInput) error {
	if input.SourceEnv.Cluster.ClusterID == "" || input.SourceEnv.Cluster.Namespace == "" {
		return errors.Wrapf(
			ErrInvalidFeatureEnvInput,
			"source environment %s has no cluster namespace to share", input.SourceEnv.Name,
		)
	}
	keyLen := len(resourceKeyPrefix) + resourceKeyRandLen
	nameLen := keyLen + len("-") + len(input.App.Name) + len(longestDerivedNameSuffix)
	// Service 名受 DNS-1035 label 约束，是派生资源中最严格的长度上限
	if nameLen > validation.DNS1035LabelMaxLength {
		return errors.Wrapf(
			ErrInvalidFeatureEnvInput,
			"app name %s is too long to share namespace: prefixed resource names would reach %d characters (max %d)",
			input.App.Name, nameLen, validation.DNS1035LabelMaxLength,
		)
	}
	return nil
}

// createEnvWithResourceKey 为复用来源环境 namespace 的特性环境分配 resourceKey 并落库。
// resourceKey 不能与来源环境的泳道名相同，否则 <lane>-<app> 与 <key>-<app> 会撞名；
// 与同 namespace 其他特性环境的 key 冲突时重新生成。
func (s *FeatureEnvService) createEnvWithResourceKey(
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

	for range resourceKeyMaxAttempts {
		key, genErr := GenResourceKey()
		if genErr != nil {
			return errors.Wrap(genErr, "generate feature environment resource key")
		}
		if laneNames.Has(key) {
			continue
		}
		env.ResourceKey = key
		err = s.createEnv(ctx, env)
		// 命名重复时重试
		if err == nil || !IsErrEnvClusterNamespaceOccupied(err) {
			return err
		}
	}
	return errors.Errorf(
		"allocate resource key for feature environment %s: no free key after %d attempts",
		env.Name, resourceKeyMaxAttempts,
	)
}

// GenResourceKey 生成 "f" + 4 位 [a-z0-9] 的随机 resourceKey。
func GenResourceKey() (string, error) {
	buf := make([]byte, resourceKeyRandLen)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.Wrap(err, "read random bytes")
	}
	for i, b := range buf {
		buf[i] = resourceKeyCharset[int(b)%len(resourceKeyCharset)]
	}
	return resourceKeyPrefix + string(buf), nil
}
