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

package bkerrs

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

var errComponentsNotInstalled = Define(
	ErrCodeNotFound, ErrDetailCodeComponentNotInstalled, "components not installed in cluster",
)

// WrapComponentsNotInstalled 包装多个组件缺失的错误，每个组件生成独立的详情。
func WrapComponentsNotInstalled(err error, components []string, clusterID string) error {
	wrappedErr := errComponentsNotInstalled.
		With("components", strings.Join(components, ", ")).
		With("clusterID", clusterID).
		WithCause(err)
	details := lo.Map(components, func(component string, _ int) Detail {
		return NewDetail(
			ErrDetailCodeComponentNotInstalled,
			fmt.Sprintf("component %s is not installed in cluster %s, "+
				"please install it before using this feature", component, clusterID),
			WithSystem("bkms"),
			WithModule(component),
		)
	})
	return wrappedErr.SetDetails(details...)
}
