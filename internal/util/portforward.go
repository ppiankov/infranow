package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// PortForwardStatus represents the current state of port-forward
type PortForwardStatus int

const (
	StatusStopped PortForwardStatus = iota
	StatusStarting
	StatusRunning
	StatusFailed
)

func (s PortForwardStatus) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// PortForward manages Kubernetes port-forwarding using client-go
type PortForward struct {
	service    string
	namespace  string
	localPort  string
	remotePort string

	clientset  *kubernetes.Clientset
	restConfig *rest.Config

	mu           sync.RWMutex
	status       PortForwardStatus
	forwarder    *portforward.PortForwarder
	stopChan     chan struct{}
	readyChan    chan struct{}
	lastError    error
	startTime    time.Time
	restartCount int
}

// NewPortForward creates a new native Go port-forward manager
func NewPortForward(service, namespace, localPort, remotePort string) (*PortForward, error) {
	// Load kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Build config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &PortForward{
		service:    service,
		namespace:  namespace,
		localPort:  localPort,
		remotePort: remotePort,
		clientset:  clientset,
		restConfig: config,
		status:     StatusStopped,
	}, nil
}

// Start initiates the port-forward
func (pf *PortForward) Start() error {
	pf.mu.Lock()
	if pf.status == StatusRunning || pf.status == StatusStarting {
		pf.mu.Unlock()
		return fmt.Errorf("port-forward already running")
	}
	pf.status = StatusStarting
	pf.mu.Unlock()

	// Get service to find a pod
	svc, err := pf.clientset.CoreV1().Services(pf.namespace).Get(context.Background(), pf.service, metav1.GetOptions{})
	if err != nil {
		pf.setStatus(StatusFailed, fmt.Errorf("failed to get service: %w", err))
		return err
	}

	// Find a pod matching service selector
	pods, err := pf.clientset.CoreV1().Pods(pf.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: svc.Spec.Selector}),
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		pf.setStatus(StatusFailed, fmt.Errorf("failed to list pods: %w", err))
		return err
	}

	if len(pods.Items) == 0 {
		err := fmt.Errorf("no running pods found for service %s", pf.service)
		pf.setStatus(StatusFailed, err)
		return err
	}

	// Use first running pod
	pod := pods.Items[0]

	// Setup port-forward
	if err := pf.setupPortForward(pod.Name); err != nil {
		pf.setStatus(StatusFailed, err)
		return err
	}

	pf.mu.Lock()
	pf.status = StatusRunning
	pf.startTime = time.Now()
	pf.restartCount++
	pf.mu.Unlock()

	return nil
}

// setupPortForward configures the actual port-forward connection
func (pf *PortForward) setupPortForward(podName string) error {
	// Build URL for port-forward
	req := pf.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pf.namespace).
		Name(podName).
		SubResource("portforward")

	// Create SPDY transport
	transport, upgrader, err := spdy.RoundTripperFor(pf.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create SPDY transport: %w", err)
	}

	// Create dialer
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	// Setup channels
	pf.stopChan = make(chan struct{}, 1)
	pf.readyChan = make(chan struct{}, 1)

	// Create port-forwarder
	ports := []string{fmt.Sprintf("%s:%s", pf.localPort, pf.remotePort)}
	fw, err := portforward.New(dialer, ports, pf.stopChan, pf.readyChan, io.Discard, io.Discard)
	if err != nil {
		return fmt.Errorf("failed to create port-forwarder: %w", err)
	}

	pf.forwarder = fw

	// Start forwarding in background
	go func() {
		if err := fw.ForwardPorts(); err != nil {
			pf.setStatus(StatusFailed, err)
		}
	}()

	// Wait for ready or timeout
	select {
	case <-pf.readyChan:
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for port-forward to be ready")
	}
}

// Stop terminates the port-forward
func (pf *PortForward) Stop() error {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	if pf.status == StatusStopped {
		return nil
	}

	if pf.stopChan != nil {
		close(pf.stopChan)
		pf.stopChan = nil
	}

	pf.status = StatusStopped
	pf.forwarder = nil

	return nil
}

// IsRunning checks if the port-forward is active
func (pf *PortForward) IsRunning() bool {
	pf.mu.RLock()
	defer pf.mu.RUnlock()
	return pf.status == StatusRunning
}

// GetStatus returns current status
func (pf *PortForward) GetStatus() PortForwardStatus {
	pf.mu.RLock()
	defer pf.mu.RUnlock()
	return pf.status
}

// GetStatusString returns status with details
func (pf *PortForward) GetStatusString() string {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	switch pf.status {
	case StatusRunning:
		uptime := time.Since(pf.startTime).Round(time.Second)
		return fmt.Sprintf("running (%s, restart #%d)", uptime, pf.restartCount)
	case StatusFailed:
		if pf.lastError != nil {
			return fmt.Sprintf("failed: %v", pf.lastError)
		}
		return "failed"
	default:
		return pf.status.String()
	}
}

// GetLastError returns the last error encountered
func (pf *PortForward) GetLastError() error {
	pf.mu.RLock()
	defer pf.mu.RUnlock()
	return pf.lastError
}

// Restart stops and restarts the port-forward
func (pf *PortForward) Restart() error {
	if err := pf.Stop(); err != nil {
		return fmt.Errorf("failed to stop port-forward: %w", err)
	}

	// Give it a moment to clean up
	time.Sleep(1 * time.Second)

	if err := pf.Start(); err != nil {
		return fmt.Errorf("failed to restart port-forward: %w", err)
	}

	return nil
}

// setStatus updates status with error
func (pf *PortForward) setStatus(status PortForwardStatus, err error) {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	pf.status = status
	pf.lastError = err
}

// GetInfo returns port-forward information
func (pf *PortForward) GetInfo() map[string]string {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	info := map[string]string{
		"service":     pf.service,
		"namespace":   pf.namespace,
		"local_port":  pf.localPort,
		"remote_port": pf.remotePort,
		"status":      pf.status.String(),
	}

	if pf.status == StatusRunning {
		info["uptime"] = time.Since(pf.startTime).Round(time.Second).String()
		info["restarts"] = fmt.Sprintf("%d", pf.restartCount)
	}

	return info
}
