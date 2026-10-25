package main

import (
	"testing"

	"github.com/ziuus/goroute/pkg/catalog"
	"github.com/ziuus/goroute/pkg/config"
)

func TestAPIKeyProviderUsesConfiguredProviderType(t *testing.T) {
	for _, test := range []struct {
		providerType string
		wantType     string
	}{
		{providerType: "mistral", wantType: "mistral"},
		{providerType: "openai", wantType: "openai"},
	} {
		t.Run(test.providerType, func(t *testing.T) {
			provider := newAPIKeyProvider(config.ProviderConfig{ID: "test", Type: test.providerType})
			if got := provider.Type(); got != test.wantType {
				t.Fatalf("provider Type() = %q, want %q", got, test.wantType)
			}
		})
	}
}

func TestMistralCatalogSelectsNativeProvider(t *testing.T) {
	entry := catalog.FindEntry("mistral")
	if entry == nil {
		t.Fatal("Mistral catalog entry not found")
	}
	if entry.Type != "mistral" {
		t.Fatalf("catalog type = %q, want %q", entry.Type, "mistral")
	}
	if entry.BaseURL != "https://api.mistral.ai/v1" {
		t.Fatalf("catalog base URL = %q, want %q", entry.BaseURL, "https://api.mistral.ai/v1")
	}
}
