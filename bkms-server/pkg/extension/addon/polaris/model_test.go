package polaris_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
)

var _ = Describe("PolarisConfig", func() {
	Describe("IsAvailableInEnv", func() {
		It("should return true only for environments in ScopeEnvNames", func() {
			config := &polaris.PolarisConfig{
				ScopeEnvNames: []string{"dev", "staging"},
			}
			Expect(config.IsAvailableInEnv("dev")).To(BeTrue())
			Expect(config.IsAvailableInEnv("staging")).To(BeTrue())
			Expect(config.IsAvailableInEnv("production")).To(BeFalse())
		})

		It("should return false when ScopeEnvNames is empty", func() {
			config := &polaris.PolarisConfig{}
			Expect(config.IsAvailableInEnv("any-env")).To(BeFalse())
		})
	})

	Describe("GetEnvWeight", func() {
		It("should use the fixed default when the environment has no explicit value", func() {
			config := &polaris.PolarisConfig{}

			Expect(config.GetEnvWeight("dev")).To(Equal(polaris.DefaultEnvWeight))
		})

		It("should prefer an explicit environment weight including zero", func() {
			config := &polaris.PolarisConfig{
				EnvWeights: map[string]int32{"dev": 0},
			}

			Expect(config.GetEnvWeight("dev")).To(BeZero())
		})
	})
})
