package gitcloneinternal

import (
	"fmt"

	"github.com/bitrise-io/go-utils/command/git"
)

// SetupSparseCheckout ...
func SetupSparseCheckout(gitCmd git.Git, sparseDirectories []string) error {
	if len(sparseDirectories) == 0 {
		return nil
	}

	initCommand := gitCmd.SparseCheckoutInit(true)
	if err := Runner.Run(initCommand); err != nil {
		return NewStepError(
			SparseCheckoutFailedTag,
			fmt.Errorf("initializing sparse-checkout config failed: %v", err),
			"Initializing sparse-checkout config has failed",
		)
	}

	sparseSetCommand := gitCmd.SparseCheckoutSet(sparseDirectories...)
	if err := Runner.Run(sparseSetCommand); err != nil {
		return NewStepError(
			SparseCheckoutFailedTag,
			fmt.Errorf("updating sparse-checkout config failed: %v", err),
			"Updating sparse-checkout config has failed",
		)
	}

	return nil
}
