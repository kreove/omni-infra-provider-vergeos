// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package data defines VergeOS MachineClass configuration.
package data

import _ "embed"

// Schema is reported to Omni for MachineClass rendering and validation.
//
//go:embed schema.json
var Schema []byte

// Data and schema.json must remain in sync.
type Data struct {
	ClusterID        int    `yaml:"cluster_id"`
	VNetID           int    `yaml:"vnet_id"`
	ImageFileID      int    `yaml:"image_file_id,omitempty"`
	Architecture     string `yaml:"architecture"`
	Cores            int    `yaml:"cores"`
	Memory           uint64 `yaml:"memory"`
	DiskSize         int64  `yaml:"disk_size"`
	PreferredTier    string `yaml:"preferred_tier,omitempty"`
	CPUType          string `yaml:"cpu_type,omitempty"`
	MachineType      string `yaml:"machine_type,omitempty"`
	DiskInterface    string `yaml:"disk_interface,omitempty"`
	NetworkInterface string `yaml:"network_interface,omitempty"`
	UEFI             *bool  `yaml:"uefi,omitempty"`
}
