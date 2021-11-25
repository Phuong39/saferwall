// Copyright 2021 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"

	"github.com/digitalocean/go-libvirt"
	gonsq "github.com/nsqio/go-nsq"
	agent "github.com/saferwall/agent/pkg/grpc"
	"github.com/saferwall/saferwall/pkg/log"
	"github.com/saferwall/saferwall/pkg/pubsub"
	"github.com/saferwall/saferwall/pkg/pubsub/nsq"
	"github.com/saferwall/saferwall/pkg/utils"
	"github.com/saferwall/saferwall/pkg/vmmanager"
	"github.com/saferwall/saferwall/services/config"
	pb "github.com/saferwall/saferwall/services/proto"
	"google.golang.org/protobuf/proto"
)

// Config represents our application config.
type Config struct {
	LogLevel     string             `mapstructure:"log_level"`
	SharedVolume string             `mapstructure:"shared_volume"`
	SnapshotName string             `mapstructure:"snapshot_name"`
	Agent        AgentCfg           `mapstructure:"agent"`
	VirtMgr      VirtManagerCfg     `mapstructure:"libvirt"`
	Producer     config.ProducerCfg `mapstructure:"producer"`
	Consumer     config.ConsumerCfg `mapstructure:"consumer"`
}

// AgentCfg represents the guest agent config.
type AgentCfg struct {
	// Destinary directory inside the guest where the agent is deployed.
	AgentDestDir string `mapstructure:"dest_dir"`
	// The sandbox binary components.
	PackageName string `mapstructure:"package_name"`
}

// VirtManagerCfg represents the virtualization manager config.
// For now, only libvirt server.
type VirtManagerCfg struct {
	// Specify whether a remote or local session.
	// local session uses "unix" and ignore the fields below.
	Network string `mapstructure:"network"`
	// IP address of the host running libvirt RPC server.
	Address string `mapstructure:"address"`
	// Port number of the SSH server.
	Port string `mapstructure:"port"`
}

// VM represents a virtual machine config.
type VM struct {
	// ID identify uniquely the VM.
	ID int32
	// Name of the VM.
	Name string
	// IP address of the VM.
	IP string
	// Snapshots list names.
	Snapshots []string
	// InUse represents the availability of the VM.
	InUse bool
	// Pointer to the domain object.
	Dom *libvirt.Domain
}

// Service represents the sandbox scan service. It adheres to the nsq.Handler
// interface. This allows us to define our own custom handlers for our messages.
// Think of these handlers much like you would an http handler.
type Service struct {
	cfg     Config
	mu      sync.Mutex
	logger  log.Logger
	pub     pubsub.Publisher
	sub     pubsub.Subscriber
	vms     []VM
	vmm     vmmanager.VMManager
	sandbox []byte
}

type scanResult struct {
	res     agent.FileScanResult
	version string
}

func toJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// New create a new sandbox service.
func New(cfg Config, logger log.Logger) (*Service, error) {
	var err error
	s := Service{}

	// retrieve the list of active VMs.
	conn, err := vmmanager.New(cfg.VirtMgr.Network, cfg.VirtMgr.Address,
		cfg.VirtMgr.Port)
	if err != nil {
		return nil, err
	}
	logger.Info("Connection established to server")

	dd, err := conn.Domains()
	if err != nil {
		return nil, err
	}

	logger.Infof("Active domains: %v", dd)

	var vms []VM
	for _, d := range dd {
		vms = append(vms, VM{
			ID:        d.Dom.ID,
			Name:      d.Dom.Name,
			IP:        d.IP,
			Snapshots: d.Snapshots,
		})
	}

	// the number of concurrent workers have to match the number of
	// available virtual machines.
	s.sub, err = nsq.NewSubscriber(
		cfg.Consumer.Topic,
		cfg.Consumer.Channel,
		cfg.Consumer.Lookupds,
		len(vms),
		&s,
	)
	if err != nil {
		return nil, err
	}

	s.pub, err = nsq.NewPublisher(cfg.Producer.Nsqd)
	if err != nil {
		return nil, err
	}

	// download the sandbox release package.
	zipPackageData, err := utils.ReadAll(s.cfg.Agent.PackageName)
	if err != nil {
		return nil, err
	}

	s.sandbox = zipPackageData
	s.vms = vms
	s.cfg = cfg
	s.logger = logger
	s.vmm = conn
	return &s, nil

}

