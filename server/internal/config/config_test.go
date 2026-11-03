package config

import (
	"os"
	"testing"
)

func TestDefaultsAndOverrides(t *testing.T) {
	os.Setenv("WEPUB_TOKEN", "t")
	defer os.Unsetenv("WEPUB_TOKEN")
	c, err := FromEnv()
	if err != nil || c.Addr != ":8080" || c.MaxUploadMB != 32 {
		t.Fatalf("defaults wrong: %+v %v", c, err)
	}
	os.Setenv("WEPUB_ADDR", ":9090")
	os.Setenv("WEPUB_MAX_UPLOAD_MB", "10")
	c, _ = FromEnv()
	if c.Addr != ":9090" || c.MaxUploadMB != 10 {
		t.Fatalf("override wrong: %+v", c)
	}
	os.Unsetenv("WEPUB_ADDR")
	os.Unsetenv("WEPUB_MAX_UPLOAD_MB")
}

func TestMissingToken(t *testing.T) {
	os.Unsetenv("WEPUB_TOKEN")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error")
	}
}
