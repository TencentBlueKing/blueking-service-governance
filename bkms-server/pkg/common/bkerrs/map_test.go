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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

var errUnknownKey = errors.New("unknown key")

type quotaExceededError struct{ limit int }

func (e *quotaExceededError) Error() string { return fmt.Sprintf("quota %d exceeded", e.limit) }

var _ = Describe("Map", func() {
	asBkErr := func(err error) *bkerrs.Error {
		var bkErr *bkerrs.Error
		Expect(errors.As(err, &bkErr)).To(BeTrue())
		return bkErr
	}

	It("should return nil for nil error", func() {
		Expect(bkerrs.Map(nil, bkerrs.Otherwise(bkerrs.ErrCodeInvalidArgument, "x"))).To(Succeed())
	})

	It("should return the error unchanged when no rule matches", func() {
		err := pkgerrors.New("boom")
		Expect(bkerrs.Map(err, bkerrs.On(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "unknown key"))).
			To(BeIdenticalTo(err))
	})

	It("should map a sentinel error found anywhere in the chain", func() {
		err := pkgerrors.Wrap(errUnknownKey, "update switch")
		mapped := bkerrs.Map(err, bkerrs.On(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "unknown key %q", "k1"))

		bkErr := asBkErr(mapped)
		Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
		Expect(mapped.Error()).To(Equal(`unknown key "k1": update switch: unknown key`))
		Expect(errors.Is(mapped, errUnknownKey)).To(BeTrue())
	})

	It("should use the first matching rule", func() {
		mapped := bkerrs.Map(errUnknownKey,
			bkerrs.On(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "first"),
			bkerrs.Otherwise(bkerrs.ErrCodeInternalServerError, "fallback"),
		)
		Expect(asBkErr(mapped).Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))

		mapped = bkerrs.Map(pkgerrors.New("other"),
			bkerrs.On(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "first"),
			bkerrs.Otherwise(bkerrs.ErrCodeInternalServerError, "fallback"),
		)
		Expect(asBkErr(mapped).Code()).To(Equal(bkerrs.ErrCodeInternalServerError))
	})

	It("should build the API error from a typed domain error", func() {
		rule := bkerrs.OnAs(func(err error, e *quotaExceededError) *bkerrs.Error {
			return bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidRequest, "at most %d items are allowed", e.limit)
		})
		mapped := bkerrs.Map(pkgerrors.Wrap(&quotaExceededError{limit: 3}, "create item"), rule)

		Expect(asBkErr(mapped).Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
		Expect(mapped.Error()).To(HavePrefix("at most 3 items are allowed: "))
	})

	It("should build the API error with details through OnFunc", func() {
		rule := bkerrs.OnFunc(errUnknownKey, func(err error) *bkerrs.Error {
			return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "unknown key").
				SetDetails(bkerrs.NewDetail("UNKNOWN_KEY", "unknown key"))
		})
		bkErr := asBkErr(bkerrs.Map(errUnknownKey, rule))
		Expect(bkErr.Details()).To(HaveLen(1))
	})
})
