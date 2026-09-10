// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/siderolabs/omni/client/pkg/imagefactory"
	"github.com/siderolabs/omni/client/pkg/infra/provision"
	vergeos "github.com/verge-io/govergeos"
	"go.uber.org/zap"

	"github.com/kreove/omni-infra-provider-vergeos/internal/pkg/provider/data"
	"github.com/kreove/omni-infra-provider-vergeos/internal/pkg/provider/resources"
)

const (
	imageCachePrefix = "omni-talos-"

	// talosPlatform is the Talos platform this provider provisions. NoCloud is
	// what makes the guest read the cloud-init files attached to the VM.
	talosPlatform = "nocloud"

	// talosDiskFormat is the disk image VergeOS imports. qcow2 is served
	// uncompressed, which matters here because VergeOS fetches it directly and
	// cannot decompress on the way in.
	talosDiskFormat = "qcow2"

	// imageDownloadTokenTTL is how long the provider asks the image factory
	// download URL to stay valid.
	//
	// This provider is unusual: it never fetches the image itself. It hands
	// the URL to VergeOS, which downloads on its own schedule, so the URL has
	// to outlive the handoff by however long that import takes. Omni's default
	// assumes the caller fetches immediately, which is never true here.
	imageDownloadTokenTTL = 2 * time.Hour
)

// ensureTalosImage resolves a manually supplied VergeOS file or creates and
// reuses a cached Talos image imported directly by VergeOS from Image Factory.
// The returned boolean is false while an asynchronous VergeOS import is still
// in progress.
func (p *Provisioner) ensureTalosImage(
	ctx context.Context,
	logger *zap.Logger,
	pctx provision.Context[*resources.Machine],
	providerData data.Data,
) (int, bool, error) {
	if providerData.ImageFileID > 0 {
		file, err := p.client.Files.Get(ctx, providerData.ImageFileID)
		if err != nil {
			return 0, false, fmt.Errorf(
				"failed to resolve VergeOS image file %d: %w",
				providerData.ImageFileID,
				err,
			)
		}

		if file.ID.Int() < 1 {
			return 0, false, fmt.Errorf(
				"VergeOS image file %d returned an invalid ID",
				providerData.ImageFileID,
			)
		}

		return file.ID.Int(), true, nil
	}

	// Omni owns the schematic upload and knows how its image factory spells a
	// medium, so the provider asks for one by description rather than building
	// a factory URL itself.
	media, err := pctx.EnsureInstallationMedia(
		ctx,
		logger,
		mediaSpecFor(providerData),
		provision.WithExtraKernelArgs("console=ttyS0,38400n8"),
		provision.WithoutConnectionParams(),
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to resolve Talos installation media: %w", err)
	}

	// VergeOS performs the download itself and has no way to send request
	// headers with it. MediaSpec.StandaloneURL asks Omni for a URL that needs
	// none, but that is a request rather than a guarantee, so a factory
	// configured in a way that still requires them has to be reported. Passing
	// the URL on regardless would fail inside VergeOS as an opaque download
	// error with nothing pointing back to the cause.
	if len(media.Headers) > 0 {
		return 0, false, fmt.Errorf(
			"the configured image factory requires request headers to download images, " +
				"which VergeOS cannot send because it fetches the image itself; " +
				"use an image factory that authenticates in the URL, or one that needs no authentication",
		)
	}

	pctx.State.TypedSpec().Value.Schematic = media.SchematicID
	pctx.State.TypedSpec().Value.TalosVersion = pctx.GetTalosVersion()

	cacheName := cacheNameFor(media)

	lockValue, _ := p.imageLocks.LoadOrStore(cacheName, &sync.Mutex{})

	imageLock, ok := lockValue.(*sync.Mutex)
	if !ok {
		return 0, false, fmt.Errorf("invalid internal image lock for %q", cacheName)
	}

	imageLock.Lock()
	defer imageLock.Unlock()

	file, err := p.client.Files.GetByName(ctx, cacheName)
	if err == nil {
		if err = validateCachedImage(file); err != nil {
			return 0, false, err
		}

		if !isVergeFileReady(file) {
			// The URL is deliberately absent from this line: it can carry
			// credentials or a download token.
			logger.Info(
				"waiting for Talos image import",
				zap.String("name", cacheName),
				zap.Int("file_id", file.ID.Int()),
				zap.String("schematic", media.SchematicID),
			)

			return file.ID.Int(), false, nil
		}

		return file.ID.Int(), true, nil
	}

	if !vergeos.IsNotFoundError(err) {
		return 0, false, fmt.Errorf("failed to inspect VergeOS image cache: %w", err)
	}

	file, err = p.client.Files.Create(ctx, &vergeos.FileCreateRequest{
		Name:          cacheName,
		Description:   describeImage(media, pctx.GetTalosVersion(), providerData),
		PreferredTier: providerData.PreferredTier,
		URL:           media.URL,
	})
	if err != nil {
		// Multiple machine requests can race to create the same cached image.
		// Re-read by deterministic name before treating creation as failed.
		existing, getErr := p.client.Files.GetByName(ctx, cacheName)
		if getErr == nil {
			if validateErr := validateCachedImage(existing); validateErr != nil {
				return 0, false, validateErr
			}

			return existing.ID.Int(), isVergeFileReady(existing), nil
		}

		return 0, false, fmt.Errorf("failed to start VergeOS image import for %q: %w", cacheName, err)
	}

	if err = validateCachedImage(file); err != nil {
		return 0, false, err
	}

	logger.Info(
		"started Talos image import",
		zap.String("name", cacheName),
		zap.Int("file_id", file.ID.Int()),
		zap.String("schematic", media.SchematicID),
		zap.String("talos_version", pctx.GetTalosVersion()),
	)

	return file.ID.Int(), isVergeFileReady(file), nil
}

