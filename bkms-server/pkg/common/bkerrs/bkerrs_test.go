package bkerrs_test

import (
	"errors"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

var _ = Describe("Error", func() {
	Describe("New", func() {
		It("should create error with code and message", func() {
			err := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "field username cannot be empty")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("field username cannot be empty"))
		})

		It("should capture stack trace", func() {
			err := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "test error")
			Expect(err.StackTrace()).NotTo(BeNil())
		})

		It("should support chaining with SetDetails", func() {
			err := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "validation failed").SetDetails(
				bkerrs.NewDetail("VAL001", "field required"),
			)
			Expect(err.Details()).To(HaveLen(1))
			Expect(err.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
		})
	})

	Describe("Errorf", func() {
		It("should create error with formatted message", func() {
			err := bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "field %s cannot be empty", "username")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("field username cannot be empty"))
		})

		It("should support multiple format arguments", func() {
			err := bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "expected %d, got %d", 10, 5)
			Expect(err.Error()).To(Equal("expected 10, got 5"))
		})
	})

	Describe("Wrap", func() {
		It("should wrap an existing error with code and message", func() {
			originalErr := errors.New("database connection failed")
			wrappedErr := bkerrs.Wrap(originalErr, bkerrs.ErrCodeInternalError, "failed to query user")

			Expect(wrappedErr).NotTo(BeNil())
			Expect(wrappedErr.Error()).To(Equal("failed to query user: database connection failed"))
		})

		It("should preserve the original error as cause", func() {
			originalErr := errors.New("original error")
			wrappedErr := bkerrs.Wrap(originalErr, bkerrs.ErrCodeInternalError, "wrapped")
			Expect(wrappedErr.Cause()).To(Equal(originalErr))
		})

		It("should support multiple levels of wrapping", func() {
			err1 := errors.New("level 1")
			err2 := bkerrs.Wrap(err1, bkerrs.ErrCodeInternalError, "level 2")
			err3 := bkerrs.Wrap(err2, bkerrs.ErrCodeInvalidArgument, "level 3")

			Expect(err3.Error()).To(Equal("level 3: level 2: level 1"))
		})
	})

	Describe("WrapBSCPNotFullyReleased", func() {
		It("should wrap not fully released error with detail code", func() {
			err := bkerrs.WrapBSCPNotFullyReleased(
				bkerrs.New(bkerrs.ErrCodeNotFound, "no fully released version"),
				"biz-1",
				"service-1",
			)

			var bkErr *bkerrs.Error
			Expect(errors.As(err, &bkErr)).To(BeTrue())
			Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeNotFound))
			details := bkErr.Details()
			Expect(details).To(HaveLen(1))
			Expect((&details).AsMaps()[0]["code"]).To(Equal(bkerrs.ErrDetailCodeNotFullyReleased))
		})
	})

	Describe("WrapBSCPNoPermission", func() {
		It("should wrap BSCP permission error with detail code", func() {
			err := bkerrs.WrapBSCPNoPermission(
				errors.New("bscp forbidden"),
				"list bscp service configs",
			)

			var bkErr *bkerrs.Error
			Expect(errors.As(err, &bkErr)).To(BeTrue())
			Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeNoPermission))
			details := bkErr.Details()
			Expect(details).To(HaveLen(1))
			Expect((&details).AsMaps()[0]["code"]).To(Equal(bkerrs.ErrDetailCodeBSCPNoPermission))
		})
	})

	Describe("Wrapf", func() {
		It("should wrap an error with formatted message", func() {
			originalErr := errors.New("timeout")
			wrappedErr := bkerrs.Wrapf(
				originalErr,
				bkerrs.ErrCodeInternalError,
				"operation failed after %d retries",
				3,
			)

			Expect(wrappedErr.Error()).To(Equal("operation failed after 3 retries: timeout"))
		})
	})

	Describe("Error methods", func() {
		var testErr *bkerrs.Error

		BeforeEach(func() {
			originalErr := errors.New("database error")
			testErr = bkerrs.Wrap(originalErr, bkerrs.ErrCodeInternalError, "query failed")
		})

		Describe("Error", func() {
			It("should return formatted error message", func() {
				Expect(testErr.Error()).To(Equal("query failed: database error"))
			})
		})

		Describe("String", func() {
			It("should return error message without details", func() {
				Expect(testErr.String()).To(Equal("query failed: database error"))
			})

			It("should include details when set", func() {
				testErr = testErr.SetDetails(
					bkerrs.NewDetail("ERR001", "detail message 1"),
					bkerrs.NewDetail("ERR002", "detail message 2"),
				)
				excepted := "query failed: database error\n" +
					" - details[1]: [ERR001] detail message 1\n" +
					" - details[2]: [ERR002] detail message 2"
				Expect(testErr.String()).To(Equal(excepted))
			})
		})

		Describe("Cause", func() {
			It("should return the original error", func() {
				cause := testErr.Cause()
				Expect(cause).NotTo(BeNil())
				Expect(cause.Error()).To(Equal("database error"))
			})
		})

		Describe("Code", func() {
			It("should return the error code", func() {
				code := testErr.Code()
				Expect(code).To(Equal(bkerrs.ErrCodeInternalError))
			})
		})

		Describe("Details", func() {
			It("should return empty details by default", func() {
				details := testErr.Details()
				Expect(details).To(BeEmpty())
			})

			It("should return set details", func() {
				testErr = testErr.SetDetails(bkerrs.NewDetail("ERR001", "test detail"))
				details := testErr.Details()
				Expect(details).To(HaveLen(1))
			})
		})

		Describe("SetDetails", func() {
			It("should set error details", func() {
				result := testErr.SetDetails(
					bkerrs.NewDetail("ERR001", "detail 1"),
					bkerrs.NewDetail("ERR002", "detail 2"),
				)
				Expect(result).To(Equal(testErr))
				Expect(testErr.Details()).To(HaveLen(2))
			})

			It("should support chaining", func() {
				err := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "test").
					SetDetails(bkerrs.NewDetail("D1", "detail 1")).
					SetDetails(bkerrs.NewDetail("D2", "detail 2"))
				Expect(err.Details()).To(HaveLen(1))
			})

			It("should replace existing details", func() {
				testErr = testErr.SetDetails(bkerrs.NewDetail("ERR001", "old"))
				testErr = testErr.SetDetails(bkerrs.NewDetail("ERR002", "new"))
				Expect(testErr.Details()).To(HaveLen(1))
			})
		})

		Describe("StackTrace", func() {
			It("should return stack trace compatible with pkg/errors", func() {
				trace := testErr.StackTrace()
				Expect(trace).NotTo(BeNil())
				Expect(len(trace)).To(BeNumerically(">", 0))
			})

			It("should return nil for error without stack", func() {
				err := &bkerrs.Error{}
				trace := err.StackTrace()
				Expect(trace).To(BeNil())
			})
		})
	})

	Describe("Format", func() {
		var testErr error

		BeforeEach(func() {
			originalErr := errors.New("original error")
			testErr = bkerrs.Wrap(originalErr, bkerrs.ErrCodeInvalidArgument, "wrapped error")
		})

		Describe("%s format", func() {
			It("should output simple error message", func() {
				output := fmt.Sprintf("%s", testErr)
				Expect(output).To(Equal("wrapped error: original error"))
			})
		})

		Describe("%v format", func() {
			It("should output error message", func() {
				output := fmt.Sprintf("%v", testErr)
				Expect(output).To(Equal("wrapped error: original error"))
			})
		})

		Describe("%+v format", func() {
			It("should output error message with stack trace", func() {
				output := fmt.Sprintf("%+v", testErr)
				Expect(output).To(ContainSubstring("wrapped error"))
				// Stack trace should contain file paths
				Expect(output).To(ContainSubstring("errs_test.go"))
			})

			It("should recursively print cause chain", func() {
				err1 := pkgerrors.New("level 1")
				err2 := bkerrs.Wrap(err1, bkerrs.ErrCodeInternalError, "level 2")
				err3 := bkerrs.Wrap(err2, bkerrs.ErrCodeInvalidArgument, "level 3")

				output := fmt.Sprintf("%+v", err3)
				Expect(output).To(ContainSubstring("level 1"))
				Expect(output).To(ContainSubstring("level 2: level 1"))
				Expect(output).To(ContainSubstring("level 3: level 2: level 1"))
				// 调用堆栈中应该包含 error 被创建 / 包装的位置
				Expect(output).To(ContainSubstring("/pkg/common/bkerrs/bkerrs_test.go"))
				// 有三层错误包装，这里完整的堆栈应该包含三次 errs_test.go
				Expect(strings.Count(output, "errs_test.go")).To(Equal(3))
				// 调用堆栈中应该包含 errs 包中方法的位置
				Expect(output).NotTo(ContainSubstring("/pkg/common/bkerrs/bkerrs.go"))
			})

			It("should recursively print cause chain when wrap by pkg/errors", func() {
				err1 := bkerrs.New(bkerrs.ErrCodeInternalError, "level 1")
				err2 := bkerrs.Wrap(err1, bkerrs.ErrCodeInvalidArgument, "level 2")
				err3 := pkgerrors.Wrap(err2, "level 3")

				output := fmt.Sprintf("%+v", err3)
				Expect(output).To(ContainSubstring("level 1"))
				Expect(output).To(ContainSubstring("level 2: level 1"))
				Expect(output).To(ContainSubstring("level 3"))
				// 目前设计中，err.Error 与 pkg/errors 中的 error Format 行为不同之处在于每次都有包含子 error 的信息
				// 可以对比 “should recursively print cause chain” 这个测试的内容以便了解其中不同之处
				Expect(output).NotTo(ContainSubstring("level 3: level 2: level 1"))
			})
		})

		Describe("%q format", func() {
			It("should output quoted error message", func() {
				output := fmt.Sprintf("%q", testErr)
				Expect(output).To(Equal("\"wrapped error: original error\""))
			})
		})

		Describe("Format with Details", func() {
			var errWithDetails *bkerrs.Error

			BeforeEach(func() {
				originalErr := errors.New("database error")
				errWithDetails = bkerrs.Wrap(originalErr, bkerrs.ErrCodeInternalError, "query failed").SetDetails(
					bkerrs.NewDetail("ERR001", "connection timeout"),
					bkerrs.NewDetail("ERR002", "retry failed"),
				)
			})

			Describe("%s format with details", func() {
				It("should output error message without details", func() {
					output := fmt.Sprintf("%s", errWithDetails)
					Expect(output).To(Equal("query failed: database error"))
				})
			})

			Describe("%v format with details", func() {
				It("should output error message without details", func() {
					output := fmt.Sprintf("%v", errWithDetails)
					Expect(output).To(Equal("query failed: database error"))
				})
			})

			Describe("%q format with details", func() {
				It("should output quoted error message without details", func() {
					output := fmt.Sprintf("%q", errWithDetails)
					Expect(output).To(Equal("\"query failed: database error\""))
				})
			})

			Describe("String method with details", func() {
				It("should include details in String output", func() {
					output := errWithDetails.String()
					expected := "query failed: database error\n" +
						" - details[1]: [ERR001] connection timeout\n" +
						" - details[2]: [ERR002] retry failed"
					Expect(output).To(Equal(expected))
				})
			})

			Describe("Multiple details with mixed fields", func() {
				It("should format multiple details with different field combinations", func() {
					err := bkerrs.New(bkerrs.ErrCodeInternalError, "operation failed").SetDetails(
						bkerrs.NewDetail("ERR001", "first error", bkerrs.WithSystem("sys1")),
						bkerrs.NewDetail("ERR002", "second error", bkerrs.WithModule("mod1")),
						bkerrs.NewDetail("ERR003", "third error",
							bkerrs.WithSystem("sys2"),
							bkerrs.WithModule("mod2"),
							bkerrs.WithExtras(map[string]string{"key": "value"})),
					)
					output := err.String()
					expected := "operation failed\n" +
						" - details[1]: [ERR001] [system:sys1] first error\n" +
						" - details[2]: [ERR002] [module:mod1] second error\n" +
						" - details[3]: [ERR003] [system:sys2] [module:mod2] third error [key:value]"
					Expect(output).To(Equal(expected))
				})
			})

			Describe("Details with error chain", func() {
				It("should format details with wrapped errors", func() {
					err1 := errors.New("root cause")
					err2 := bkerrs.Wrap(err1, bkerrs.ErrCodeInternalError, "level 2")
					err3 := bkerrs.Wrap(err2, bkerrs.ErrCodeInvalidArgument, "level 3").SetDetails(
						bkerrs.NewDetail("CHAIN001", "error in chain",
							bkerrs.WithSystem("test"),
							bkerrs.WithExtras(map[string]string{"level": "3"})),
					)

					output := err3.String()
					expected := "level 3: level 2: root cause\n" +
						" - details[1]: [CHAIN001] [system:test] error in chain [level:3]"
					Expect(output).To(Equal(expected))
				})
			})
		})
	})
})

