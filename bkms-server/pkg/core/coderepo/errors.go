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

package coderepo

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

// 代码库错误，由 ToBkErr 转为接口错误
var (
	// ErrSourceTypeNotFound 接入方式不存在或未启用
	ErrSourceTypeNotFound = errors.New("code repository source type not found")
	// ErrRepositoryNotFound 代码库不存在
	ErrRepositoryNotFound = errors.New("code repository not found")
	// ErrAliasRequired 蓝盾托管方式未传别名
	ErrAliasRequired = errors.New("code repository alias is required")
	// ErrAliasInvalid 无法由仓库地址生成别名
	ErrAliasInvalid = errors.New("cannot derive code repository alias from url")
	// ErrCredentialRequired 缺少仓库账号或密码
	ErrCredentialRequired = errors.New("repository username and password are required")
	// ErrPasswordRequired 修改账号时未填写密码
	ErrPasswordRequired = errors.New("repository password is required when username changes")
	// ErrCredentialInvalid 账号密码错误或无权访问
	ErrCredentialInvalid = errors.New("repository username or password is incorrect")
	// ErrUnreachable 无法访问代码库
	ErrUnreachable = errors.New("repository is unreachable")
	// ErrUnsupported 接入方式不支持该操作
	ErrUnsupported = errors.New("operation is not supported by the source type")
)

// ToBkErr 转为接口错误：用户可修正的返回可读提示，其余为内部错误
func ToBkErr(err error, action string) error {
	switch {
	case errors.Is(err, ErrSourceTypeNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "不支持该代码库类型")
	case errors.Is(err, ErrAliasRequired):
		return bkerrs.New(bkerrs.ErrCodeInvalidRequest, "代码库别名不能为空")
	case errors.Is(err, ErrAliasInvalid):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "无法由仓库地址生成代码库别名，请检查仓库地址")
	case errors.Is(err, ErrCredentialRequired):
		return bkerrs.New(bkerrs.ErrCodeInvalidRequest, "首次关联该代码库需要填写仓库账号和仓库密码")
	case errors.Is(err, ErrPasswordRequired):
		return bkerrs.New(bkerrs.ErrCodeInvalidRequest, "修改仓库账号时需要重新填写仓库密码")
	case errors.Is(err, ErrCredentialInvalid):
		return bkerrs.New(bkerrs.ErrCodeInvalidRequest, "仓库账号或密码不正确，无法访问该代码库")
	case errors.Is(err, ErrUnreachable):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "无法访问该代码库，请确认填写的是可访问的 HTTP(S) 仓库地址")
	case errors.Is(err, ErrUnsupported):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "该代码库类型不支持此操作")
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, action)
	}
}
