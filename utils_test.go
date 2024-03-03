package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func chdir(t *testing.T, dir string) {
	cwd, err := os.Getwd()
	require.Nilf(t, err, "Error getting working directory: %v", err)

	err = os.Chdir(dir)
	require.Nilf(t, err, "Error changing working directory: %v", err)

	t.Cleanup(func() {
		err := os.Chdir(cwd)
		require.Nilf(t, err, "Error reverting working directory: %v", err)
	})
}
