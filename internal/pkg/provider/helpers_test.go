// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"testing"

	"github.com/your-org/omni-infra-provider-vergeos/internal/pkg/provider/data"
)

func TestApplyDefaults(t *testing.T) {
	t.Parallel()

	value := data.Data{}
	applyDefaults(&value)

	if value.Architecture != "amd64" {
		t.Fatalf("unexpected architecture %q", value.Architecture)
	}

	if value.DiskInterface != "virtio-scsi" {
		t.Fatalf("unexpected disk interface %q", value.DiskInterface)
	}

	if value.NetworkInterface != "virtio" {
		t.Fatalf("unexpected network interface %q", value.NetworkInterface)
	}

	if value.UEFI == nil || !*value.UEFI {
		t.Fatal("UEFI should default to true")
	}
}

func TestFilterValue(t *testing.T) {
	t.Parallel()

	if got, want := filterValue("node's-name"), "node''s-name"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNoCloudMetaData(t *testing.T) {
	t.Parallel()

	got := noCloudMetaData("talos-worker-01")
	want := "instance-id: \"talos-worker-01\"\nlocal-hostname: \"talos-worker-01\"\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidateProviderDataAllowsAutomaticImage(t *testing.T) {
	t.Parallel()

	uefi := true
	value := data.Data{
		ClusterID:        1,
		VNetID:           2,
		Architecture:     "amd64",
		Cores:            2,
		Memory:           4096,
		DiskSize:         20,
		DiskInterface:    "virtio-scsi",
		NetworkInterface: "virtio",
		UEFI:             &uefi,
	}

	if err := validateProviderData(value); err != nil {
		t.Fatalf("automatic image mode should be valid: %v", err)
	}
}
