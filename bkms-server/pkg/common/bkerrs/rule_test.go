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
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

var (
	errUnknownKey = errors.New("unknown key")
	errReadOnly   = errors.New("read only")
)

type quotaExceededError struct{ limit int }

func (e *quotaExceededError) Error() string { return fmt.Sprintf("quota %d exceeded", e.limit) }

func asBkErr(err error) *bkerrs.Error {
	var bkErr *bkerrs.Error
	Expect(errors.As(err, &bkErr)).To(BeTrue())
	return bkErr
}

var _ = Describe("Match", func() {
	It("should return nil for nil error", func() {
		Expect(bkerrs.Match(nil, bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "x"))).To(Succeed())
	})

	It("should return the error unchanged when no rule matches", func() {
		err := pkgerrors.New("boom")
		Expect(bkerrs.Match(err, bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "unknown key"))).
			To(BeIdenticalTo(err))
	})

	It("should match a sentinel error found anywhere in the chain", func() {
		err := pkgerrors.Wrap(errUnknownKey, "update switch")
		rule := bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "unknown key %q", "k1")
		matched := bkerrs.Match(err, rule)

		bkErr := asBkErr(matched)
		Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
		Expect(matched.Error()).To(Equal(`unknown key "k1": update switch: unknown key`))
		Expect(errors.Is(matched, errUnknownKey)).To(BeTrue())
	})

	It("should use the first matching rule", func() {
		rules := []bkerrs.Rule{
			bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "first"),
			bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidRequest, "second"),
			bkerrs.WhenIs(errReadOnly, bkerrs.ErrCodeInvalidRequest, "read only"),
		}
		Expect(asBkErr(bkerrs.Match(errUnknownKey, rules...)).Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
		Expect(asBkErr(bkerrs.Match(errReadOnly, rules...)).Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
	})

	It("should override an internal error wrapped by the handler when a rule matches", func() {
		rule := bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "unknown key")

		matched := bkerrs.Match(bkerrs.Wrap(errUnknownKey, bkerrs.ErrCodeInternalServerError, "update"), rule)
		Expect(asBkErr(matched).Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))

		matched = bkerrs.Match(bkerrs.Wrap(errReadOnly, bkerrs.ErrCodeInternalServerError, "update"), rule)
		Expect(asBkErr(matched).Code()).To(Equal(bkerrs.ErrCodeInternalServerError))
	})

	It("should build the API error from a typed domain error", func() {
		rule := bkerrs.WhenAs(func(err error, e *quotaExceededError) *bkerrs.Error {
			return bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidRequest, "at most %d items are allowed", e.limit)
		})
		matched := bkerrs.Match(pkgerrors.Wrap(&quotaExceededError{limit: 3}, "create item"), rule)

		Expect(asBkErr(matched).Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
		Expect(matched.Error()).To(HavePrefix("at most 3 items are allowed: "))
	})

	It("should build the API error with details through OnFunc", func() {
		rule := bkerrs.WhenIsFunc(errUnknownKey, func(err error) *bkerrs.Error {
			return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "unknown key").
				SetDetails(bkerrs.NewDetail("UNKNOWN_KEY", "unknown key"))
		})
		bkErr := asBkErr(bkerrs.Match(errUnknownKey, rule))
		Expect(bkErr.Details()).To(HaveLen(1))
	})
})

var _ = Describe("Rules", func() {
	It("should return the first matching rule in the group", func() {
		group := bkerrs.Rules(
			bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "first"),
			bkerrs.WhenIs(errReadOnly, bkerrs.ErrCodeInvalidRequest, "second"),
		)
		Expect(asBkErr(bkerrs.Match(errUnknownKey, group)).Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
		Expect(asBkErr(bkerrs.Match(errReadOnly, group)).Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
		Expect(group(pkgerrors.New("other"))).To(BeNil())
	})

	It("should be combinable with other rules and groups", func() {
		inner := bkerrs.Rules(bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "inner"))
		matched := bkerrs.Match(errReadOnly,
			bkerrs.Rules(inner),
			bkerrs.WhenIs(errReadOnly, bkerrs.ErrCodeNotFound, "outer"),
		)
		Expect(asBkErr(matched).Code()).To(Equal(bkerrs.ErrCodeNotFound))
	})

	It("should match nothing when empty", func() {
		Expect(bkerrs.Rules()(errUnknownKey)).To(BeNil())
	})
})

var _ = Describe("AbortWithErr", func() {
	newContext := func() *gin.Context {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		return c
	}

	It("should record the matched API error and abort", func() {
		c := newContext()
		bkerrs.AbortWithErr(c, pkgerrors.Wrap(errUnknownKey, "update"),
			bkerrs.WhenIs(errUnknownKey, bkerrs.ErrCodeInvalidArgument, "unknown key"))

		Expect(c.IsAborted()).To(BeTrue())
		Expect(asBkErr(c.Errors.Last().Err).Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
	})

	It("should record the error unchanged without rules", func() {
		c := newContext()
		err := pkgerrors.New("boom")
		bkerrs.AbortWithErr(c, err)

		Expect(c.IsAborted()).To(BeTrue())
		Expect(c.Errors.Last().Err).To(BeIdenticalTo(err))
	})
})