// Start kicks in the service to start consuming events.
func (s *Service) Start() error {
	s.logger.Infof("start consuming from topic: %s ...", s.cfg.Consumer.Topic)
	s.sub.Start()

	return nil
}

// HandleMessage is the only requirement needed to fulfill the nsq.Handler.
func (s *Service) HandleMessage(m *gonsq.Message) error {
	if len(m.Body) == 0 {
		return errors.New("body is blank re-enqueue message")
	}

	fileScanCfg := config.FileScanCfg{}
	ctx := context.Background()

	// Deserialize the msg sent from the web apis.
	err := json.Unmarshal(m.Body, &fileScanCfg)
	if err != nil {
		s.logger.Errorf("failed unmarshalling json messge body: %v", err)
		return err
	}

	sha256 := fileScanCfg.SHA256
	logger := s.logger.With(ctx, "sha256", sha256)
	logger.Info("start processing")

	// Find a free virtual machine to process this job.
	vm := s.findFreeVM()
	if vm == nil {
		logger.Infof("no VM currently available, call 911")
		return errors.New("failed to find a free VM")
	}

	logger = s.logger.With(ctx, "VM", vm.Name)
	logger.Infof("VM %s was selected", vm.Name)

	// Revert the VM to a clean state.
	err = s.vmm.Revert(*vm.Dom, s.cfg.SnapshotName)
	if err != nil {
		logger.Errorf("failed to revert the VM: %v", err)
	}

	// Perform the actual detonation.
	res, err := s.detonate(logger, vm, sha256, fileScanCfg.Dynamic)
	if err != nil {
		logger.Errorf("failed to detonation the sample: %v", err)
		s.freeVM(vm)
		return err
	}

	// Make sure to free the VM for next job.
	s.freeVM(vm)

	payloads := []*pb.Message_Payload{
		{Module: "sandbox", Body: toJSON(res)},
	}

	msg := &pb.Message{Sha256: sha256, Payload: payloads}
	peMsg, err := proto.Marshal(msg)
	if err != nil {
		logger.Errorf("failed to marshal message: %v", err)
		return err
	}

	err = s.pub.Publish(ctx, s.cfg.Producer.Topic, peMsg)
	if err != nil {
		logger.Errorf("failed to publish message: %v", err)
		return err
	}

	return nil
}

func (s *Service) detonate(logger log.Logger, vm *VM,
	sha256 string, cfg config.DynFileScanCfg) (scanResult, error) {

	ctx := context.Background()

	// Establish a gRPC connection to the agent server running
	// inside the guest.
	client, err := agent.New(vm.IP)
	if err != nil {
		logger.Errorf("failed to establish connection to server: %v", err)
		return scanResult{}, nil
	}

	// Deploy the sandbox component files inside the guest.
	ver, err := client.Deploy(ctx, s.cfg.Agent.AgentDestDir, s.sandbox)
	if err != nil {
		return scanResult{}, nil
	}
	logger.Infof("sandbox version %s has been deployed", ver)

	src := filepath.Join(s.cfg.SharedVolume, sha256)
	sampleContent, err := utils.ReadAll(src)
	if err != nil {
		return scanResult{}, nil
	}

	// Analyze the sample. This call will block until results
	// are ready.
	sandboxCfg := toJSON(cfg)
	res, err := client.Analyze(ctx, sandboxCfg, sampleContent)
	if err != nil {
		return scanResult{}, nil
	}

	return scanResult{res: res, version: ver}, nil

}

// findFreeVM iterates over the list of available VM and find
// one which is currently not in use.
func (s *Service) findFreeVM() *VM {
	var freeVM *VM
	s.mu.Lock()
	for _, vm := range s.vms {
		if !vm.InUse {
			vm.InUse = true
			freeVM = &vm
			break
		}
	}
	s.mu.Unlock()
	return freeVM
}

// freeVM makes the VM free for consumption.
func (s *Service) freeVM(vm *VM) {
	s.mu.Lock()
	vm.InUse = false
	s.mu.Unlock()
}
