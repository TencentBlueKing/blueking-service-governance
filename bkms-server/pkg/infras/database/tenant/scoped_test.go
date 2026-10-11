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

package tenant

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type sampleStore interface {
	Scopable[sampleStore]
	collection() *Collection
}

type sampleStoreImpl struct {
	Base[sampleStore]
}

func (s *sampleStoreImpl) collection() *Collection {
	return s.Collection()
}

func newSampleStore(coll *Collection) sampleStore {
	wrap := func(b Base[sampleStore]) sampleStore {
		return &sampleStoreImpl{Base: b}
	}
	return &sampleStoreImpl{Base: NewBase(coll, wrap)}
}

var _ = Describe("Base", func() {
	It("panics when wrap is nil", func() {
		Expect(func() { NewBase[sampleStore](newTestCollection("workspaces"), nil) }).To(Panic())
	})

	It("keeps the default tenant scope without Scopes", func() {
		store := newSampleStore(newTestCollection("workspaces"))
		Expect(store.collection().scope).To(Equal(scopeTenant))
		Expect(store.collection().specifiedTenantID).To(BeNil())
	})

	It("returns a new store with the scope applied and leaves the original unchanged", func() {
		store := newSampleStore(newTestCollection("workspaces"))

		scoped := store.Scopes(AllTenants())

		Expect(scoped.collection().scope).To(Equal(scopeCrossTenant))
		Expect(store.collection().scope).To(Equal(scopeTenant))
	})

	It("applies scopes in order so the later one wins", func() {
		store := newSampleStore(newTestCollection("workspaces"))

		scoped := store.Scopes(AllTenants(), WithTenant("tenant-b"))

		Expect(scoped.collection().scope).To(Equal(scopeTenant))
		Expect(*scoped.collection().specifiedTenantID).To(Equal("tenant-b"))
	})

	It("supports chained Scopes calls", func() {
		store := newSampleStore(newTestCollection("workspaces"))

		scoped := store.Scopes(WithTenant("tenant-b")).Scopes(AllTenants())

		Expect(scoped.collection().scope).To(Equal(scopeCrossTenant))
		Expect(scoped.collection().specifiedTenantID).To(BeNil())
	})

	It("skips nil scopes", func() {
		store := newSampleStore(newTestCollection("workspaces"))

		scoped := store.Scopes(nil, WithTenant("tenant-b"), nil)

		Expect(*scoped.collection().specifiedTenantID).To(Equal("tenant-b"))
	})
})
