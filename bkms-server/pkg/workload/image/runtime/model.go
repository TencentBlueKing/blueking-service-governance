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

package runtime

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/distribution/reference"
	"github.com/pkg/errors"
)

var (
	// ErrRuntimeImageNotFound 运行时镜像记录不存在
	ErrRuntimeImageNotFound = errors.New("runtime image not found")

	// ErrRuntimeImageAlreadyExists 运行时镜像记录已存在
	ErrRuntimeImageAlreadyExists = errors.New("runtime image already exists")
)

// ImageType 平台运行时镜像类型
type ImageType string

const (
	// ImageTypeBuilder 构建镜像
	ImageTypeBuilder ImageType = "builder"
	// ImageTypeRunner 运行镜像
	ImageTypeRunner ImageType = "runner"

	// maxImageDescriptionLen 运行时镜像描述最大字符数
	maxImageDescriptionLen = 512
)

// Image 平台运行时镜像记录
type Image struct {
	// ID 记录 ID
	ID string `bson:"_id,omitempty"`
	// Type 镜像类型，取值为 builder 或 runner
	Type ImageType `bson:"type"`
	// Name 镜像仓库名称，不包含 tag 或 digest
	Name string `bson:"name"`
	// Description 描述
	Description string `bson:"description"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// ListOptions 运行时镜像查询选项
type ListOptions struct {
	// Type 镜像类型，为空时查询全部类型
	Type ImageType
	// Keyword 搜索关键字，按名称或描述模糊匹配
	Keyword string
}

// Validate 校验运行时镜像记录
func (i *Image) Validate() error {
	if i == nil {
		return errors.New("runtime image is nil")
	}
	if err := i.validateType(); err != nil {
		return err
	}
	if err := i.validateName(); err != nil {
		return err
	}
	if err := i.validateDescription(); err != nil {
		return err
	}
	return nil
}

// validateType 校验镜像类型
func (i *Image) validateType() error {
	switch i.Type {
	case ImageTypeBuilder, ImageTypeRunner:
		return nil
	default:
		return errors.Errorf("unsupported runtime image type: %s", i.Type)
	}
}

// validateName 校验镜像仓库名称
func (i *Image) validateName() error {
	// 先去除首尾空白，避免只配置空白字符的镜像名称通过校验
	name := strings.TrimSpace(i.Name)
	if name == "" {
		return errors.New("image name is required")
	}

	named, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return errors.Wrapf(err, "invalid image name %s", name)
	}

	// 镜像仓库名称只接受 repository，不允许携带 tag 或 digest
	if !reference.IsNameOnly(named) {
		return errors.Errorf("image name %s must not contain tag or digest", name)
	}
	return nil
}

// validateDescription 校验镜像描述长度
func (i *Image) validateDescription() error {
	if utf8.RuneCountInString(i.Description) > maxImageDescriptionLen {
		return errors.Errorf("image description must not exceed %d characters", maxImageDescriptionLen)
	}
	return nil
}
