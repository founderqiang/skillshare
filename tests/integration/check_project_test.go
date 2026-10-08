//go:build !online

package integration

import (
	"testing"

	"skillshare/internal/testutil"
)

func TestCheckProject_NoItems(t *testing.T) {
	sb := testutil.NewSandbox(t)
	defer sb.Cleanup()

	projectRoot := sb.SetupProjectDir("claude")

	result := sb.RunCLIInDir(projectRoot, "check", "-p")
	result.AssertSuccess(t)
	result.AssertOutputContains(t, "No tracked")
}
