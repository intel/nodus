package e2e

import (
	"github.com/IntelAI/nodus/pkg/nptest"
	"testing"
)

func TestNPTestLibrary(t *testing.T) {
	np := nptest.New()
	defer np.Shutdown()

	np.Test(t, "assert 0 pods within 10s")
	np.Test(t, "create 1 large node")
	np.Test(t, "assert 1 large node")
	np.Test(t, "create 2 small nodes")
	np.Test(t, "assert 2 small nodes")

	np.Test(t, "create 1 4-cpu pod")
	np.Test(t, "assert 1 4-cpu pod is Running within 4s")
	np.Test(t, "delete 1 4-cpu pod")

	np.Test(t, "create 3 1-cpu pods")
	np.Test(t, "assert 3 1-cpu pods are Running within 4s")

	np.Test(t, "change 1 1-cpu pod from Running to Succeeded")
	np.Test(t, "assert 0 1-cpu pods are Pending")
	np.Test(t, "assert 2 1-cpu pods are Running")
	np.Test(t, "assert 1 1-cpu pod is Succeeded")
	np.Test(t, "assert 0 1-cpu pods are Failed")

	np.Test(t, "change 1 1-cpu pod from Running to Failed")
	np.Test(t, "assert 0 1-cpu pods are Pending")
	np.Test(t, "assert 1 1-cpu pod is Running")
	np.Test(t, "assert 1 1-cpu pod is Succeeded")
	np.Test(t, "assert 1 1-cpu pod is Failed")

	np.Test(t, "assert 2 1-cpu pods are Succeeded within 10s")

	np.Test(t, "assert 2 1-cpu pods are Succeeded within 10s")
}
