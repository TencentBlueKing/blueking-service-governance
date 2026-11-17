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

package bkerrs_test

import (
	"errors"
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

var (
	errTestItemDuplicated = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument, "TEST_ITEM_DUPLICATED", "item name duplicated",
	)
	errTestItemNotFound = bkerrs.Define(
		bkerrs.ErrCodeNotFound, "TEST_ITEM_NOT_FOUND", "item not found",
	)
	errTestDataInconsistent = bkerrs.Define(
		bkerrs.ErrCodeInternalServerError, "TEST_DATA_INCONSISTENT", "data inconsistent",
	)
)

var _ = Describe("Define", func() {
	It("should carry code and detail code", func() {
		Expect(errTestItemDuplicated.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
		Expect(errTestItemDuplicated.DetailCode()).To(Equal(bkerrs.ErrDetailCode("TEST_ITEM_DUPLICATED")))
		Expect(errTestItemDuplicated.Error()).To(Equal("item name duplicated"))
	})

	It("should panic on duplicated detail code", func() {
		Expect(func() {
			_ = bkerrs.Define(bkerrs.ErrCodeInvalidArgument, "TEST_ITEM_DUPLICATED", "again")
		}).To(PanicWith(ContainSubstring("TEST_ITEM_DUPLICATED is already defined")))
	})

	It("should panic on empty detail code", func() {
		Expect(func() {
			_ = bkerrs.Define(bkerrs.ErrCodeInvalidArgument, "", "no detail code")
		}).To(Panic())
	})

	It("should list definitions sorted by detail code", func() {
		detailCodes := []bkerrs.ErrDetailCode{}
		for _, def := range bkerrs.Definitions() {
			detailCodes = append(detailCodes, def.DetailCode())
		}
		Expect(detailCodes).To(ContainElements(
			bkerrs.ErrDetailCode("TEST_DATA_INCONSISTENT"),
			bkerrs.ErrDetailCode("TEST_ITEM_DUPLICATED"),
			bkerrs.ErrDetailCode("TEST_ITEM_NOT_FOUND"),
		))
		Expect(slices.IsSorted(detailCodes)).To(BeTrue())
	})
})

var _ = Describe("With", func() {
	It("should return a copy carrying params without modifying the definition", func() {
		err := errTestItemDuplicated.With("itemName", "a").With("env", "dev")

		Expect(err.Params()).To(Equal(map[string]string{"itemName": "a", "env": "dev"}))
		Expect(err.Error()).To(Equal("item name duplicated (env=dev, itemName=a)"))
		Expect(errTestItemDuplicated.Params()).To(BeEmpty())
	})

	It("should not share params between derived errors", func() {
		base := errTestItemDuplicated.With("itemName", "a")
		derived := base.With("itemName", "b")

		Expect(base.Params()).To(HaveKeyWithValue("itemName", "a"))
		Expect(derived.Params()).To(HaveKeyWithValue("itemName", "b"))
	})

	It("should record the stack of the caller", func() {
		err := errTestItemDuplicated.With("itemName", "a")
		Expect(fmt.Sprintf("%+v", err)).To(ContainSubstring("define_test.go"))
	})
})

var _ = Describe("Is", func() {
	It("should match derived errors with their definition", func() {
		err := pkgerrors.Wrap(errTestItemDuplicated.With("itemName", "a"), "create item")
		Expect(errors.Is(err, errTestItemDuplicated)).To(BeTrue())
		Expect(errors.Is(err, errTestItemNotFound)).To(BeFalse())
	})

	It("should not match plain bkerrs errors with the same code", func() {
		plain := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "item name duplicated")
		Expect(errors.Is(plain, errTestItemDuplicated)).To(BeFalse())
		Expect(errors.Is(errTestItemDuplicated, plain)).To(BeFalse())
	})

	It("should not panic on nil errors", func() {
		var nilErr *bkerrs.Error
		Expect(nilErr.Is(errTestItemDuplicated)).To(BeFalse())
		Expect(errTestItemDuplicated.Is(nilErr)).To(BeFalse())
	})
})

var _ = Describe("FromError", func() {
	It("should return nil for unclassified errors", func() {
		Expect(bkerrs.FromError(errors.New("boom"))).To(BeNil())
		Expect(bkerrs.FromError(nil)).To(BeNil())
	})

	It("should find the classified error through context wrappers", func() {
		err := pkgerrors.Wrap(
			fmt.Errorf("validate: %w", errTestItemDuplicated.With("itemName", "a")),
			"create item",
		)
		got := bkerrs.FromError(err)
		Expect(got).NotTo(BeNil())
		Expect(got.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
		Expect(got.Params()).To(HaveKeyWithValue("itemName", "a"))
	})

	It("should prefer the outer error when reclassified with WithCause", func() {
		inner := pkgerrors.Wrap(errTestItemNotFound.With("name", "a"), "get item")
		err := pkgerrors.Wrap(errTestDataInconsistent.WithCause(inner), "update item")

		got := bkerrs.FromError(err)
		Expect(got.Code()).To(Equal(bkerrs.ErrCodeInternalServerError))
		Expect(errors.Is(err, errTestItemNotFound)).To(BeTrue())
		Expect(err.Error()).To(Equal("update item: data inconsistent: get item: item not found (name=a)"))
	})
})

var _ = Describe("Details of defined errors", func() {
	It("should be generated from detail code and params", func() {
		details := errTestItemDuplicated.With("itemName", "a").Details()
		Expect(details.AsMaps()).To(Equal([]map[string]any{{
			"code":    bkerrs.ErrDetailCode("TEST_ITEM_DUPLICATED"),
			"message": "item name duplicated",
			"system":  bkerrs.SystemName,
			"module":  "",
			"extras":  map[string]string{"itemName": "a"},
		}}))
	})

	It("should prefer explicitly set details", func() {
		details := errTestItemDuplicated.SetDetails(bkerrs.NewDetail("CUSTOM", "custom detail")).Details()
		Expect(details).To(HaveLen(1))
		Expect(details.AsMaps()[0]["code"]).To(Equal(bkerrs.ErrDetailCode("CUSTOM")))
	})

	It("should not modify the definition when setting details on it", func() {
		_ = errTestItemDuplicated.SetDetails(bkerrs.NewDetail("CUSTOM", "custom detail"))
		details := errTestItemDuplicated.Details()
		Expect(details.AsMaps()[0]["code"]).To(Equal(bkerrs.ErrDetailCode("TEST_ITEM_DUPLICATED")))
	})

	It("should stay empty for plain bkerrs errors", func() {
		Expect(bkerrs.New(bkerrs.ErrCodeInvalidArgument, "bad").Details()).To(BeEmpty())
	})
})
