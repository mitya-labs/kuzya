package main

import (
	"os"
	"strings"
	"testing"
)

func TestMaskKey(t *testing.T) {
	if got := maskKey(""); got != "(не задан)" {
		t.Errorf("empty=%q", got)
	}
	if got := maskKey("sk-1234567890"); got != "sk-… (13 симв.)" {
		t.Errorf("mask=%q", got)
	}
	if got := maskKey("short"); got != "****" {
		t.Errorf("short=%q", got)
	}
}

func TestSaveLoadRemoteConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KUZYA_HOME", dir)
	for _, k := range []string{"KUZYA_REMOTE_URL", "KUZYA_REMOTE_KEY", "KUZYA_REMOTE_MODEL"} {
		os.Unsetenv(k)
	}
	if err := saveRemoteConfig("https://example.com/v1", "secret-key-123", "m1"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(remoteEnvFile())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("права %o", info.Mode().Perm())
	}
	os.Unsetenv("KUZYA_REMOTE_URL")
	os.Unsetenv("KUZYA_REMOTE_KEY")
	os.Unsetenv("KUZYA_REMOTE_MODEL")
	loadRemoteConfig()
	if os.Getenv("KUZYA_REMOTE_URL") != "https://example.com/v1" || os.Getenv("KUZYA_REMOTE_MODEL") != "m1" {
		t.Fatalf("не загрузилось: %v", os.Environ())
	}
	if !strings.Contains(remoteStatus(), "настроена") {
		t.Errorf("status=%q", remoteStatus())
	}
}
