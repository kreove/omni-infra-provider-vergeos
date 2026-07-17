// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package provider implements the VergeOS Omni infrastructure provider.
package provider

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/siderolabs/omni/client/pkg/infra/provision"
	"github.com/siderolabs/omni/client/pkg/omni/resources/infra"
	vergeos "github.com/verge-io/govergeos"
	"go.uber.org/zap"

	"github.com/kreove/omni-infra-provider-vergeos/internal/pkg/provider/data"
	"github.com/kreove/omni-infra-provider-vergeos/internal/pkg/provider/resources"
)

const (
	bootDiskName   = "disk0"
	primaryNICName = "nic0"
)

// Provisioner provisions Talos VMs in VergeOS.
type Provisioner struct {
	client              *vergeos.Client
	imageFactoryBaseURL string
	imageLocks          sync.Map
}

// NewProvisioner creates a VergeOS provisioner.
func NewProvisioner(client *vergeos.Client, imageFactoryBaseURL string) *Provisioner {
	return &Provisioner{
		client:              client,
		imageFactoryBaseURL: imageFactoryBaseURL,
	}
}

// ProvisionSteps implements infra.Provisioner.
func (p *Provisioner) ProvisionSteps() []provision.Step[*resources.Machine] {
	return []provision.Step[*resources.Machine]{
		provision.NewStep("validateRequest", func(_ context.Context, _ *zap.Logger, pctx provision.Context[*resources.Machine]) error {
			if len(pctx.GetRequestID()) > 63 {
				return fmt.Errorf("machine request name cannot be longer than 63 characters")
			}

			var providerData data.Data
			if err := pctx.UnmarshalProviderData(&providerData); err != nil {
				return err
			}

			applyDefaults(&providerData)

			return validateProviderData(providerData)
		}),
		provision.NewStep("createSchematic", func(ctx context.Context, logger *zap.Logger, pctx provision.Context[*resources.Machine]) error {
			schematic, err := pctx.GenerateSchematicID(
				ctx,
				logger,
				provision.WithExtraKernelArgs("console=ttyS0,38400n8"),
				provision.WithoutConnectionParams(),
			)
			if err != nil {
				return err
			}

			pctx.State.TypedSpec().Value.Schematic = schematic
			pctx.State.TypedSpec().Value.TalosVersion = pctx.GetTalosVersion()

			return nil
		}),
		provision.NewStep("ensureTarget", func(ctx context.Context, _ *zap.Logger, pctx provision.Context[*resources.Machine]) error {
			var providerData data.Data
			if err := pctx.UnmarshalProviderData(&providerData); err != nil {
				return err
			}

			applyDefaults(&providerData)

			if _, err := p.client.Clusters.Get(ctx, providerData.ClusterID); err != nil {
				return fmt.Errorf("failed to resolve VergeOS cluster %d: %w", providerData.ClusterID, err)
			}

			if _, err := p.client.Networks.Get(ctx, providerData.VNetID); err != nil {
				return fmt.Errorf("failed to resolve VergeOS VNET %d: %w", providerData.VNetID, err)
			}

			return nil
		}),
		provision.NewStep("ensureImage", func(ctx context.Context, logger *zap.Logger, pctx provision.Context[*resources.Machine]) error {
			var providerData data.Data
			if err := pctx.UnmarshalProviderData(&providerData); err != nil {
				return err
			}

			applyDefaults(&providerData)

			imageID, ready, err := p.ensureTalosImage(ctx, logger, pctx, providerData)
			if err != nil {
				return err
			}

			if !ready {
				return provision.NewRetryInterval(10 * time.Second)
			}

			pctx.State.TypedSpec().Value.VolumeId = strconv.Itoa(imageID)

			return nil
		}),
		provision.NewStep("syncMachine", func(ctx context.Context, logger *zap.Logger, pctx provision.Context[*resources.Machine]) error {
			var providerData data.Data
			if err := pctx.UnmarshalProviderData(&providerData); err != nil {
				return err
			}

			applyDefaults(&providerData)

			vm, err := p.findVM(ctx, pctx.GetRequestID())
			if err != nil {
				return err
			}

			if vm == nil {
				createdVM, createErr := p.createVM(ctx, pctx, providerData)
				if createErr != nil {
					return createErr
				}

				pctx.State.TypedSpec().Value.Uuid = createdVM.UUID

				logger.Info(
					"created VergeOS VM",
					zap.String("name", createdVM.Name),
					zap.Int("id", createdVM.ID.Int()),
					zap.Int("machine_id", createdVM.Machine),
				)

				return provision.NewRetryInterval(5 * time.Second)
			}

			if pctx.State.TypedSpec().Value.Uuid == "" {
				pctx.State.TypedSpec().Value.Uuid = vm.UUID
			}

			if err = p.ensureVMSettings(ctx, vm, providerData); err != nil {
				return err
			}
			
			machineID, err := vmMachineID(vm)
			if err != nil {
				return err
			}

			imageFileID, err := strconv.Atoi(pctx.State.TypedSpec().Value.VolumeId)
			if err != nil || imageFileID < 1 {
				return fmt.Errorf("invalid VergeOS image file ID %q", pctx.State.TypedSpec().Value.VolumeId)
			}

			if err = p.ensureBootDisk(ctx, machineID, imageFileID, providerData); err != nil {
				return err
			}

			if err = p.ensureNIC(ctx, machineID, providerData); err != nil {
				return err
			}

			if !vm.PowerState {
				if err = p.client.VMs.PowerOn(ctx, vm.ID.Int()); err != nil {
					if vergeos.IsTimeoutError(err) {
						return provision.NewRetryInterval(5 * time.Second)
					}

					return fmt.Errorf("failed to power on VergeOS VM %q: %w", vm.Name, err)
				}

				return provision.NewRetryInterval(5 * time.Second)
			}

			logger.Info(
				"machine is running",
				zap.String("name", vm.Name),
				zap.Int("id", vm.ID.Int())
			)

			return nil
		}),
	}
}

