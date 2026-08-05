package bscp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("BSCP API Client", func() {
	Describe("wrapOperationHTTPError", func() {
		It("marks forbidden responses as no-permission", func() {
			err := wrapOperationHTTPError("list_service_versions", 403, []byte(`{"message":"forbidden"}`))

			Expect(errors.Is(err, ErrNoPermission)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("http code: 403"))
		})

		It("does not mark internal errors as no-permission", func() {
			err := wrapOperationHTTPError("list_service_versions", 500, []byte(`{"message":"boom"}`))

			Expect(errors.Is(err, ErrNoPermission)).To(BeFalse())
			Expect(err.Error()).To(ContainSubstring("http code: 500"))
		})
	})
})
