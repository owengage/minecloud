package serverwrapper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAvailableMiB(t *testing.T) {
	a, err := AvailableMiB()
	require.NoError(t, err)

	fmt.Println(a, "MiB")
	t.Fail()
}
