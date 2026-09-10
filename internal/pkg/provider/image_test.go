// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"strings"
	"testing"

	"github.com/siderolabs/omni/client/pkg/imagefactory"
	vergeos "github.com/verge-io/govergeos"

	"github.com/kreove/omni-infra-provider-vergeos/internal/pkg/provider/data"
)

func TestMediaSpecFor(t *testing.T) {
	t.Parallel()

	spec := mediaSpecFor(data.Data{Architecture: "amd64"})

	if spec.Kind != imagefactory.InstallationMediaKindDisk {
		t.Errorf("Kind = %q, want a disk image", spec.Kind)
	}

	// NoCloud is what makes Talos read the cloud-init files attached to the
	// VM; any other platform would ignore them.
	if spec.Platform != talosPlatform {
		t.Errorf("Platform = %q, want %q", spec.Platform, talosPlatform)
	}

	// VergeOS downloads the file as-is and cannot decompress on the way in, so
	// the uncompressed qcow2 is the only workable artifact here.
	if spec.Format != talosDiskFormat {
		t.Errorf("Format = %q, want %q", spec.Format, talosDiskFormat)
	}

	if spec.Architecture != "amd64" {
		t.Errorf("Architecture = %q, want amd64", spec.Architecture)
	}
}

// VergeOS is handed a bare URL and fetches it itself, with nowhere to put a
// request header. Asking for a standalone URL is what keeps authentication
// inside the URL where VergeOS can actually use it.
func TestMediaSpecForRequestsAStandaloneURL(t *testing.T) {
	t.Parallel()

	if !mediaSpecFor(data.Data{Architecture: "amd64"}).StandaloneURL {
		t.Error("StandaloneURL must be set: VergeOS cannot send request headers")
	}
}

// The URL is handed off rather than fetched, so it has to outlive the handoff
// by however long VergeOS takes to import. Omni's default assumes an immediate
// fetch, which is never true for this provider.
func TestMediaSpecForRequestsALongDownloadToken(t *testing.T) {
	t.Parallel()

	spec := mediaSpecFor(data.Data{Architecture: "amd64"})

	if spec.DownloadTokenTTL < imageDownloadTokenTTL {
		t.Errorf("DownloadTokenTTL = %s, want at least %s", spec.DownloadTokenTTL, imageDownloadTokenTTL)
	}
}

// The spec must satisfy the image factory's own rules, or the medium is
// rejected at resolve time rather than here.
func TestMediaSpecForIsValid(t *testing.T) {
	t.Parallel()

	if err := mediaSpecFor(data.Data{Architecture: "amd64"}).Validate(); err != nil {
		t.Errorf("media spec is invalid: %v", err)
	}

	if err := mediaSpecFor(data.Data{}).Validate(); err == nil {
		t.Error("expected an empty architecture to fail media spec validation")
	}
}

// The name must come from StorageKey, never the URL: the URL can carry
// credentials or a download token, so a name derived from it would change when
// those rotate and orphan the file already imported under the old name.
func TestCacheNameForUsesStorageKey(t *testing.T) {
	t.Parallel()

	got := cacheNameFor(imagefactory.InstallationMedia{
		StorageKey:  "abc123",
		URL:         "https://factory.talos.dev/image/x/y/z?token=secret",
		SchematicID: "schematic123",
	})

	if got != imageCachePrefix+"abc123."+talosDiskFormat {
		t.Fatalf("cacheNameFor() = %q", got)
	}

	if strings.Contains(got, "secret") || strings.Contains(got, "token") {
		t.Errorf("cache name %q leaks the download URL", got)
	}

	// VergeOS keys file handling off the extension, so it has to survive.
	if !strings.HasSuffix(got, "."+talosDiskFormat) {
		t.Errorf("cache name %q lost the %q extension", got, talosDiskFormat)
	}
}

func TestCacheNameForIsStableAndDistinct(t *testing.T) {
	t.Parallel()

	first := cacheNameFor(imagefactory.InstallationMedia{StorageKey: "aaa"})

	if first != cacheNameFor(imagefactory.InstallationMedia{StorageKey: "aaa"}) {
		t.Error("cache name is not stable across calls")
	}

	if first == cacheNameFor(imagefactory.InstallationMedia{StorageKey: "bbb"}) {
		t.Error("distinct media must not share a cache name")
	}
}

func TestDescribeImage(t *testing.T) {
	t.Parallel()

	got := describeImage(
		imagefactory.InstallationMedia{
			SchematicID: "schematic123",
			URL:         "https://factory.talos.dev/image?token=secret",
		},
		"v1.12.4",
		data.Data{Architecture: "amd64"},
	)

	for _, want := range []string{"v1.12.4", "amd64", "schematic123"} {
		if !strings.Contains(got, want) {
			t.Errorf("description %q is missing %q", got, want)
		}
	}

	// VergeOS shows the description to anyone who can list files, and the URL
	// can carry credentials.
	if strings.Contains(got, "secret") || strings.Contains(got, "http") {
		t.Errorf("description %q leaks the download URL", got)
	}
}

func TestValidateCachedImage(t *testing.T) {
	t.Parallel()

	if err := validateCachedImage(nil); err == nil {
		t.Error("expected an error for a nil file")
	}

	if err := validateCachedImage(&vergeos.File{Name: "x"}); err == nil {
		t.Error("expected an error for a file with no ID")
	}

	if err := validateCachedImage(&vergeos.File{ID: 7, Name: "x"}); err != nil {
		t.Errorf("unexpected error for a valid file: %v", err)
	}
}

// A cached file's stored URL and a freshly issued one differ for the same image
// whenever the factory authenticates downloads with a token. Comparing them
// would report a bogus cache collision and fail every provision after the first
// rotation, so identity comes from the name alone.
func TestValidateCachedImageIgnoresTheSourceURL(t *testing.T) {
	t.Parallel()

	file := &vergeos.File{
		ID:   7,
		Name: "omni-talos-abc123.qcow2",
		URL:  "https://factory.talos.dev/image/x/y/z?token=an-older-token",
	}

	if err := validateCachedImage(file); err != nil {
		t.Errorf("a rotated source URL must not invalidate the cache: %v", err)
	}
}

func TestIsVergeFileReady(t *testing.T) {
	t.Parallel()

	if isVergeFileReady(nil) {
		t.Error("nil file must not be ready")
	}

	// An import in flight has a record but no bytes yet.
	if isVergeFileReady(&vergeos.File{ID: 7}) {
		t.Error("a file with no size must not be ready")
	}

	if !isVergeFileReady(&vergeos.File{ID: 7, Filesize: 1024}) {
		t.Error("a sized file must be ready")
	}
}
