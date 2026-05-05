package v1

import (
	"fmt"
	"time"
)

type Pod struct {
	Main       PodVM   `json:"main"`
	Additional []PodVM `json:"additional,omitempty"`

	UID string `json:"uid,omitempty"`

	Worker      string    `json:"worker,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`

	PodState
	Meta
}

func (pod *Pod) SetVersion(version uint64) {
	pod.Version = version
}

func (pod *Pod) IsScheduled() bool {
	return pod.Worker != ""
}

func (pod *Pod) Members() []PodVM {
	members := make([]PodVM, 0, 1+len(pod.Additional))
	members = append(members, pod.Main)
	members = append(members, pod.Additional...)
	return members
}

func (pod *Pod) Validate() error {
	if pod.Main.Name == "" {
		return fmt.Errorf("main VM name is empty")
	}

	names := map[string]struct{}{}
	labels := Labels{}
	var platform *PodPlatform

	for _, member := range pod.Members() {
		if member.Name == "" {
			return fmt.Errorf("pod VM name is empty")
		}
		if _, ok := names[member.Name]; ok {
			return fmt.Errorf("pod VM name %q is duplicated", member.Name)
		}
		names[member.Name] = struct{}{}

		if err := member.Validate(); err != nil {
			return err
		}
		for key, value := range member.Labels {
			if existingValue, ok := labels[key]; ok && existingValue != value {
				return fmt.Errorf("pod VMs require conflicting values for label %q", key)
			}
			labels[key] = value
		}

		memberPlatform := member.Platform()
		if platform == nil {
			platform = &memberPlatform
			continue
		}
		if *platform != memberPlatform {
			return fmt.Errorf("all pod VMs must use the same os, arch and runtime")
		}
	}

	return nil
}

type PodVM struct {
	Name string `json:"name,omitempty"`

	Image           string          `json:"image,omitempty"`
	ImagePullPolicy ImagePullPolicy `json:"imagePullPolicy,omitempty"`
	CPU             uint64          `json:"cpu,omitempty"`
	Memory          uint64          `json:"memory,omitempty"`
	DiskSize        uint64          `json:"diskSize,omitempty"`
	NetBridged      string          `json:"net-bridged,omitempty"`
	Headless        bool            `json:"headless,omitempty"`
	Nested          bool            `json:"nested,omitempty"`

	VMSpec

	Username      string    `json:"username,omitempty"`
	Password      string    `json:"password,omitempty"`
	BootScript    *VMScript `json:"boot_script,omitempty"`
	StartupScript *VMScript `json:"startup_script,omitempty"`

	RestartPolicy RestartPolicy `json:"restart_policy,omitempty"`
	RandomSerial  bool          `json:"randomSerial,omitempty"`
	Resources     Resources     `json:"resources,omitempty"`
	Labels        Labels        `json:"labels,omitempty"`
	HostDirs      []HostDir     `json:"hostDirs,omitempty"`
}

func (vm *PodVM) Validate() error {
	asVM := vm.ToVM("", "", false)
	return asVM.Validate()
}

func (vm *PodVM) Platform() PodPlatform {
	os := vm.OS
	if os == "" {
		os = OSDarwin
	}
	arch := vm.Arch
	if arch == "" {
		arch = ArchitectureARM64
	}
	runtime := vm.Runtime
	if runtime == "" {
		runtime = RuntimeTart
	}

	return PodPlatform{
		OS:      os,
		Arch:    arch,
		Runtime: runtime,
	}
}

func (vm *PodVM) ToVM(name string, podName string, podMain bool) VM {
	return VM{
		Image:           vm.Image,
		ImagePullPolicy: vm.ImagePullPolicy,
		CPU:             vm.CPU,
		Memory:          vm.Memory,
		DiskSize:        vm.DiskSize,
		NetBridged:      vm.NetBridged,
		Headless:        vm.Headless,
		Nested:          vm.Nested,
		VMSpec:          vm.VMSpec,
		Username:        vm.Username,
		Password:        vm.Password,
		BootScript:      vm.BootScript,
		StartupScript:   vm.StartupScript,
		RestartPolicy:   vm.RestartPolicy,
		RandomSerial:    vm.RandomSerial,
		Resources:       vm.Resources,
		Labels:          vm.Labels,
		HostDirs:        vm.HostDirs,
		PodName:         podName,
		PodVMName:       vm.Name,
		PodMain:         podMain,
		Meta: Meta{
			Name: name,
		},
	}
}

type PodPlatform struct {
	OS      OS
	Arch    Architecture
	Runtime Runtime
}

type PodState struct {
	Status        PodStatus                  `json:"status,omitempty"`
	StatusMessage string                     `json:"status_message,omitempty"`
	VMs           map[string]PodMemberStatus `json:"vms,omitempty"`
}

type PodMemberStatus struct {
	Name          string   `json:"name,omitempty"`
	Status        VMStatus `json:"status,omitempty"`
	StatusMessage string   `json:"status_message,omitempty"`
	Worker        string   `json:"worker,omitempty"`
}

type PodStatus string

const (
	PodStatusPending PodStatus = "pending"
	PodStatusRunning PodStatus = "running"
	PodStatusFailed  PodStatus = "failed"
)
