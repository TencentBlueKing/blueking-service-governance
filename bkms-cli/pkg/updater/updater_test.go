package updater

import (
	"errors"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version"
)

func useCurrentVersion(value string) {
	originalVersion := version.Version
	version.Version = value
	DeferCleanup(func() { version.Version = originalVersion })
}

var _ = Describe("Updater", func() {
	Describe("provider selection", func() {
		It("rejects builds without an update source", func() {
			originalSource := updateSource
			updateSource = ""
			DeferCleanup(func() { updateSource = originalSource })

			_, err := newProvider()
			Expect(err).To(MatchError(MatchRegexp("self-update is not configured.*update source")))
			Expect(errors.Is(err, ErrUpdateNotConfigured)).To(BeTrue())
		})
	})

	Describe("version comparison", func() {
		DescribeTable("only reports a strictly newer SemVer as available",
			func(currentValue, latestValue string, expected bool) {
				current, err := parseVersion(currentValue)
				Expect(err).NotTo(HaveOccurred())
				latest, err := parseVersion(latestValue)
				Expect(err).NotTo(HaveOccurred())

				Expect(buildInfo(current, latest).Available).To(Equal(expected))
			},
			Entry("newer", "v1.2.3", "v1.3.0", true),
			Entry("same with optional v prefix", "v1.2.3", "1.2.3", false),
			Entry("older", "v1.3.0", "v1.2.3", false),
			Entry("prerelease of the current version", "v1.2.3", "v1.2.3-fix.1", false),
		)

		It("rejects development builds without a version", func() {
			_, err := parseVersion("")
			Expect(errors.Is(err, ErrInvalidVersion)).To(BeTrue())
		})

		DescribeTable("rejects non-SemVer tags",
			func(value string) {
				_, err := parseVersion(value)
				Expect(errors.Is(err, ErrInvalidVersion)).To(BeTrue())
			},
			Entry("date tag", "v20260101"),
			Entry("date fix tag", "v20260101-fix"),
			Entry("partial version", "v1.2"),
			Entry("leading zero", "v01.2.3"),
			Entry("another product tag", "bkms-server/v1.2.3"),
		)

		It("trims version files and linker values", func() {
			version, err := parseVersion("  v1.2.3\n")
			Expect(err).NotTo(HaveOccurred())
			Expect(version.String()).To(Equal("1.2.3"))
		})

		It("accepts the bkms-cli Git tag format", func() {
			version, err := parseVersion("bkms-cli/v1.2.3")
			Expect(err).NotTo(HaveOccurred())
			Expect(version.String()).To(Equal("1.2.3"))
		})
	})

	Describe("binary size", func() {
		It("accepts an unknown or maximum-sized asset", func() {
			Expect(validateBinarySize(-1)).To(Succeed())
			Expect(validateBinarySize(maxBinarySize)).To(Succeed())
		})

		It("rejects an asset larger than the limit", func() {
			err := validateBinarySize(maxBinarySize + 1)
			Expect(errors.Is(err, ErrBinaryTooLarge)).To(BeTrue())
		})

		It("normalizes the standard library size error", func() {
			err := normalizeBinarySizeError(&http.MaxBytesError{Limit: maxBinarySize})
			Expect(errors.Is(err, ErrBinaryTooLarge)).To(BeTrue())
		})
	})
})
