// Copyright 2024 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Package main implements a certificate creation utility for Timestamp Authority.
// It supports creating root and leaf certificates using (AWS, GCP, Azure).
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sigstore/timestamp-authority/pkg/certmaker"
	"github.com/sigstore/timestamp-authority/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// CLI flags and env vars for config.
// Supports AWS KMS, Google Cloud KMS, and Azure Key Vault configurations.
var (
	version string

	rootCmd = &cobra.Command{
		Use:     "tsa-certificate-maker",
		Short:   "Create certificate chains for Timestamp Authority",
		Long:    `A tool for creating root, intermediate, and leaf certificates for Timestamp Authority with timestamping capabilities`,
		Version: version,
	}

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create certificate chain",
		RunE:  runCreate,
	}
)

func mustBindPFlag(key string, flag *pflag.Flag) {
	if err := viper.BindPFlag(key, flag); err != nil {
		log.Logger.Fatal("failed to bind flag", zap.String("flag", key), zap.Error(err))
	}
}

func mustBindEnv(key, envVar string) {
	if err := viper.BindEnv(key, envVar); err != nil {
		log.Logger.Fatal("failed to bind env var", zap.String("var", envVar), zap.Error(err))
	}
}

func init() {
	log.ConfigureLogger("prod")

	rootCmd.AddCommand(createCmd)
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// KMS provider flags
	createCmd.Flags().String("kms-type", "", "KMS provider type (awskms, gcpkms, azurekms, hashivault)")
	createCmd.Flags().String("aws-region", "", "AWS KMS region")
	createCmd.Flags().String("azure-tenant-id", "", "Azure KMS tenant ID")
	createCmd.Flags().String("gcp-credentials-file", "", "Path to credentials file for GCP KMS")
	createCmd.Flags().String("vault-token", "", "HashiVault token")
	createCmd.Flags().String("vault-address", "", "HashiVault server address")

	// Root certificate flags
	createCmd.Flags().String("root-template", "pkg/certmaker/templates/root-template.json", "Path to root certificate template")
	createCmd.Flags().String("root-key-id", "", "KMS key identifier for root certificate")
	createCmd.Flags().String("root-cert", "root.pem", "Output path for root certificate")

	// Intermediate certificate flags
	createCmd.Flags().String("intermediate-template", "pkg/certmaker/templates/intermediate-template.json", "Path to intermediate certificate template")
	createCmd.Flags().String("intermediate-key-id", "", "KMS key identifier for intermediate certificate")
	createCmd.Flags().String("intermediate-cert", "intermediate.pem", "Output path for intermediate certificate")

	// Leaf certificate flags
	createCmd.Flags().String("leaf-template", "pkg/certmaker/templates/leaf-template.json", "Path to leaf certificate template")
	createCmd.Flags().String("leaf-key-id", "", "KMS key identifier for leaf certificate")
	createCmd.Flags().String("leaf-cert", "leaf.pem", "Output path for leaf certificate")

	// Bind flags to viper
	mustBindPFlag("kms-type", createCmd.Flags().Lookup("kms-type"))
	mustBindPFlag("aws-region", createCmd.Flags().Lookup("aws-region"))
	mustBindPFlag("azure-tenant-id", createCmd.Flags().Lookup("azure-tenant-id"))
	mustBindPFlag("gcp-credentials-file", createCmd.Flags().Lookup("gcp-credentials-file"))
	mustBindPFlag("vault-token", createCmd.Flags().Lookup("vault-token"))
	mustBindPFlag("vault-address", createCmd.Flags().Lookup("vault-address"))

	mustBindPFlag("root-template", createCmd.Flags().Lookup("root-template"))
	mustBindPFlag("root-key-id", createCmd.Flags().Lookup("root-key-id"))
	mustBindPFlag("root-cert", createCmd.Flags().Lookup("root-cert"))

	mustBindPFlag("intermediate-template", createCmd.Flags().Lookup("intermediate-template"))
	mustBindPFlag("intermediate-key-id", createCmd.Flags().Lookup("intermediate-key-id"))
	mustBindPFlag("intermediate-cert", createCmd.Flags().Lookup("intermediate-cert"))

	mustBindPFlag("leaf-template", createCmd.Flags().Lookup("leaf-template"))
	mustBindPFlag("leaf-key-id", createCmd.Flags().Lookup("leaf-key-id"))
	mustBindPFlag("leaf-cert", createCmd.Flags().Lookup("leaf-cert"))

	// Bind environment variables
	mustBindEnv("kms-type", "KMS_TYPE")
	mustBindEnv("aws-region", "AWS_REGION")
	mustBindEnv("azure-tenant-id", "AZURE_TENANT_ID")
	mustBindEnv("gcp-credentials-file", "GCP_CREDENTIALS_FILE")
	mustBindEnv("vault-token", "VAULT_TOKEN")
	mustBindEnv("vault-address", "VAULT_ADDR")

	mustBindEnv("root-key-id", "KMS_ROOT_KEY_ID")
	mustBindEnv("intermediate-key-id", "KMS_INTERMEDIATE_KEY_ID")
	mustBindEnv("leaf-key-id", "KMS_LEAF_KEY_ID")
}

