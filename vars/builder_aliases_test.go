package vars

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilderAliases(t *testing.T) {
	require.Equal(t, "foobar", BuilderNameFromExtraData("foobar"))
	require.Equal(t, "builder0x69", BuilderNameFromExtraData("@builder0x69"))
	require.Equal(t, "bob the builder", BuilderNameFromExtraData("s1e2xf"))
}