var _ = Describe("Detail", func() {
	Describe("Detail", func() {
		Describe("NewDetail", func() {
			It("should create a detail with code and message", func() {
				detail := bkerrs.NewDetail("ERR001", "test error")
				Expect(detail.String()).To(ContainSubstring("ERR001"))
				Expect(detail.String()).To(ContainSubstring("test error"))
			})

			It("should support system option", func() {
				detail := bkerrs.NewDetail("ERR001", "test", bkerrs.WithSystem("bkms"))
				Expect(detail.String()).To(Equal("[ERR001] [system:bkms] test"))
			})

			It("should support module option", func() {
				detail := bkerrs.NewDetail("ERR001", "test", bkerrs.WithModule("auth"))
				Expect(detail.String()).To(Equal("[ERR001] [module:auth] test"))
			})

			It("should support extras option", func() {
				extras := map[string]string{"foo": "bar"}
				detail := bkerrs.NewDetail("ERR001", "test", bkerrs.WithExtras(extras))
				// extras 的 key-value 会被展开为独立的方括号
				Expect(detail.String()).To(Equal("[ERR001] test [foo:bar]"))
			})

			It("should support multiple options", func() {
				extras := map[string]string{"key": "value"}
				detail := bkerrs.NewDetail(
					"ERR001",
					"test",
					bkerrs.WithSystem("bkms"),
					bkerrs.WithModule("auth"),
					bkerrs.WithExtras(extras),
				)
				// 格式: [code] [system] [module] message [extras]
				Expect(detail.String()).To(Equal("[ERR001] [system:bkms] [module:auth] test [key:value]"))
			})
		})
	})

	Describe("Details", func() {
		Describe("AsMaps", func() {
			It("should convert details to map slice", func() {
				extras := map[string]string{"key": "value"}
				details := bkerrs.Details{
					bkerrs.NewDetail("ERR001", "error 1", bkerrs.WithSystem("sys1")),
					bkerrs.NewDetail("ERR002", "error 2", bkerrs.WithModule("mod1")),
					bkerrs.NewDetail("ERR003", "error 3", bkerrs.WithExtras(extras)),
				}

				mapSlice := details.AsMaps()
				var emptyStrMap map[string]string
				Expect(mapSlice).To(Equal([]map[string]any{
					{
						"code":    bkerrs.ErrDetailCode("ERR001"),
						"message": "error 1",
						"system":  "sys1",
						"module":  "",
						"extras":  emptyStrMap,
					},
					{
						"code":    bkerrs.ErrDetailCode("ERR002"),
						"message": "error 2",
						"system":  "",
						"module":  "mod1",
						"extras":  emptyStrMap,
					},
					{
						"code":    bkerrs.ErrDetailCode("ERR003"),
						"message": "error 3",
						"system":  "",
						"module":  "",
						"extras":  extras,
					},
				}))
			})

			It("should handle empty details", func() {
				details := bkerrs.Details{}
				mapSlice := details.AsMaps()
				Expect(mapSlice).To(BeEmpty())
			})
		})
	})
})

