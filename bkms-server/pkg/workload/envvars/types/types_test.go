package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnvVariableList", func() {
	Describe("ToDeduplicatedList", func() {
		It("should keep only the last item for duplicate keys while preserving effective item order", func() {
			vars := EnvVariableList{
				{Key: "SHARED_KEY", Value: "workspace-value"},
				{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
				{Key: "SHARED_KEY", Value: "envtype-value"},
				{Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"},
				{Key: "SHARED_KEY", Value: "env-value"},
				{Key: "ENV_ONLY_KEY", Value: "env-only-value"},
				{Key: "SHARED_KEY", Value: "app-value"},
				{Key: "APP_ONLY_KEY", Value: "app-only-value"},
			}

			Expect(vars.ToDeduplicatedList()).To(Equal(EnvVariableList{
				{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
				{Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"},
				{Key: "ENV_ONLY_KEY", Value: "env-only-value"},
				{Key: "SHARED_KEY", Value: "app-value"},
				{Key: "APP_ONLY_KEY", Value: "app-only-value"},
			}))
		})
	})
})
