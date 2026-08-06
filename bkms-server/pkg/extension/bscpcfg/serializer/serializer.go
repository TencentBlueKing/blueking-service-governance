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

// Package serializer 定义应用配置管理 Gin v2 API 的请求/响应结构体。
package serializer

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// ValidWorkloadKinds 合法的 WorkloadKind 值列表
var ValidWorkloadKinds = []string{
	k8skind.Deploy,
	k8skind.STS,
	k8skind.DS,
	k8skind.GameDeploy,
	k8skind.GameSTS,
}

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterStructValidation(validatePatchMetadataInputStruct, PatchMetadataInput{})
	}
}

// validatePatchMetadataInputStruct 校验 PatchMetadataInput 中 WorkloadKind 字段值是否合法。
func validatePatchMetadataInputStruct(sl validator.StructLevel) {
	input := sl.Current().Interface().(PatchMetadataInput)
	if input.WorkloadKind == nil {
		return
	}
	kind := strings.TrimSpace(*input.WorkloadKind)
	if kind == "" {
		return
	}
	for _, v := range ValidWorkloadKinds {
		if kind == v {
			return
		}
	}
	sl.ReportError(
		*input.WorkloadKind,
		"WorkloadKind",
		"WorkloadKind",
		"workload_kind",
		strings.Join(ValidWorkloadKinds, "|"),
	)
}

// normalizeMountPath 规范化挂载路径（空值保持为空，非空则 filepath.Clean）。
func normalizeMountPath(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	return filepath.Clean(s)
}

func errBlankField(field string) error {
	return bkerrs.New(bkerrs.ErrCodeInvalidArgument, fmt.Sprintf("%s must not be blank", field))
}

func errNoField() error {
	return bkerrs.New(bkerrs.ErrCodeInvalidArgument, "at least one field must be provided")
}

// -------------------------------- URI Input --------------------------------

// AppIDURI 包含 appID 路径参数的 URI 绑定结构体。
type AppIDURI struct {
	AppID string `uri:"appID" binding:"required"`
}

// AppEnvURI 包含 appID 和 envName 路径参数的 URI 绑定结构体。
type AppEnvURI struct {
	AppID   string `uri:"appID" binding:"required"`
	EnvName string `uri:"envName" binding:"required"`
}

// -------------------------------- JSON Input --------------------------------

// PatchMetadataInput 更新 Metadata 的请求体（PATCH 语义：传值则更新，不传则不变）。
type PatchMetadataInput struct {
	// MountPath 配置文件挂载路径（传入则更新）
	MountPath *string `json:"mountPath"`
	// WorkloadName 指定被注入 bscp 配置的目标 workload 名称（传入则更新）
	WorkloadName *string `json:"workloadName"`
	// WorkloadKind 目标工作负载类型（传入则更新）
	WorkloadKind *string `json:"workloadKind"`
}

// ToUpdateModel 校验输入并转换为 model.MetadataUpdate。
// 至少需要传入一个字段，否则返回错误。
func (input *PatchMetadataInput) ToUpdateModel() (*model.MetadataUpdate, error) {
	update := &model.MetadataUpdate{}
	hasUpdate := false

	if input.MountPath != nil {
		mp := normalizeMountPath(*input.MountPath)
		if mp == "" {
			return nil, errBlankField("mountPath")
		}
		update.MountPath = &mp
		hasUpdate = true
	}
	if input.WorkloadName != nil {
		wl := strings.TrimSpace(*input.WorkloadName)
		if wl == "" {
			return nil, errBlankField("workloadName")
		}
		update.WorkloadName = &wl
		hasUpdate = true
	}
	if input.WorkloadKind != nil {
		wk := strings.TrimSpace(*input.WorkloadKind)
		update.WorkloadKind = &wk
		hasUpdate = true
	}

	if !hasUpdate {
		return nil, errNoField()
	}
	return update, nil
}

// PatchEnvBindingInput 更新 EnvBinding的请求体。
type PatchEnvBindingInput struct {
	// Services 绑定的下发服务列表（全量替换）
	Services []ServiceRefInput `json:"apps"`
}

// ServiceRefInput 下发服务引用输入项。
type ServiceRefInput struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// -------------------------------- Output --------------------------------

