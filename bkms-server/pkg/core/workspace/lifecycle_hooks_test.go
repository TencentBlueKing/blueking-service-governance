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

	It("runs named create hooks in registration order", func() {
		calls := make([]string, 0, 2)
		Expect(RegisterCreateHook("first", func(context.Context, Workspace) error {
			calls = append(calls, "first")
			return nil
		})).To(BeTrue())
		Expect(RegisterCreateHook("second", func(context.Context, Workspace) error {
			calls = append(calls, "second")
			return nil
		})).To(BeTrue())

		Expect(runCreateHooks(context.Background(), Workspace{ID: "workspace-hooks"})).To(Succeed())
		Expect(calls).To(Equal([]string{"first", "second"}))
	})

	It("does not replace a hook registered with the same name", func() {
		Expect(RegisterDeleteHook("cleanup", func(context.Context, string) error {
			return nil
		})).To(BeTrue())
		Expect(RegisterDeleteHook("cleanup", func(context.Context, string) error {
			return nil
		})).To(BeFalse())
		Expect(IsDeleteHookRegistered("cleanup")).To(BeTrue())
	})

	It("stops the lifecycle when a hook fails", func() {
		expected := errors.New("cleanup failed")
		calledAfterFailure := false
		Expect(RegisterDeleteHook("failing", func(context.Context, string) error {
			return expected
		})).To(BeTrue())
		Expect(RegisterDeleteHook("later", func(context.Context, string) error {
			calledAfterFailure = true
			return nil
		})).To(BeTrue())

		err := runDeleteHooks(context.Background(), "workspace-hooks")
		Expect(err).To(MatchError(ContainSubstring("run workspace delete hook failing")))
		Expect(errors.Is(err, expected)).To(BeTrue())
		Expect(calledAfterFailure).To(BeFalse())
	})
})
