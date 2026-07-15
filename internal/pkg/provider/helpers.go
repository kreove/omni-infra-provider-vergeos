// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"fmt"
	"math"
	"strings"

	vergeos "github.com/verge-io/govergeos"

	"github.com/your-org/omni-infra-provider-vergeos/internal/pkg/provider/data"
)

func validateProviderData(value data.Data) error {
	if value.ClusterID < 1 {
		return fmt.Errorf("cluster_id must be greater than zero")
	}

	if value.VNetID < 1 {
		return fmt.Errorf("vnet_id must be greater than zero")
	}

	if value.ImageFileID < 0 {
		return fmt.Errorf("image_file_id cannot be negative")
	}

	if value.Architecture != "amd64" {
		return fmt.Errorf("architecture %q is not supported by this alpha provider; use amd64", value.Architecture)
	}

	if value.Cores < 1 {
		return fmt.Errorf("cores must be greater than zero")
	}

	if value.Memory < 2048 {
		return fmt.Errorf("memory must be at least 2048 MiB")
	}

	if value.Memory > math.MaxInt {
		return fmt.Errorf("memory value is too large")
	}

	if value.DiskSize < 5 {
		return fmt.Errorf("disk_size must be at least 5 GiB")
	}

	return nil
}

func applyDefaults(value *data.Data) {
	if value.Architecture == "" {
		value.Architecture = "amd64"
	}

	if value.DiskInterface == "" {
		value.DiskInterface = "virtio-scsi"
	}

	if value.NetworkInterface == "" {
		value.NetworkInterface = "virtio"
	}

	if value.UEFI == nil {
		uefi := true
		value.UEFI = &uefi
	}
}

func noCloudMetaData(name string) string {
	return fmt.Sprintf("instance-id: %q\nlocal-hostname: %q\n", name, name)
}

func filterValue(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func vmMachineID(vm *vergeos.VM) (int, error) {
	if vm.Machine > 0 {
		return vm.Machine, nil
	}

	return 0, fmt.Errorf("VergeOS VM %q did not return an internal machine ID", vm.Name)
}
