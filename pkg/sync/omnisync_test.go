package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecryptOmniCredential(t *testing.T) {
	secret := "bf80ce3f2197cba0d3c04719dcae0338eb9d5d7e684b58226ec19c16523ea0cd"
	plain := "sk-direct-test-key"
	dec, err := DecryptOmniCredential(plain, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec != plain {
		t.Fatalf("expected %s, got %s", plain, dec)
	}
}

func TestSyncLive(t *testing.T) {
	home, _ := os.UserHomeDir()
	cfgPath := filepath.Join(home, ".goroute", "config.yaml")
	res, err := SyncFromOmniRoute(cfgPath)
	if err != nil {
		t.Fatalf("SyncFromOmniRoute failed: %v", err)
	}
	t.Logf("Sync result: providers=%d, accounts=%d, combos=%d", res.ProvidersImported, res.AccountsImported, res.CombosImported)
	for _, d := range res.Details {
		t.Logf("  detail: %s", d)
	}
}
