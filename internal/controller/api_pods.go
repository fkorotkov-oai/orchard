package controller

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	storepkg "github.com/cirruslabs/orchard/internal/controller/store"
	"github.com/cirruslabs/orchard/internal/responder"
	"github.com/cirruslabs/orchard/internal/simplename"
	"github.com/cirruslabs/orchard/pkg/resource/v1"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (controller *Controller) createPod(ctx *gin.Context) responder.Responder {
	if responder := controller.authorize(ctx, v1.ServiceAccountRoleComputeWrite); responder != nil {
		return responder
	}

	var pod v1.Pod
	if err := ctx.ShouldBindJSON(&pod); err != nil {
		return responder.JSON(http.StatusBadRequest, NewErrorResponse("invalid JSON was provided"))
	}

	if pod.Name == "" {
		return responder.JSON(http.StatusPreconditionFailed, NewErrorResponse("Pod name is empty"))
	} else if err := simplename.Validate(pod.Name); err != nil {
		return responder.JSON(http.StatusPreconditionFailed, NewErrorResponse("Pod name %v", err))
	}
	if err := pod.Validate(); err != nil {
		return responder.JSON(http.StatusPreconditionFailed, NewErrorResponse("%v", err))
	}

	pod.UID = uuid.New().String()
	pod.CreatedAt = time.Now()
	pod.Status = v1.PodStatusPending
	pod.VMs = map[string]v1.PodMemberStatus{}

	members := pod.Members()
	vms := make([]v1.VM, 0, len(members))
	for index := range members {
		member := members[index]
		if err := simplename.Validate(member.Name); err != nil {
			return responder.JSON(http.StatusPreconditionFailed,
				NewErrorResponse("Pod VM name %q %v", member.Name, err))
		}
		if member.Image == "" {
			return responder.JSON(http.StatusPreconditionFailed,
				NewErrorResponse("Pod VM %q image is empty", member.Name))
		}

		vm := member.ToVM(podVMResourceName(pod.Name, member.Name), pod.Name, index == 0)
		if responder := controller.prepareVM(&vm); responder != nil {
			return responder
		}

		pod.VMs[member.Name] = v1.PodMemberStatus{
			Name:   vm.Name,
			Status: vm.Status,
			Worker: vm.Worker,
		}
		vms = append(vms, vm)
	}

	response := controller.storeUpdate(func(txn storepkg.Transaction) responder.Responder {
		if _, err := txn.GetPod(pod.Name); err == nil {
			return responder.JSON(http.StatusConflict, NewErrorResponse("Pod with this name already exists"))
		} else if !errors.Is(err, storepkg.ErrNotFound) {
			return responder.Error(err)
		}

		for _, vm := range vms {
			if _, err := txn.GetVM(vm.Name); err == nil {
				return responder.JSON(http.StatusConflict,
					NewErrorResponse("Pod VM %q conflicts with an existing VM", vm.Name))
			} else if !errors.Is(err, storepkg.ErrNotFound) {
				return responder.Error(err)
			}
		}

		for _, vm := range vms {
			if err := txn.SetVM(vm); err != nil {
				return responder.Error(err)
			}
		}
		if err := txn.SetPod(pod); err != nil {
			return responder.Error(err)
		}

		return responder.JSON(http.StatusOK, &pod)
	})
	controller.scheduler.RequestScheduling()
	return response
}

func (controller *Controller) getPod(ctx *gin.Context) responder.Responder {
	if responder := controller.authorize(ctx, v1.ServiceAccountRoleComputeRead); responder != nil {
		return responder
	}

	return controller.storeView(func(txn storepkg.Transaction) responder.Responder {
		pod, err := txn.GetPod(ctx.Param("name"))
		if err != nil {
			return responder.Error(err)
		}

		if err := hydratePodState(txn, pod); err != nil {
			return responder.Error(err)
		}

		return responder.JSON(http.StatusOK, pod)
	})
}

func (controller *Controller) listPods(ctx *gin.Context) responder.Responder {
	if responder := controller.authorize(ctx, v1.ServiceAccountRoleComputeRead); responder != nil {
		return responder
	}

	return controller.storeView(func(txn storepkg.Transaction) responder.Responder {
		pods, err := txn.ListPods()
		if err != nil {
			return responder.Error(err)
		}

		for index := range pods {
			if err := hydratePodState(txn, &pods[index]); err != nil {
				return responder.Error(err)
			}
		}

		return responder.JSON(http.StatusOK, pods)
	})
}

func (controller *Controller) deletePod(ctx *gin.Context) responder.Responder {
	if responder := controller.authorize(ctx, v1.ServiceAccountRoleComputeWrite); responder != nil {
		return responder
	}

	return controller.storeUpdate(func(txn storepkg.Transaction) responder.Responder {
		pod, err := txn.GetPod(ctx.Param("name"))
		if err != nil {
			return responder.Error(err)
		}

		for _, member := range pod.Members() {
			vmName := podVMResourceName(pod.Name, member.Name)
			vm, err := txn.GetVM(vmName)
			if err != nil && !errors.Is(err, storepkg.ErrNotFound) {
				return responder.Error(err)
			}
			if vm != nil {
				if err := txn.DeleteVM(vm.Name); err != nil {
					return responder.Error(err)
				}
				if err := txn.DeleteEvents("vms", vm.UID); err != nil {
					return responder.Error(err)
				}
			}
		}

		if err := txn.DeletePod(pod.Name); err != nil {
			return responder.Error(err)
		}

		return responder.Code(http.StatusOK)
	})
}

func hydratePodState(txn storepkg.Transaction, pod *v1.Pod) error {
	pod.Status = v1.PodStatusRunning
	pod.StatusMessage = ""
	pod.VMs = map[string]v1.PodMemberStatus{}

	for _, member := range pod.Members() {
		vm, err := txn.GetVM(podVMResourceName(pod.Name, member.Name))
		if err != nil {
			return err
		}

		pod.VMs[member.Name] = v1.PodMemberStatus{
			Name:          vm.Name,
			Status:        vm.Status,
			StatusMessage: vm.StatusMessage,
			Worker:        vm.Worker,
		}

		switch vm.Status {
		case v1.VMStatusFailed:
			pod.Status = v1.PodStatusFailed
			pod.StatusMessage = vm.StatusMessage
		case v1.VMStatusPending:
			if pod.Status != v1.PodStatusFailed {
				pod.Status = v1.PodStatusPending
			}
		}
	}

	return nil
}

func podVMResourceName(podName string, vmName string) string {
	return fmt.Sprintf("%s:%s", podName, vmName)
}
