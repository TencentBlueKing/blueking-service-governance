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

package store

import (
	"fmt"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	// ErrNotFound 文档不存在，所有 GenericStore 的 NotFound 错误都满足 errors.Is(err, ErrNotFound)
	ErrNotFound = errors.New("document not found")
	// ErrDuplicate 唯一键冲突，所有 GenericStore 的冲突错误都满足 errors.Is(err, ErrDuplicate)
	ErrDuplicate = errors.New("duplicate key")
)

// IgnoreNotFound 在 err 是 NotFound 错误时返回 nil，否则原样返回 err，
// 用于「目标不存在即视为完成」的写操作，例如：
//
//	return store.IgnoreNotFound(records.Update(ctx, fields, ByID(id)))
func IgnoreNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// storeError 让同一个错误既能匹配通用哨兵错误，也能匹配 Meta 中的业务错误，
// 这样调用方可以按需选择 errors.Is(err, store.ErrNotFound) 或 errors.Is(err, ErrXxxNotFound)
type storeError struct {
	generic error
	domain  error
	detail  string
	cause   error
}

func (e *storeError) Error() string {
	msg := e.generic.Error()
	if e.domain != nil {
		msg = e.domain.Error()
	}
	msg += ": " + e.detail
	if e.cause != nil {
		msg += ": " + e.cause.Error()
	}
	return msg
}

func (e *storeError) Is(target error) bool {
	return target == e.generic || (e.domain != nil && errors.Is(e.domain, target))
}

func (e *storeError) Unwrap() error { return e.cause }

func (s *GenericStore[T]) notFound(q *query) error {
	return &storeError{
		generic: ErrNotFound,
		domain:  s.meta.NotFound,
		detail:  fmt.Sprintf("collection %s, filter %v", s.coll.Name(), q.filter()),
	}
}

// wrapWriteErr 将唯一键冲突转换为 Duplicate 错误，其余错误附带操作描述后返回
func (s *GenericStore[T]) wrapWriteErr(err error, op string) error {
	if mongo.IsDuplicateKeyError(err) {
		return &storeError{
			generic: ErrDuplicate,
			domain:  s.meta.Duplicate,
			detail:  fmt.Sprintf("%s %s", op, s.coll.Name()),
			cause:   err,
		}
	}
	return errors.Wrapf(err, "%s %s", op, s.coll.Name())
}
