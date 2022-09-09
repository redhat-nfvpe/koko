package api

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"

	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/util"
)

var (
	// RuntimeEndpoint is CRI server runtime endpoint
	RuntimeEndpoint string
	// Timeout  of connecting to server (default: 10s)
	Timeout time.Duration
)

func getRuntimeClientConnection() (*grpc.ClientConn, error) {
	//return nil, fmt.Errorf("--runtime-endpoint is not set")
	RuntimeEndpoint = "unix:///var/run/crio/crio.sock"
	Timeout = 10 * time.Second

	addr, dialer, err := util.GetAddressAndDialer(RuntimeEndpoint)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(Timeout), grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, fmt.Errorf("failed to connect, make sure you are running as root and the runtime has been started: %v", err)
	}
	return conn, nil
}

// GetCrioRuntimeClient retrieves crio grpc client
func GetCrioRuntimeClient() (pb.RuntimeServiceClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := getRuntimeClientConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %v", err)
	}
	runtimeClient := pb.NewRuntimeServiceClient(conn)
	return runtimeClient, conn, nil
}

// CloseCrioConnection closes grpc connection in client
func CloseCrioConnection(conn *grpc.ClientConn) error {
	if conn == nil {
		return nil
	}
	return conn.Close()
}

// GetCrioContainerNS retrieves container's network namespace from
// cri-o container id, given as containerID.
func GetCrioContainerNS(runtimeClient pb.RuntimeServiceClient, procPrefix, containerID string) (namespace string, err error) {
	if err != nil {
		return "", err
	}

	request := &pb.ContainerStatusRequest{
		ContainerId: containerID,
		Verbose:     true,
	}
	r, err := runtimeClient.ContainerStatus(context.TODO(), request)
	if err != nil {
		return "", err
	}

	var prefix string
	if procPrefix == "" {
		prefix = ""
	} else {
		prefix = fmt.Sprintf("%s", procPrefix)
	}
	namespace = fmt.Sprintf("%s/proc/%s/ns/net", prefix, r.Info["pid"])

	return namespace, err
}
