package scheduler_test

import (
	"testing"
	"time"

	"github.com/cirruslabs/orchard/internal/controller/scheduler"
	v1 "github.com/cirruslabs/orchard/pkg/resource/v1"
	"github.com/stretchr/testify/require"
)

func TestProcessVMsSkipsUnscheduledPodVMs(t *testing.T) {
	unscheduled, _ := scheduler.ProcessVMs([]v1.VM{
		{
			Meta: v1.Meta{Name: "standalone"},
		},
		{
			Meta:    v1.Meta{Name: "pod-vm"},
			PodName: "pod",
		},
	})

	require.Len(t, unscheduled, 1)
	require.Equal(t, "standalone", unscheduled[0].Name)
}

func TestProcessPodsSortsByCreationTime(t *testing.T) {
	createdAt := time.Now()
	unscheduled := scheduler.ProcessPods([]v1.Pod{
		{
			Meta: v1.Meta{Name: "second", CreatedAt: createdAt.Add(time.Second)},
		},
		{
			Meta: v1.Meta{Name: "first", CreatedAt: createdAt},
		},
	})

	require.Len(t, unscheduled, 2)
	require.Equal(t, "first", unscheduled[0].Name)
	require.Equal(t, "second", unscheduled[1].Name)
}
