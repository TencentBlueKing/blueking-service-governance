package polaris

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Polaris weight JSON patch", func() {
	It("should test the target service before adding an explicit zero weight", func() {
		patch, err := buildWeightPatch("app-config-polaris-service", 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(patch).To(MatchJSON(`[
			{"op":"test","path":"/spec/services/0/name","value":"app-config-polaris-service"},
			{"op":"add","path":"/spec/services/0/weight","value":0}
		]`))
	})
})
