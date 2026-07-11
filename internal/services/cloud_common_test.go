package services

import (
	"testing"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/infrastructure/credentials"
)

func TestNormalizeImageExt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		ext         string
		contentType string
		want        string
	}{
		{"明示拡張子（ドット付き）", ".webp", "image/png", ".webp"},
		{"明示拡張子（ドット無し）", "JPG", "image/png", ".jpg"},
		{"contentType png", "", "image/png", ".png"},
		{"contentType gif", "", "image/gif", ".gif"},
		{"contentType jpeg", "", "image/jpeg", ".jpg"},
		{"contentType jpg 表記", "", "image/jpg", ".jpg"},
		{"contentType webp は .webp（旧版は .png に丸めて破損していた）", "", "image/webp", ".webp"},
		{"contentType bmp は .bmp", "", "image/bmp", ".bmp"},
		{"contentType avif は .avif", "", "image/avif", ".avif"},
		{"大文字 contentType も拾う", "", "IMAGE/WEBP", ".webp"},
		{"未知 contentType は .png にフォールバック", "", "image/svg+xml", ".png"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeImageExt(c.ext, c.contentType); got != c.want {
				t.Fatalf("normalizeImageExt(%q, %q) = %q, want %q", c.ext, c.contentType, got, c.want)
			}
		})
	}
}

func TestResolveS3ConfigPropagatesForcePathStyle(t *testing.T) {
	t.Parallel()

	cfg := resolveS3Config(config.Config{
		S3Endpoint:       "https://example.invalid",
		S3Region:         "auto",
		S3Bucket:         "fallback-bucket",
		S3ForcePathStyle: true,
		S3UseTLS:         true,
	}, &credentials.Credential{
		Endpoint:   "https://minio.local",
		Region:     "us-east-1",
		BucketName: "games",
	})

	if !cfg.ForcePathStyle {
		t.Fatal("expected ForcePathStyle to follow app config (MinIO path-style)")
	}
	if cfg.Endpoint != "https://minio.local" || cfg.Bucket != "games" || cfg.Region != "us-east-1" {
		t.Fatalf("unexpected s3 config: %+v", cfg)
	}
	if !cfg.UseTLS {
		t.Fatal("expected UseTLS from base config")
	}
}
