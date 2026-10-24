package mistral

import (
	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/providers/openai"
)

const DefaultBaseURL = "https://api.mistral.ai/v1"

// Provider adapts Mistral's OpenAI-compatible API to GoRoute's provider type.
type Provider struct {
	*openai.Provider
}

func NewProvider(id, baseURL, apiKey string, headers map[string]string) *Provider {
	var keys []string
	if apiKey != "" {
		keys = append(keys, apiKey)
	}
	return NewProviderWithKeys(id, baseURL, keys, headers)
}

func NewProviderWithKeys(id, baseURL string, apiKeys []string, headers map[string]string) *Provider {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Provider{Provider: openai.NewProviderWithKeys(id, baseURL, apiKeys, headers)}
}

func (p *Provider) Type() string {
	return "mistral"
}

var _ providers.Provider = (*Provider)(nil)