// mediaSpecFor describes the installation medium a Machine Class asks for.
func mediaSpecFor(providerData data.Data) provision.MediaSpec {
	return provision.MediaSpec{
		MediaSpec: imagefactory.MediaSpec{
			Kind:         imagefactory.InstallationMediaKindDisk,
			Platform:     talosPlatform,
			Architecture: providerData.Architecture,
			Format:       talosDiskFormat,
		},
		DownloadTokenTTL: imageDownloadTokenTTL,
		// VergeOS is handed the bare URL and downloads it itself, so it must
		// carry any authentication inside the URL: VergeOS has nowhere to put
		// a header.
		StandaloneURL: true,
	}
}

// cacheNameFor derives the VergeOS file name for a medium.
//
// StorageKey exists for exactly this: it identifies the medium and changes only
// when the medium does. The URL must not be used instead -- it can carry
// credentials or a download token, so a name derived from it would change
// whenever those rotate and orphan the file already imported under the old
// name.
func cacheNameFor(media imagefactory.InstallationMedia) string {
	return imageCachePrefix + media.StorageKey + "." + talosDiskFormat
}

// describeImage renders the description stored on a cached image.
//
// The name is a digest and says nothing to a human. The source URL is
// deliberately not recorded: it can carry credentials, and VergeOS stores the
// description in plain sight of anyone who can list files.
func describeImage(media imagefactory.InstallationMedia, talosVersion string, providerData data.Data) string {
	return fmt.Sprintf(
		"Talos %s %s, schematic %s, managed by Sidero Omni",
		talosVersion,
		providerData.Architecture,
		media.SchematicID,
	)
}

// validateCachedImage checks that a cached file record is usable.
//
// It deliberately does not compare the file's source URL against the one just
// issued. Those can differ for the same image -- an image factory that
// authenticates downloads puts a token in the URL, and a later reconcile is
// issued a fresh one -- so comparing them would report a bogus cache collision
// and fail every provision after the first token rotation. The cache name is
// derived from the medium's storage key, which identifies the image on its own.
func validateCachedImage(file *vergeos.File) error {
	if file == nil {
		return fmt.Errorf("VergeOS returned an empty image file response")
	}

	if file.ID.Int() < 1 {
		return fmt.Errorf("VergeOS image file %q returned an invalid ID", file.Name)
	}

	return nil
}

func isVergeFileReady(file *vergeos.File) bool {
	return file != nil && file.ID.Int() > 0 && file.Filesize > 0
}
