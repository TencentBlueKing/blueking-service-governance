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

package devmode

import (
	"github.com/jinzhu/copier"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	componentdevmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
)

// allowedWorkPaths 允许的 WorkPath 值（根据应用类型不同）
var allowedWorkPaths = []string{
	componentdevmode.TrpcWorkPath,
	componentdevmode.TafWorkPath,
}

// allowedMountPaths 允许的 MountPath 值（根据应用类型不同）
var allowedMountPaths = []string{
	componentdevmode.TrpcMountPath,
	componentdevmode.TafMountPath,
}

// Spec stores dev mode settings.
type Spec struct {
	// Enabled 表示是否启用开发模式
	Enabled *bool `bson:"enabled,omitempty"`
	// WorkPath 表示工作根目录
	WorkPath *string `bson:"workPath,omitempty"`
	// MountPath 表示脚本文件挂载路径
	MountPath *string `bson:"mountPath,omitempty"`
}

// SetPathsByAppType 根据应用类型设置 WorkPath 和 MountPath。
// 不同类型的应用使用不同的开发模式路径，此方法统一处理路径赋值逻辑。
func (s *Spec) SetPathsByAppType(appType string) {
	switch appType {
	case bkmsapp.AppTypeTAF:
		s.WorkPath = lo.ToPtr(componentdevmode.TafWorkPath)
		s.MountPath = lo.ToPtr(componentdevmode.TafMountPath)
	case bkmsapp.AppTypeTRPC:
		s.WorkPath = lo.ToPtr(componentdevmode.TrpcWorkPath)
		s.MountPath = lo.ToPtr(componentdevmode.TrpcMountPath)
	}
}

// Clone deep-copies the section.
// 注意：WorkPath 和 MountPath 不在此处填充默认值，因为 spec 层不知道应用类型（trpc/taf）。
// 路径的设置由上层 devmode 组件（New / FromAppModel）根据应用类型负责。
func Clone(spec *Spec) *Spec {
	if spec == nil {
		return nil
	}

	cloned := new(Spec)
	_ = copier.CopyWithOption(cloned, spec, copier.Option{DeepCopy: true})
	if !HasData(cloned) {
		return nil
	}

	return cloned
}

// HasData returns whether the section carries any explicit configuration.
func HasData(spec *Spec) bool {
	return spec != nil && (spec.Enabled != nil || spec.WorkPath != nil || spec.MountPath != nil)
}

// Merge overlays non-nil values from override onto base.
func Merge(base, override *Spec) *Spec {
	switch {
	case base == nil && override == nil:
		return nil
	case base == nil:
		return Clone(override)
	case override == nil:
		return Clone(base)
	}

	merged := Clone(base)
	if override.Enabled != nil {
		merged.Enabled = override.Enabled
	}
	if override.WorkPath != nil {
		merged.WorkPath = override.WorkPath
	}
	if override.MountPath != nil {
		merged.MountPath = override.MountPath
	}
	return Clone(merged)
}

// AppendPatch adds MongoDB $set entries for this section.
func AppendPatch(set *bson.D, spec *Spec) {
	if spec == nil {
		return
	}
	if spec.Enabled != nil {
		*set = append(*set, bson.E{Key: "devMode.enabled", Value: spec.Enabled})
	}
	if spec.WorkPath != nil {
		*set = append(*set, bson.E{Key: "devMode.workPath", Value: spec.WorkPath})
	}
	if spec.MountPath != nil {
		*set = append(*set, bson.E{Key: "devMode.mountPath", Value: spec.MountPath})
	}
}
