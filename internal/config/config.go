// @fileoverview アプリ設定の読み込みとデフォルト値を定義する。
package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config はアプリ全体の設定を保持する。
type Config struct {
	AppDataDir          string
	DatabasePath        string
	LogLevel            string
	S3Endpoint          string
	S3Region            string
	S3Bucket            string
	S3ForcePathStyle    bool
	S3UseTLS            bool
	CloudMetadataKey    string
	CloudImagesPrefix   string
	CredentialNamespace string
}

// LoadFromEnv は環境変数から設定を読み込む。
func LoadFromEnv() Config {
	appDataDir := getEnv("CLOUDLAUNCH_APPDATA", defaultAppDataDir())
	databasePath := getEnv("CLOUDLAUNCH_DB_PATH", filepath.Join(appDataDir, "app.db"))

	return Config{
		AppDataDir:          appDataDir,
		DatabasePath:        databasePath,
		LogLevel:            getEnv("CLOUDLAUNCH_LOG_LEVEL", "info"),
		S3Endpoint:          getEnv("CLOUDLAUNCH_S3_ENDPOINT", ""),
		S3Region:            getEnv("CLOUDLAUNCH_S3_REGION", "auto"),
		S3Bucket:            getEnv("CLOUDLAUNCH_S3_BUCKET", ""),
		S3ForcePathStyle:    getEnvBool("CLOUDLAUNCH_S3_FORCE_PATH_STYLE", false),
		S3UseTLS:            getEnvBool("CLOUDLAUNCH_S3_USE_TLS", true),
		CloudMetadataKey:    getEnv("CLOUDLAUNCH_CLOUD_METADATA_KEY", "games.json"),
		CloudImagesPrefix:   getEnv("CLOUDLAUNCH_CLOUD_IMAGES_PREFIX", "images/"),
		CredentialNamespace: getEnv("CLOUDLAUNCH_CREDENTIAL_NAMESPACE", "CloudLaunch"),
	}
}

func defaultAppDataDir() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "CloudLaunch")
}

// getEnv は環境変数を読み取り、空の場合はfallbackを返す。
func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// getEnvBool は環境変数を真偽値として読み取る。
func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}
