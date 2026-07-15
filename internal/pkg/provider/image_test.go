// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"strings"
	"testing"

	vergeos "github.com/verge-io/govergeos"
)

func TestBuildTalosImageReference(t *testing.T) {
	t.Parallel()

	imageURL, cacheName, err := buildTalosImageReference(
		"https://factory.talos.dev/",
		"schematic123",
		"v1.12.4",
		"amd64",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantURL := "https://factory.talos.dev/image/schematic123/v1.12.4/nocloud-amd64.qcow2"
	if imageURL != wantURL {
		t.Fatalf("got URL %q, want %q", imageURL, wantURL)
	}

	if !strings.HasPrefix(cacheName, imageCachePrefix) || !strings.HasSuffix(cacheName, ".qcow2") {
		t.Fatalf("unexpected cache name %q", cacheName)
	}

	secondURL, secondName, err := buildTalosImageReference(
		"https://factory.talos.dev",
		"schematic123",
		"v1.12.4",
		"amd64",
	)
	if err != nil {
		t.Fatalf("unexpected second error: %v", err)
	}

	if imageURL != secondURL || cacheName != secondName {
		t.Fatal("image reference must be deterministic")
	}
}

func TestBuildTalosImageReferenceRejectsMissingScheme(t *testing.T) {
	t.Parallel()

	_, _, err := buildTalosImageReference(
		"factory.talos.dev",
		"schematic123",
		"v1.12.4",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected invalid base URL error")
	}
}

func TestValidateCachedImage(t *testing.T) {
	t.Parallel()

	file := &vergeos.File{
		ID:       vergeos.FlexInt(10),
		Name:     "omni-talos-test.qcow2",
		Filesize: 1024,
		URL:      "https://factory.talos.dev/image/a/v1/nocloud-amd64.qcow2",
	}

	if err := validateCachedImage(file, file.URL); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if !isVergeFileReady(file) {
		t.Fatal("file with a non-zero size should be ready")
	}

	file.Filesize = 0
	if isVergeFileReady(file) {
		t.Fatal("zero-size file should not be ready")
	}
}

func TestValidateCachedImageRejectsURLCollision(t *testing.T) {
	t.Parallel()

	file := &vergeos.File{
		ID:   vergeos.FlexInt(10),
		Name: "omni-talos-test.qcow2",
		URL:  "https://example.invalid/other.qcow2",
	}

	if err := validateCachedImage(file, "https://factory.talos.dev/image/a/v1/nocloud-amd64.qcow2"); err == nil {
		t.Fatal("expected cache collision error")
	}
}
