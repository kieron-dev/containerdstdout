package containerdstdout_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestContainerdstdout(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Containerdstdout Suite")
}