var _ = Describe("Integration", func() {
	Describe("Stack trace integration", func() {
		It("should be compatible with pkg/errors", func() {
			err := bkerrs.New(bkerrs.ErrCodeInternalError, "test error")

			// Should implement stackTracer interface
			trace := err.StackTrace()
			Expect(trace).NotTo(BeNil())

			// Should be compatible with pkg/errors.Frame
			for _, frame := range trace {
				// Frame should be formattable
				output := fmt.Sprintf("%+v", frame)
				Expect(output).NotTo(BeEmpty())
			}
		})

		It("should capture correct call stack", func() {
			err := bkerrs.New(bkerrs.ErrCodeInternalError, "test error from helper")
			trace := err.StackTrace()

			// Stack should contain frames
			Expect(len(trace)).To(BeNumerically(">", 0))

			// Each frame should be formattable
			for _, frame := range trace {
				output := fmt.Sprintf("%+v", frame)
				Expect(output).NotTo(BeEmpty())
			}
		})
	})

	Describe("Error chain traversal", func() {
		It("should support errors.Is", func() {
			originalErr := errors.New("original error")
			wrappedErr := bkerrs.Wrap(originalErr, bkerrs.ErrCodeInternalError, "wrapped")

			// 测试 errors.Is 可以正确识别错误链中的原始错误
			Expect(errors.Is(wrappedErr, originalErr)).To(BeTrue())

			// 直接访问方法，无需类型断言
			Expect(wrappedErr.Code()).To(Equal(bkerrs.ErrCodeInternalError))
			Expect(wrappedErr.Cause()).To(Equal(originalErr))
		})

		It("should support errors.Is with multi-level wrapping", func() {
			baseErr := errors.New("base error")
			err1 := bkerrs.Wrap(baseErr, bkerrs.ErrCodeInternalError, "level 1")
			err2 := bkerrs.Wrap(err1, bkerrs.ErrCodeInvalidArgument, "level 2")

			// errors.Is 应该能够遍历整个错误链
			Expect(errors.Is(err2, baseErr)).To(BeTrue())
			Expect(errors.Is(err2, err1)).To(BeTrue())
			Expect(errors.Is(err1, baseErr)).To(BeTrue())

			// 不相关的错误应该返回 false
			otherErr := errors.New("other error")
			Expect(errors.Is(err2, otherErr)).To(BeFalse())
		})

		It("should support Unwrap method", func() {
			originalErr := errors.New("original")
			wrappedErr := bkerrs.Wrap(originalErr, bkerrs.ErrCodeInternalError, "wrapped")

			// 测试 Unwrap 方法
			Expect(wrappedErr.Unwrap()).To(Equal(originalErr))

			// Unwrap 应该和 Cause 返回相同的值
			Expect(wrappedErr.Unwrap()).To(Equal(wrappedErr.Cause()))
		})

		It("should traverse multi-level error chain", func() {
			err1 := errors.New("level 1")
			err2 := bkerrs.Wrap(err1, bkerrs.ErrCodeInternalError, "level 2")
			err3 := bkerrs.Wrap(err2, bkerrs.ErrCodeInvalidArgument, "level 3")

			// Level 3
			Expect(err3.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))

			// Level 2
			var e2 *bkerrs.Error
			errors.As(err3.Cause(), &e2)
			Expect(e2.Code()).To(Equal(bkerrs.ErrCodeInternalError))

			// Level 1
			e1 := e2.Cause()
			Expect(e1.Error()).To(Equal("level 1"))
		})
	})

	Describe("Edge cases", func() {
		It("should handle nil cause", func() {
			err := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "test")
			Expect(err.Cause()).To(BeNil())
		})

		It("should handle empty message", func() {
			err := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal(""))
		})

		It("should handle special characters in message", func() {
			err := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "error with\nnewline\tand\ttabs")
			output := fmt.Sprintf("%q", err)
			Expect(output).To(ContainSubstring("\\n"))
			Expect(output).To(ContainSubstring("\\t"))
		})

		It("should handle very long error chains", func() {
			err := errors.New("base")
			for i := 0; i < 10; i++ {
				err = bkerrs.Wrapf(err, bkerrs.ErrCodeInternalError, "level %d", i)
			}
			Expect(err).NotTo(BeNil())
			output := fmt.Sprintf("%+v", err)
			Expect(output).To(ContainSubstring("level 9"))
			Expect(output).To(ContainSubstring("base"))
		})
	})
})
