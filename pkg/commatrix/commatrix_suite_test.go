package commatrix_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCommatrix(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "commatrix Suite")
}
