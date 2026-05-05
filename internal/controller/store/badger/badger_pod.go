//nolint:dupl // maybe we'll figure out how to make DB resource accessors generic in the future
package badger

import (
	"path"

	"github.com/cirruslabs/orchard/pkg/resource/v1"
)

const SpacePods = "/pods"

func PodKey(name string) []byte {
	return []byte(path.Join(SpacePods, name))
}

func (txn *Transaction) GetPod(name string) (*v1.Pod, error) {
	return genericGet[v1.Pod](txn, PodKey(name))
}

func (txn *Transaction) SetPod(pod v1.Pod) error {
	return genericSet[v1.Pod](txn, PodKey(pod.Name), pod)
}

func (txn *Transaction) DeletePod(name string) error {
	return genericDelete(txn, PodKey(name))
}

func (txn *Transaction) ListPods() ([]v1.Pod, error) {
	return genericList[v1.Pod](txn, SpacePods)
}
