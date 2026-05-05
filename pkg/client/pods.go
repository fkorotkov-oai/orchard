package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cirruslabs/orchard/pkg/resource/v1"
	"github.com/coder/websocket"
)

type PodsService struct {
	client *Client
}

func (service *PodsService) Create(ctx context.Context, pod *v1.Pod) error {
	return service.client.request(ctx, http.MethodPost, "pods", pod, nil, nil)
}

func (service *PodsService) List(ctx context.Context) ([]v1.Pod, error) {
	var pods []v1.Pod
	err := service.client.request(ctx, http.MethodGet, "pods", nil, &pods, nil)
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func (service *PodsService) Get(ctx context.Context, name string) (*v1.Pod, error) {
	var pod v1.Pod
	err := service.client.request(ctx, http.MethodGet, fmt.Sprintf("pods/%s", url.PathEscape(name)),
		nil, &pod, nil)
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func (service *PodsService) Delete(ctx context.Context, name string) error {
	return service.client.request(ctx, http.MethodDelete, fmt.Sprintf("pods/%s", url.PathEscape(name)),
		nil, nil, nil)
}

func (service *PodsService) PortForward(
	ctx context.Context,
	podName string,
	vmName string,
	port uint16,
	waitSeconds uint16,
) (net.Conn, error) {
	return service.client.wsRequest(ctx, service.vmPath(podName, vmName, "port-forward"), map[string]string{
		"port": strconv.FormatUint(uint64(port), 10),
		"wait": strconv.FormatUint(uint64(waitSeconds), 10),
	})
}

func (service *PodsService) ExecSession(
	ctx context.Context,
	podName string,
	vmName string,
	options ExecSessionOptions,
) (*websocket.Conn, error) {
	params := map[string]string{
		"wait": strconv.FormatUint(uint64(options.WaitSeconds), 10),
	}
	if options.Command != "" {
		params["command"] = options.Command
	}
	if options.Interactive {
		params["interactive"] = strconv.FormatBool(true)
	}
	if options.TTY {
		params["tty"] = strconv.FormatBool(true)
	}
	if options.Rows > 0 {
		params["rows"] = strconv.FormatUint(uint64(options.Rows), 10)
	}
	if options.Cols > 0 {
		params["cols"] = strconv.FormatUint(uint64(options.Cols), 10)
	}
	for key, value := range options.Env {
		params[fmt.Sprintf("env[%s]", key)] = value
	}
	if options.Workdir != "" {
		params["workdir"] = options.Workdir
	}
	if options.Session != "" {
		params["session"] = options.Session
	}

	return service.client.wsRequestRaw(ctx, service.vmPath(podName, vmName, "exec"), params)
}

func (service *PodsService) IP(ctx context.Context, podName string, vmName string, waitSeconds uint16) (string, error) {
	result := struct {
		IP string `json:"ip"`
	}{}
	err := service.client.request(ctx, http.MethodGet, service.vmPath(podName, vmName, "ip"),
		nil, &result, map[string]string{"wait": strconv.FormatUint(uint64(waitSeconds), 10)})
	if err != nil {
		return "", err
	}
	return result.IP, nil
}

func (service *PodsService) LogsWithOptions(
	ctx context.Context,
	podName string,
	vmName string,
	options LogsOptions,
) (lines []string, err error) {
	var events []v1.Event
	params := map[string]string{}
	if options.Limit > 0 {
		params["limit"] = strconv.Itoa(options.Limit)
	}
	if options.Order != "" {
		params["order"] = string(options.Order)
	}
	if len(params) == 0 {
		params = nil
	}
	err = service.client.request(ctx, http.MethodGet, service.vmPath(podName, vmName, "events"), nil, &events, params)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.Kind == v1.EventKindLogLine {
			lines = append(lines, event.Payload)
		}
	}
	return lines, nil
}

func (service *PodsService) vmPath(podName string, vmName string, suffix string) string {
	if vmName == "" {
		return fmt.Sprintf("pods/%s/%s", url.PathEscape(podName), suffix)
	}
	return fmt.Sprintf("pods/%s/vms/%s/%s", url.PathEscape(podName), url.PathEscape(vmName), suffix)
}
