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

// Package customruntime 定义工作空间维度的自定义运行时镜像记录。
//
// 它与平台官方镜像（pkg/workload/image/runtime）刻意保持独立：官方镜像无工作空间
// 维度、带描述且由平台维护，自定义镜像归属某个工作空间、无描述且由用户在构建配置
// 中隐式添加。实体类型不共用，避免官方镜像被自定义镜像的演进牵连；builder / runner
// 枚举与官方镜像相同，因此 ImageType 直接复用 runtime 包的定义。
//
// 本包只存镜像仓库名（不含 tag），不存 tag：可用 tag 属于快照数据，由
// pkg/workload/image/snapshot 按 repoKey 维护，本包在新镜像入库时触发它初始化一次。
//
// 记录的产生是隐式的：用户在构建配置里填了一个落在本工作空间生效镜像源路径下的镜像，
// 保存成功后由 PersistManager 补登记，用于下次填写时作为候选项展示，没有独立的增删接口。
// 三个环节分别是 existence.go 判定归属与存在性、store.go 落库、persist.go 编排两者。
package customruntime

import (
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/pkg/errors"

	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
)

// ErrCustomRuntimeImageNotFound 自定义运行时镜像记录不存在
var ErrCustomRuntimeImageNotFound = errors.New("custom runtime image not found")

// ImageType 镜像类型，与平台官方镜像共用同一套 builder / runner 枚举
type ImageType = workloadruntime.ImageType

const (
	// ImageTypeBuilder 构建镜像
	ImageTypeBuilder = workloadruntime.ImageTypeBuilder
	// ImageTypeRunner 运行镜像
	ImageTypeRunner = workloadruntime.ImageTypeRunner
)

// Image 工作空间自定义运行时镜像记录。
//
// 记录归属工作空间而非应用，因此不含 appID；同一镜像同时用作 builder 与 runner 时
// 落两条记录，由 Type 区分。出于安全考虑，本实体不保存任何账密字段，访问镜像仓库
// 所需的凭证运行时按 WorkspaceID 反查 image_registries 获取。
type Image struct {
	// ID 记录 ID
	ID string `bson:"_id,omitempty"`
	// WorkspaceID 所属工作空间，官方镜像无此维度
	WorkspaceID string `bson:"workspaceID"`
	// Type 镜像类型，取值为 builder 或 runner，决定该镜像出现在构建配置的哪个字段的候选中
	Type ImageType `bson:"type"`
	// Name 镜像仓库名称，存含仓库前缀的完整地址且不包含 tag 或 digest，
	// 例如 docker.bkrepo.example.com/demo/repo/my-golang
	Name string `bson:"name"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// ListOptions 自定义运行时镜像查询选项
type ListOptions struct {
	// Type 镜像类型，为空时查询全部类型
	Type ImageType
	// Keyword 搜索关键字，按名称模糊匹配
	Keyword string
}

// Validate 校验自定义运行时镜像记录
func (i *Image) Validate() error {
	if i == nil {
		return errors.New("custom runtime image is nil")
	}
	// 按落库唯一键顺序校验：先归属空间，再类型，最后名称格式
	if err := i.validateWorkspaceID(); err != nil {
		return err
	}
	if err := i.validateType(); err != nil {
		return err
	}
	return i.validateName()
}

// validateWorkspaceID 校验所属工作空间
func (i *Image) validateWorkspaceID() error {
	// 只拒绝空白，不在这里改写原值；落库时由 Upsert 再 trim，保证查询与写入口径一致
	if strings.TrimSpace(i.WorkspaceID) == "" {
		return errors.New("workspaceID is required")
	}
	return nil
}

// validateType 校验镜像类型
func (i *Image) validateType() error {
	// 只接受 builder / runner，空值与未知枚举一律拒绝
	switch i.Type {
	case ImageTypeBuilder, ImageTypeRunner:
		return nil
	default:
		return errors.Errorf("unsupported custom runtime image type: %s", i.Type)
	}
}

// validateName 校验镜像仓库名称
func (i *Image) validateName() error {
	return ValidateRepositoryName(i.Name)
}

// ValidateRepositoryName 校验自定义镜像仓库名称
//
// 必须在原始输入里写出仓库 host（例如 docker.bkrepo.example.com/proj/repo），且不得携带 tag 或 digest
// ParseNormalizedNamed 会把 nginx 补成 docker.io/library/nginx，因此还要对照规范化后的 Domain 与输入第一段
func ValidateRepositoryName(name string) error {
	// 先去除首尾空白，避免只配置空白字符的镜像名称通过校验
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("image name is required")
	}

	named, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return errors.Wrapf(err, "invalid image name %s", name)
	}

	// IsNameOnly 为 false 表示带了 tag 或 digest，与「name 只存仓库地址」的契约冲突
	if !reference.IsNameOnly(named) {
		return errors.Errorf("image name %s must not contain tag or digest", name)
	}

	// 短名规范化后 Domain 是 docker.io，但输入第一段对不上，据此拒绝 nginx、library/nginx
	if !strings.EqualFold(nameRegistryHost(name), reference.Domain(named)) {
		return errors.Errorf("image name %s must include a registry host", name)
	}
	return nil
}

// nameRegistryHost 取镜像名第一段作为调用方写出的仓库 host，没有 '/' 则视为未写 host
func nameRegistryHost(name string) string {
	i := strings.IndexRune(name, '/')
	if i <= 0 {
		return ""
	}
	return name[:i]
}
