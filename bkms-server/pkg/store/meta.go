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

// Package store 提供基于泛型的通用 MongoDB Store。
//
// 只需要通用 CRUD 的数据模型无需再定义自己的 Store，只要通过 Meta 描述集合名、
// 业务错误与读写钩子，再通过 Of(db, meta) 即可获得 GenericStore[T]。
// 过滤、排序、更新中的字段名既可以写 bson 字段名，也可以写 Go 字段名，嵌套字段使用 "a.b" 形式；
// 两种写法等价，代码中优先使用 bson 字段名。
package store

// Meta 描述一个数据模型的存储元信息
//
// 三类钩子的分工（每类可以有多个，按顺序执行，任一失败即中止）：
//   - BeforeSave 作用于调用方传入的对象，用于补默认值、校验等，修改对调用方可见；
//     仅在写入完整文档时执行（Create / CreateMany / Replace，以及传入 *T 的 Upsert）
//   - Encode 作用于对象的深拷贝，用于加密、序列化等不应泄露给调用方的转换；
//     在所有写入前执行，Update / Upsert 传入 map 时作用于由 map 构造出的对象，其余字段均为零值，
//     最终只写入 map 中的字段。因此 Encode 必须逐字段独立转换、能处理零值：
//     不做校验（交给 BeforeSave），也不依据其它字段决定如何转换（如按 isSensitive 决定是否加密 value），
//     否则局部更新时其它字段为零值会得到错误的结果
//   - Decode 作用于从数据库读出的每一个对象（Get / List），通常是 Encode 的逆操作
//
// 敏感字段的加解密可借助 crypto.OnDemand 写成一行钩子，见 pkg/workload/helmcore/credential。
type Meta[T any] struct {
	// Collection 集合名，必填
	Collection string
	// NotFound 文档不存在时返回的业务错误，返回的 error 同时满足 errors.Is(err, ErrNotFound)
	NotFound error
	// Duplicate 唯一键冲突时返回的业务错误，返回的 error 同时满足 errors.Is(err, ErrDuplicate)
	Duplicate error

	BeforeSave []func(*T) error
	Encode     []func(*T) error
	Decode     []func(*T) error
}
