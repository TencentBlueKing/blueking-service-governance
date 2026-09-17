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
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RefKind 引用类型
type RefKind string

const (
	// RefKindBranch 分支
	RefKindBranch RefKind = "branch"
	// RefKindTag Tag
	RefKindTag RefKind = "tag"
)

// RefsQuery 分支 / Tag 查询参数
type RefsQuery struct {
	// Kind 引用类型
	Kind RefKind
	// Search 名称关键字，忽略大小写
	Search string
	// Page 页码，从 1 开始
	Page int64
	// PageSize 每页数量
	PageSize int64
}

// Ref 分支或 Tag
type Ref struct {
	// Name 名称，如 master、v1.0.0
	Name string
	// Path 完整路径，如 refs/heads/master
	Path string
	// SHA 指向的提交 ID
	SHA string
	// LinkURL 在代码库服务中的访问地址，不支持时为空
	LinkURL string
}

// Controller 访问代码库，各接入方式分别实现
type Controller interface {
	// Touch 校验能否访问
	Touch(ctx context.Context) error
	// ListRefs 分页列出分支或 Tag
	ListRefs(ctx context.Context, query RefsQuery) ([]Ref, error)
}

// ControllerFactory 创建 Controller，repo 可以尚未保存
type ControllerFactory func(ctx context.Context, db *mongo.Database, repo *CodeRepository) (Controller, error)

// controllerFactories spec -> Controller 工厂
var controllerFactories = map[string]ControllerFactory{}

// RegisterControllerFactory 注册 spec 的 Controller 工厂，在 init 中调用
func RegisterControllerFactory(spec string, factory ControllerFactory) {
	controllerFactories[spec] = factory
}

// NewController 按接入方式创建 Controller
func NewController(ctx context.Context, db *mongo.Database, repo *CodeRepository) (Controller, error) {
	sourceType, err := ResolveSourceType(repo.SourceType)
	if err != nil {
		return nil, err
	}
	factory, ok := controllerFactories[sourceType.Spec]
	if !ok {
		return nil, errors.Wrapf(ErrUnsupported, "no controller for spec %s", sourceType.Spec)
	}
	controller, err := factory(ctx, db, repo)
	if err != nil {
		return nil, errors.Wrapf(err, "create %s controller for repository %s", sourceType.Spec, repo.URL)
	}
	return controller, nil
}
