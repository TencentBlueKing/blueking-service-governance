package workspace

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Workspace lifecycle hooks", func() {
	BeforeEach(func() {
		ResetLifecycleHooksForTest()
	})

	AfterEach(func() {
		ResetLifecycleHooksForTest()
	})

	It("does not replace a hook registered with the same name", func() {
		Expect(RegisterPreDeleteHook("cleanup", func(context.Context, string) error {
			return nil
		})).To(BeTrue())
		Expect(RegisterPreDeleteHook("cleanup", func(context.Context, string) error {
			return nil
		})).To(BeFalse())
		Expect(IsPreDeleteHookRegistered("cleanup")).To(BeTrue())
	})

	It("stops the lifecycle when a hook fails", func() {
		expected := errors.New("cleanup failed")
		calledAfterFailure := false
		Expect(RegisterPreDeleteHook("failing", func(context.Context, string) error {
			return expected
		})).To(BeTrue())
		Expect(RegisterPreDeleteHook("later", func(context.Context, string) error {
			calledAfterFailure = true
			return nil
		})).To(BeTrue())

		err := runPreDeleteHooks(context.Background(), "workspace-hooks")
		Expect(err).To(MatchError(ContainSubstring("run workspace pre-delete hook failing")))
		Expect(errors.Is(err, expected)).To(BeTrue())
		Expect(calledAfterFailure).To(BeFalse())
	})
})