func runCreate(_ *cobra.Command, _ []string) error {
	defer func() { rootCmd.SilenceUsage = true }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build KMS config from flags and environment
	config := certmaker.KMSConfig{
		Type:              viper.GetString("kms-type"),
		RootKeyID:         viper.GetString("root-key-id"),
		IntermediateKeyID: viper.GetString("intermediate-key-id"),
		LeafKeyID:         viper.GetString("leaf-key-id"),
		Options:           make(map[string]string),
	}

	// Handle KMS provider options
	switch config.Type {
	case "awskms":
		if awsRegion := viper.GetString("aws-region"); awsRegion != "" {
			config.Options["aws-region"] = awsRegion
		}
	case "gcpkms":
		if gcpCredsFile := viper.GetString("gcp-credentials-file"); gcpCredsFile != "" {
			// Check if credentials file exists before trying to use it
			if _, err := os.Stat(gcpCredsFile); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("failed to initialize KMS: credentials file not found: %s", gcpCredsFile)
				}
				return fmt.Errorf("failed to initialize KMS: error accessing credentials file: %w", err)
			}
			config.Options["gcp-credentials-file"] = gcpCredsFile
		}
	case "azurekms":
		if azureTenantID := viper.GetString("azure-tenant-id"); azureTenantID != "" {
			config.Options["azure-tenant-id"] = azureTenantID
		}
	case "hashivault":
		if vaultToken := viper.GetString("vault-token"); vaultToken != "" {
			config.Options["vault-token"] = vaultToken
		}
		if vaultAddr := viper.GetString("vault-address"); vaultAddr != "" {
			config.Options["vault-address"] = vaultAddr
		}
	}

	km, err := certmaker.InitKMS(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize KMS: %w", err)
	}

	// Validate template paths
	if err := certmaker.ValidateTemplatePath(viper.GetString("root-template")); err != nil {
		return fmt.Errorf("root template error: %w", err)
	}
	if err := certmaker.ValidateTemplatePath(viper.GetString("leaf-template")); err != nil {
		return fmt.Errorf("leaf template error: %w", err)
	}
	if viper.GetString("intermediate-key-id") != "" {
		if err := certmaker.ValidateTemplatePath(viper.GetString("intermediate-template")); err != nil {
			return fmt.Errorf("intermediate template error: %w", err)
		}
	}

	return certmaker.CreateCertificates(km, config, viper.GetString("root-template"), viper.GetString("leaf-template"), viper.GetString("root-cert"), viper.GetString("leaf-cert"), viper.GetString("intermediate-key-id"), viper.GetString("intermediate-template"), viper.GetString("intermediate-cert"))
}

func main() {
	rootCmd.SilenceErrors = true
	if err := rootCmd.Execute(); err != nil {
		if rootCmd.SilenceUsage {
			log.Logger.Fatal("Command failed", zap.Error(err))
		} else {
			os.Exit(1)
		}
	}
}
