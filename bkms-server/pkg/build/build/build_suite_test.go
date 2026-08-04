package build

import (
	"testing"

	"git.woa.com/bcs/bkms-govern/apps/bkms-server/pkg/common/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBuild(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Build Suite")
}