// MetadataOutput Metadata 输出对象。
type MetadataOutput struct {
	AppID          string    `json:"appID"`
	BscpBizID      string    `json:"bscpBizID"`
	MountPath      string    `json:"mountPath"`
	WorkloadName   string    `json:"workloadName"`
	WorkloadKind   string    `json:"workloadKind"`
	CredentialName string    `json:"credentialName"`
	FeedAddr       string    `json:"feedAddr"`
	Token          string    `json:"token"`
	PostHookID     string    `json:"postHookID"`
	Operator       string    `json:"operator"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// MetadataResponse 包装 MetadataOutput 的响应结构。
type MetadataResponse struct {
	Data *MetadataOutput `json:"data"`
}

// EnvBindingOutput EnvBinding输出对象。
type EnvBindingOutput struct {
	AppID            string              `json:"appID"`
	EnvName          string              `json:"envName"`
	BscpBizID        string              `json:"bscpBizID"`
	MountPath        string              `json:"mountPath"`
	WorkloadName     string              `json:"workloadName"`
	WorkloadKind     string              `json:"workloadKind"`
	Services         []*ServiceRefOutput `json:"apps"`
	FeedAddr         string              `json:"feedAddr"`
	Token            string              `json:"token"`
	DefaultFileAppID string              `json:"defaultFileAppID"`
	Operator         string              `json:"operator"`
	CreatedAt        time.Time           `json:"createdAt"`
	UpdatedAt        time.Time           `json:"updatedAt"`
}

// ServiceRefOutput 下发服务引用输出项。
type ServiceRefOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EnvBindingResponse 包装 EnvBindingOutput 的响应结构。
type EnvBindingResponse struct {
	Data *EnvBindingOutput `json:"data"`
}

// EnvBindingListResponse 包装 EnvBindingOutput 列表的响应结构。
type EnvBindingListResponse struct {
	Data []*EnvBindingOutput `json:"data"`
}

// -----------------------------------------------------------------------------
// Empty output
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// -------------------------------- Converter --------------------------------

// FromModel 将 model.Metadata 转换为 MetadataOutput。
func (o *MetadataOutput) FromModel(m *model.Metadata) *MetadataOutput {
	if o == nil {
		return nil
	}
	if m == nil {
		return o
	}
	o.AppID = m.AppID
	o.BscpBizID = m.BscpBizID
	o.MountPath = m.MountPath
	o.WorkloadName = m.WorkloadName
	o.WorkloadKind = m.WorkloadKind
	o.CredentialName = m.CredentialName
	o.FeedAddr = m.FeedAddr
	o.Token = m.Token
	o.PostHookID = m.PostHookID
	o.Operator = m.Operator
	o.CreatedAt = m.CreatedAt
	o.UpdatedAt = m.UpdatedAt
	return o
}

// FromModel 将 model.Snapshot 转换为 EnvBindingOutput。
func (o *EnvBindingOutput) FromModel(d *model.Snapshot) *EnvBindingOutput {
	if o == nil {
		return nil
	}
	if d == nil || d.EnvBinding == nil || d.Metadata == nil {
		return new(EnvBindingOutput)
	}
	o.AppID = d.EnvBinding.AppID
	o.EnvName = d.EnvBinding.EnvName
	o.BscpBizID = d.Metadata.BscpBizID
	o.MountPath = d.Metadata.MountPath
	o.WorkloadName = d.Metadata.WorkloadName
	o.WorkloadKind = d.Metadata.WorkloadKind
	o.FeedAddr = d.Metadata.FeedAddr
	o.Token = d.Metadata.Token
	o.DefaultFileAppID = d.EnvBinding.DefaultServiceID
	o.Operator = d.EnvBinding.Operator
	o.CreatedAt = d.EnvBinding.CreatedAt
	o.UpdatedAt = d.EnvBinding.UpdatedAt
	o.Services = lo.Map(d.EnvBinding.Services, func(svc model.ServiceRef, _ int) *ServiceRefOutput {
		return &ServiceRefOutput{
			ID:   svc.ID,
			Name: svc.Name,
		}
	})
	return o
}