func (p *Provisioner) createVM(
	ctx context.Context,
	pctx provision.Context[*resources.Machine],
	providerData data.Data,
) (*vergeos.VM, error) {
	enabled := true
	serialPort := true
	secureBoot := false
	guestAgent := providerData.GuestAgent

	vm, err := p.client.VMs.Create(ctx, &vergeos.VMCreateRequest{
		Name:                pctx.GetRequestID(),
		Description:         "Talos machine managed by Sidero Omni",
		Enabled:             &enabled,
		Cluster:             &providerData.ClusterID,
		CPUCores:            providerData.Cores,
		CPUType:             providerData.CPUType,
		RAM:                 int(providerData.Memory),
		MachineType:         providerData.MachineType,
		UEFI:                providerData.UEFI,
		SecureBoot:          &secureBoot,
		Console:             "vnc",
		SerialPort:          &serialPort,
		OSFamily:            "linux",
		OSDescription:       "Talos Linux",
		GuestAgent:          &guestAgent,
		CreatedFrom:         "custom",
		CloudInitDataSource: "nocloud",
		CloudInitFiles: []vergeos.CloudInitFileRef{
			{
				Name:     "user-data",
				Contents: pctx.ConnectionParams.JoinConfig,
			},
			{
				Name:     "meta-data",
				Contents: noCloudMetaData(pctx.GetRequestID()),
			},
			{
				Name:     "network-config",
				Contents: "version: 1\n",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create VergeOS VM %q: %w", pctx.GetRequestID(), err)
	}

	return vm, nil
}

func (p *Provisioner) ensureVMSettings(
	ctx context.Context,
	vm *vergeos.VM,
	providerData data.Data,
) error {
	if vm.GuestAgent == providerData.GuestAgent {
		return nil
	}

	guestAgent := providerData.GuestAgent

	_, err := p.client.VMs.Update(
		ctx,
		vm.ID.Int(),
		&vergeos.VMUpdateRequest{
			GuestAgent: &guestAgent,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"failed to update QEMU Guest Agent setting for VM %q: %w",
			vm.Name,
			err,
		)
	}

	return provision.NewRetryInterval(5 * time.Second)
}

func (p *Provisioner) ensureBootDisk(ctx context.Context, machineID, imageFileID int, providerData data.Data) error {
	drive, err := p.client.VMDrives.GetByName(ctx, machineID, bootDiskName)
	if err == nil {
		if drive.Status == "importing" {
			return provision.NewRetryInterval(5 * time.Second)
		}

		return nil
	}

	if !vergeos.IsNotFoundError(err) {
		return fmt.Errorf("failed to inspect boot disk: %w", err)
	}

	orderID := 0

	_, err = p.client.VMDrives.Create(ctx, machineID, &vergeos.VMDriveCreateRequest{
		OrderID:       &orderID,
		Name:          bootDiskName,
		Description:   "Talos boot disk managed by Sidero Omni",
		Interface:     providerData.DiskInterface,
		Media:         "import",
		File:          imageFileID,
		SizeGB:        providerData.DiskSize,
		PreferredTier: providerData.PreferredTier,
	})
	if err != nil {
		if vergeos.IsTimeoutError(err) {
			return provision.NewRetryInterval(5 * time.Second)
		}

		return fmt.Errorf("failed to create/import Talos boot disk: %w", err)
	}

	return nil
}

func (p *Provisioner) ensureNIC(ctx context.Context, machineID int, providerData data.Data) error {
	nics, err := p.client.VMNICs.List(ctx, machineID)
	if err != nil {
		return fmt.Errorf("failed to list VergeOS NICs: %w", err)
	}

	for _, nic := range nics {
		if nic.Name == primaryNICName {
			return nil
		}
	}

	orderID := 0

	_, err = p.client.VMNICs.Create(ctx, machineID, &vergeos.VMNICCreateRequest{
		OrderID:     &orderID,
		Name:        primaryNICName,
		Description: "Talos primary NIC managed by Sidero Omni",
		Interface:   providerData.NetworkInterface,
		VNET:        providerData.VNetID,
	})
	if err != nil {
		return fmt.Errorf("failed to create VergeOS NIC: %w", err)
	}

	return nil
}

func (p *Provisioner) findVM(ctx context.Context, name string) (*vergeos.VM, error) {
	vms, err := p.client.VMs.List(ctx, vergeos.WithFilter(fmt.Sprintf("name eq '%s'", filterValue(name))))
	if err != nil {
		return nil, fmt.Errorf("failed to query VergeOS VM %q: %w", name, err)
	}

	if len(vms) == 0 {
		return nil, nil
	}

	if len(vms) > 1 {
		return nil, fmt.Errorf("multiple VergeOS VMs have the name %q", name)
	}

	vm, err := p.client.VMs.Get(ctx, vms[0].ID.Int())
	if err != nil {
		return nil, fmt.Errorf("failed to read VergeOS VM %q: %w", name, err)
	}

	return vm, nil
}

// Deprovision implements infra.Provisioner.
func (p *Provisioner) Deprovision(
	ctx context.Context,
	logger *zap.Logger,
	_ *resources.Machine,
	machineRequest *infra.MachineRequest,
) error {
	vm, err := p.findVM(ctx, machineRequest.Metadata().ID())
	if err != nil {
		return err
	}

	if vm == nil {
		logger.Info("machine deprovisioned")

		return nil
	}

	if vm.PowerState {
		if err = p.client.VMs.PowerOff(ctx, vm.ID.Int()); err != nil {
			if vergeos.IsTimeoutError(err) {
				return provision.NewRetryInterval(5 * time.Second)
			}

			return fmt.Errorf("failed to power off VergeOS VM %q: %w", vm.Name, err)
		}

		return provision.NewRetryInterval(5 * time.Second)
	}

	machineID, err := vmMachineID(vm)
	if err != nil {
		return err
	}

	nics, err := p.client.VMNICs.List(ctx, machineID)
	if err != nil {
		return fmt.Errorf("failed to list NICs while deleting VM %q: %w", vm.Name, err)
	}

	for _, nic := range nics {
		if err = p.client.VMNICs.Delete(ctx, nic.ID.Int()); err != nil && !vergeos.IsNotFoundError(err) {
			return fmt.Errorf("failed to delete NIC %d from VM %q: %w", nic.ID.Int(), vm.Name, err)
		}
	}

	drives, err := p.client.VMDrives.List(ctx, machineID)
	if err != nil {
		return fmt.Errorf("failed to list drives while deleting VM %q: %w", vm.Name, err)
	}

	for _, drive := range drives {
		if err = p.client.VMDrives.Delete(ctx, drive.ID.Int()); err != nil && !vergeos.IsNotFoundError(err) {
			return fmt.Errorf("failed to delete drive %d from VM %q: %w", drive.ID.Int(), vm.Name, err)
		}
	}

	if err = p.client.VMs.Delete(ctx, vm.ID.Int()); err != nil && !vergeos.IsNotFoundError(err) {
		return fmt.Errorf("failed to delete VergeOS VM %q: %w", vm.Name, err)
	}

	return provision.NewRetryInterval(5 * time.Second)
}
