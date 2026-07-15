// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package main runs the VergeOS Omni infrastructure provider.
package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/siderolabs/omni/client/pkg/client"
	"github.com/siderolabs/omni/client/pkg/infra"
	"github.com/spf13/cobra"
	vergeos "github.com/verge-io/govergeos"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/your-org/omni-infra-provider-vergeos/internal/pkg/provider"
	"github.com/your-org/omni-infra-provider-vergeos/internal/pkg/provider/data"
	"github.com/your-org/omni-infra-provider-vergeos/internal/pkg/provider/meta"
)

//go:embed data/icon.svg
var icon []byte

var cfg struct {
	omniAPIEndpoint        string
	serviceAccountKey      string
	providerName           string
	providerDescription    string
	vergeOSEndpoint        string
	vergeOSAPIKey          string
	vergeOSUsername        string
	vergeOSPassword        string
	vergeOSInsecure        bool
	vergeOSTimeout         time.Duration
	imageFactoryBaseURL    string
	omniInsecureSkipVerify bool
}

var rootCmd = &cobra.Command{
	Use:          "omni-infra-provider-vergeos",
	Short:        "VergeOS Omni infrastructure provider",
	Long:         "Connects to Sidero Omni as an infrastructure provider and manages Talos VMs in VergeOS.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		loggerConfig := zap.NewProductionConfig()
		logger, err := loggerConfig.Build(zap.AddStacktrace(zapcore.ErrorLevel))
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		if cfg.omniAPIEndpoint == "" {
			return fmt.Errorf("omni-api-endpoint is required")
		}

		if cfg.vergeOSEndpoint == "" {
			return fmt.Errorf("vergeos-endpoint is required")
		}

		vergeOptions := []vergeos.ClientOption{
			vergeos.WithBaseURL(cfg.vergeOSEndpoint),
			vergeos.WithInsecureTLS(cfg.vergeOSInsecure),
			vergeos.WithTimeout(cfg.vergeOSTimeout),
			vergeos.WithUserAgent("omni-infra-provider-vergeos/alpha"),
		}

		switch {
		case cfg.vergeOSAPIKey != "":
			vergeOptions = append(vergeOptions, vergeos.WithAPIKey(cfg.vergeOSAPIKey))
		case cfg.vergeOSUsername != "" && cfg.vergeOSPassword != "":
			vergeOptions = append(vergeOptions, vergeos.WithCredentials(cfg.vergeOSUsername, cfg.vergeOSPassword))
		default:
			return fmt.Errorf("set VERGEOS_API_KEY or both VERGEOS_USERNAME and VERGEOS_PASSWORD")
		}

		vergeClient, err := vergeos.NewClient(vergeOptions...)
		if err != nil {
			return fmt.Errorf("failed to create VergeOS client: %w", err)
		}

		provisioner := provider.NewProvisioner(
			vergeClient,
			cfg.imageFactoryBaseURL,
		)
		infrastructureProvider, err := infra.NewProvider(
			meta.ProviderID,
			provisioner,
			infra.ProviderConfig{
				Name:        cfg.providerName,
				Description: cfg.providerDescription,
				Icon:        base64.RawStdEncoding.EncodeToString(icon),
				Schema:      string(data.Schema),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create infrastructure provider: %w", err)
		}

		clientOptions := []client.Option{
			client.WithInsecureSkipTLSVerify(cfg.omniInsecureSkipVerify),
		}
		if cfg.serviceAccountKey != "" {
			clientOptions = append(clientOptions, client.WithServiceAccount(cfg.serviceAccountKey))
		}

		logger.Info(
			"starting VergeOS infrastructure provider",
			zap.String("provider_id", meta.ProviderID),
			zap.String("vergeos_endpoint", cfg.vergeOSEndpoint),
			zap.String("image_factory_base_url", cfg.imageFactoryBaseURL),
		)

		return infrastructureProvider.Run(
			cmd.Context(),
			logger,
			infra.WithOmniEndpoint(cfg.omniAPIEndpoint),
			infra.WithClientOptions(clientOptions...),
			infra.WithEncodeRequestIDsIntoTokens(),
		)
	},
}

func main() {
	if err := app(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func app() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer cancel()

	return rootCmd.ExecuteContext(ctx)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func init() {
	rootCmd.Flags().StringVar(
		&cfg.omniAPIEndpoint,
		"omni-api-endpoint",
		os.Getenv("OMNI_ENDPOINT"),
		"Omni API endpoint (defaults to OMNI_ENDPOINT)",
	)
	rootCmd.Flags().StringVar(
		&meta.ProviderID,
		"id",
		meta.ProviderID,
		"provider ID registered in Omni",
	)
	rootCmd.Flags().StringVar(
		&cfg.serviceAccountKey,
		"omni-service-account-key",
		os.Getenv("OMNI_SERVICE_ACCOUNT_KEY"),
		"Omni service account key (defaults to OMNI_SERVICE_ACCOUNT_KEY)",
	)
	rootCmd.Flags().StringVar(&cfg.providerName, "provider-name", "VergeOS", "provider name shown in Omni")
	rootCmd.Flags().StringVar(
		&cfg.providerDescription,
		"provider-description",
		"VergeOS infrastructure provider (alpha)",
		"provider description shown in Omni",
	)
	rootCmd.Flags().StringVar(
		&cfg.vergeOSEndpoint,
		"vergeos-endpoint",
		firstNonEmpty(os.Getenv("VERGEOS_ENDPOINT"), os.Getenv("VERGEOS_HOST")),
		"VergeOS base URL (defaults to VERGEOS_ENDPOINT, then VERGEOS_HOST)",
	)
	rootCmd.Flags().StringVar(
		&cfg.imageFactoryBaseURL,
		"image-factory-base-url",
		firstNonEmpty(os.Getenv("TALOS_IMAGE_FACTORY_BASE_URL"), "https://factory.talos.dev"),
		"Talos Image Factory base URL",
	)
	rootCmd.Flags().StringVar(
		&cfg.vergeOSAPIKey,
		"vergeos-api-key",
		os.Getenv("VERGEOS_API_KEY"),
		"VergeOS API key (defaults to VERGEOS_API_KEY)",
	)
	rootCmd.Flags().StringVar(
		&cfg.vergeOSUsername,
		"vergeos-username",
		os.Getenv("VERGEOS_USERNAME"),
		"VergeOS username when API-key authentication is not used",
	)
	rootCmd.Flags().StringVar(
		&cfg.vergeOSPassword,
		"vergeos-password",
		os.Getenv("VERGEOS_PASSWORD"),
		"VergeOS password when API-key authentication is not used",
	)
	rootCmd.Flags().BoolVar(
		&cfg.vergeOSInsecure,
		"vergeos-insecure-skip-verify",
		false,
		"skip VergeOS TLS certificate verification",
	)
	rootCmd.Flags().DurationVar(&cfg.vergeOSTimeout, "vergeos-timeout", 3*time.Minute, "VergeOS API timeout")
	rootCmd.Flags().BoolVar(
		&cfg.omniInsecureSkipVerify,
		"insecure-skip-verify",
		false,
		"skip Omni TLS certificate verification",
	)
}
