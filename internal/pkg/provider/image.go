// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/siderolabs/omni/client/pkg/infra/provision"
	vergeos "github.com/verge-io/govergeos"
	"go.uber.org/zap"

	"github.com/kreove/omni-infra-provider-vergeos/internal/pkg/provider/data"
	"github.com/kreove/omni-infra-provider-vergeos/internal/pkg/provider/resources"
)

const imageCachePrefix = "omni-talos-"

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

	imageURL, cacheName, err := buildTalosImageReference(
		p.imageFactoryBaseURL,
		pctx.State.TypedSpec().Value.Schematic,
		pctx.GetTalosVersion(),
		providerData.Architecture,
	)
	if err != nil {
		return 0, false, err
	}

	lockValue, _ := p.imageLocks.LoadOrStore(cacheName, &sync.Mutex{})
	imageLock, ok := lockValue.(*sync.Mutex)
	if !ok {
		return 0, false, fmt.Errorf("invalid internal image lock for %q", cacheName)
	}

	imageLock.Lock()
	defer imageLock.Unlock()

	file, err := p.client.Files.GetByName(ctx, cacheName)
	if err == nil {
		if err = validateCachedImage(file, imageURL); err != nil {
			return 0, false, err
		}

		if !isVergeFileReady(file) {
			logger.Info(
				"waiting for Talos image import",
				zap.String("name", cacheName),
				zap.Int("file_id", file.ID.Int()),
				zap.String("url", imageURL),
			)

			return file.ID.Int(), false, nil
		}

		return file.ID.Int(), true, nil
	}

	if !vergeos.IsNotFoundError(err) {
		return 0, false, fmt.Errorf("failed to inspect VergeOS image cache: %w", err)
	}

	description := fmt.Sprintf(
		"Talos %s %s, schematic %s, managed by Sidero Omni",
		pctx.GetTalosVersion(),
		providerData.Architecture,
		pctx.State.TypedSpec().Value.Schematic,
	)

	file, err = p.client.Files.Create(ctx, &vergeos.FileCreateRequest{
		Name:          cacheName,
		Description:   description,
		PreferredTier: providerData.PreferredTier,
		URL:           imageURL,
	})
	if err != nil {
		// Multiple machine requests can race to create the same cached image.
		// Re-read by deterministic name before treating creation as failed.
		existing, getErr := p.client.Files.GetByName(ctx, cacheName)
		if getErr == nil {
			if validateErr := validateCachedImage(existing, imageURL); validateErr != nil {
				return 0, false, validateErr
			}

			return existing.ID.Int(), isVergeFileReady(existing), nil
		}

		return 0, false, fmt.Errorf(
			"failed to start VergeOS image import from %q: %w",
			imageURL,
			err,
		)
	}

	if err = validateCachedImage(file, imageURL); err != nil {
		return 0, false, err
	}

	logger.Info(
		"started Talos image import",
		zap.String("name", cacheName),
		zap.Int("file_id", file.ID.Int()),
		zap.String("url", imageURL),
	)

	return file.ID.Int(), isVergeFileReady(file), nil
}

func buildTalosImageReference(
	baseURL,
	schematic,
	talosVersion,
	architecture string,
) (imageURL, cacheName string, err error) {
	if strings.TrimSpace(schematic) == "" {
		return "", "", fmt.Errorf("cannot build Talos image URL without a schematic ID")
	}

	if strings.TrimSpace(talosVersion) == "" {
		return "", "", fmt.Errorf("cannot build Talos image URL without a Talos version")
	}

	if strings.TrimSpace(architecture) == "" {
		return "", "", fmt.Errorf("cannot build Talos image URL without an architecture")
	}

	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", "", fmt.Errorf("invalid Image Factory URL %q: %w", baseURL, err)
	}

	if base.Scheme != "https" && base.Scheme != "http" {
		return "", "", fmt.Errorf("Image Factory URL must use HTTP or HTTPS")
	}

	if base.Host == "" {
		return "", "", fmt.Errorf("Image Factory URL %q has no host", baseURL)
	}

	base.Path = path.Join(
		base.Path,
		"image",
		schematic,
		talosVersion,
		fmt.Sprintf("nocloud-%s.qcow2", architecture),
	)
	base.RawPath = ""
	base.RawQuery = ""
	base.Fragment = ""

	imageURL = base.String()
	hash := sha256.Sum256([]byte(imageURL))
	cacheName = imageCachePrefix + hex.EncodeToString(hash[:12]) + ".qcow2"

	return imageURL, cacheName, nil
}

func validateCachedImage(file *vergeos.File, expectedURL string) error {
	if file == nil {
		return fmt.Errorf("VergeOS returned an empty image file response")
	}

	if file.ID.Int() < 1 {
		return fmt.Errorf("VergeOS image file %q returned an invalid ID", file.Name)
	}

	if file.URL != "" && file.URL != expectedURL {
		return fmt.Errorf(
			"VergeOS image cache collision: file %q points to %q instead of %q",
			file.Name,
			file.URL,
			expectedURL,
		)
	}

	return nil
}

func isVergeFileReady(file *vergeos.File) bool {
	return file != nil && file.ID.Int() > 0 && file.Filesize > 0
}
