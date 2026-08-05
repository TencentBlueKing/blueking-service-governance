package updater

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpdater(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/updater Suite")
}
