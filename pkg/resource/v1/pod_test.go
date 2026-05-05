package v1_test

import (
	"testing"

	v1 "github.com/cirruslabs/orchard/pkg/resource/v1"
	"github.com/stretchr/testify/require"
)

func TestPodValidate(t *testing.T) {
	pod := v1.Pod{
		Main: v1.PodVM{
			Name:  "main",
			Image: "main-image",
		},
		Additional: []v1.PodVM{
			{
				Name:  "db",
				Image: "db-image",
			},
		},
	}

	require.NoError(t, pod.Validate())
}

func TestPodValidateDuplicateVMNames(t *testing.T) {
	pod := v1.Pod{
		Main: v1.PodVM{
			Name:  "main",
			Image: "main-image",
		},
		Additional: []v1.PodVM{
			{
				Name:  "main",
				Image: "db-image",
			},
		},
	}

	require.EqualError(t, pod.Validate(), `pod VM name "main" is duplicated`)
}

func TestPodValidateConflictingPlatforms(t *testing.T) {
	pod := v1.Pod{
		Main: v1.PodVM{
			Name:  "main",
			Image: "main-image",
		},
		Additional: []v1.PodVM{
			{
				Name:   "db",
				Image:  "db-image",
				VMSpec: v1.VMSpec{Runtime: v1.RuntimeVetu},
			},
		},
	}

	require.EqualError(t, pod.Validate(), "all pod VMs must use the same os, arch and runtime")
}
