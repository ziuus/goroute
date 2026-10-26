package guardrails

import (
	"regexp"
	"strings"
)

// SensitiveRule defines a pattern to detect and redact sensitive data.
type SensitiveRule struct {
	Name        string
	Regex       *regexp.Regexp
	Replacement string
	Severity    string
}

// Built-in credential & secret patterns (OmniRoute parity: credentialPatterns.ts + piiMasker.ts)
var (
	CredentialRules = []SensitiveRule{
		{Name: "openai_proj", Regex: regexp.MustCompile(`sk-proj-[A-Za-z0-9_-]{20,}`), Replacement: "[REDACTED:openai]", Severity: "high"},
		{Name: "openai", Regex: regexp.MustCompile(`\bsk-[A-Za-z0-9]{48}\b`), Replacement: "[REDACTED:openai]", Severity: "high"},
		{Name: "anthropic", Regex: regexp.MustCompile(`sk-ant-api[0-9]*-[A-Za-z0-9_-]{20,}`), Replacement: "[REDACTED:anthropic]", Severity: "high"},
		{Name: "anthropic_alt", Regex: regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`), Replacement: "[REDACTED:anthropic]", Severity: "high"},
		{Name: "groq", Regex: regexp.MustCompile(`\bgsk_[A-Za-z0-9]{20,}`), Replacement: "[REDACTED:groq]", Severity: "high"},
		{Name: "xai", Regex: regexp.MustCompile(`\bxai-[A-Za-z0-9]{20,}`), Replacement: "[REDACTED:xai]", Severity: "high"},
		{Name: "google", Regex: regexp.MustCompile(`AIza[0-9A-Za-z_-]{20,}`), Replacement: "[REDACTED:google]", Severity: "high"},
		{Name: "github", Regex: regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`), Replacement: "[REDACTED:github]", Severity: "high"},
		{Name: "slack", Regex: regexp.MustCompile(`xox[bpoa]-[A-Za-z0-9-]{10,}`), Replacement: "[REDACTED:slack]", Severity: "high"},
		{Name: "huggingface", Regex: regexp.MustCompile(`hf_[A-Za-z0-9]{34}`), Replacement: "[REDACTED:hf]", Severity: "high"},
		{Name: "aws_access_key", Regex: regexp.MustCompile(`AKIA[0-9A-Z]{16}`), Replacement: "[REDACTED:aws]", Severity: "high"},
		{Name: "generic_api_key", Regex: regexp.MustCompile(`(?i)(?:sk|pk|api|key|token)[_-][a-zA-Z0-9]{20,}`), Replacement: "[REDACTED:api_key]", Severity: "high"},
	}

	PIIRules = []SensitiveRule{
		{Name: "email", Regex: regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`), Replacement: "[EMAIL_REDACTED]", Severity: "medium"},
		{Name: "ssn", Regex: regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), Replacement: "[SSN_REDACTED]", Severity: "high"},
		{Name: "credit_card", Regex: regexp.MustCompile(`\b(?:\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}|\d{4}[-\s]?\d{6}[-\s]?\d{4,5})\b`), Replacement: "[CC_REDACTED]", Severity: "high"},
		{Name: "phone_us", Regex: regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`), Replacement: "[PHONE_REDACTED]", Severity: "medium"},
	}

	// Prompt injection and jailbreak escape patterns (OmniRoute parity: promptInjection.ts)
	InjectionRules = []SensitiveRule{
		{Name: "system_override_inline", Regex: regexp.MustCompile(`(?i)\bsystem\s*:\s*override\b`), Severity: "high"},
		{Name: "markdown_system_block", Regex: regexp.MustCompile("(?i)`{3,}\\s*system\\b"), Severity: "high"},
		{Name: "ignore_previous", Regex: regexp.MustCompile(`(?i)\bignore\s+(all\s+)?previous\s+instructions\b`), Severity: "high"},
		{Name: "disregard_instructions", Regex: regexp.MustCompile(`(?i)\bdisregard\s+(all\s+)?prior\s+instructions\b`), Severity: "high"},
		{Name: "developer_mode", Regex: regexp.MustCompile(`(?i)\benable\s+developer\s+mode\b`), Severity: "high"},
	}
)

// DetectPromptInjection checks if text contains adversarial system prompt overrides.
func DetectPromptInjection(content string) (bool, string) {
	if content == "" {
		return false, ""
	}
	for _, rule := range InjectionRules {
		if rule.Regex.MatchString(content) {
			return true, rule.Name
		}
	}
	return false, ""
}

type RedactionResult struct {
	CleanedText string
	Detections  map[string]int
	Modified    bool
}

// RedactCredentials removes API keys and tokens from content.
func RedactCredentials(content string) RedactionResult {
	return applyRules(content, CredentialRules)
}

// RedactPII removes personally identifiable information (emails, SSNs, credit cards).
func RedactPII(content string) RedactionResult {
	return applyRules(content, PIIRules)
}

// RedactAll strips both credentials and PII.
func RedactAll(content string) RedactionResult {
	r1 := applyRules(content, CredentialRules)
	r2 := applyRules(r1.CleanedText, PIIRules)

	combinedDetections := make(map[string]int)
	for k, v := range r1.Detections {
		combinedDetections[k] += v
	}
	for k, v := range r2.Detections {
		combinedDetections[k] += v
	}

	return RedactionResult{
		CleanedText: r2.CleanedText,
		Detections:  combinedDetections,
		Modified:    r1.Modified || r2.Modified,
	}
}

func applyRules(text string, rules []SensitiveRule) RedactionResult {
	if text == "" {
		return RedactionResult{CleanedText: text, Detections: nil, Modified: false}
	}

	res := text
	detections := make(map[string]int)
	modified := false

	for _, rule := range rules {
		matches := rule.Regex.FindAllStringIndex(res, -1)
		if len(matches) > 0 {
			detections[rule.Name] += len(matches)
			res = rule.Regex.ReplaceAllString(res, rule.Replacement)
			modified = true
		}
	}

	return RedactionResult{
		CleanedText: res,
		Detections:  detections,
		Modified:    modified,
	}
}

// RedactHeaders scrubs authorization and API key tokens from request headers.
func RedactHeaders(headers map[string][]string) map[string][]string {
	sanitized := make(map[string][]string)
	for k, vals := range headers {
		lower := strings.ToLower(k)
		if lower == "authorization" || lower == "x-api-key" || lower == "api-key" || lower == "apikey" {
			var masked []string
			for _, v := range vals {
				parts := strings.SplitN(v, " ", 2)
				if len(parts) == 2 {
					masked = append(masked, parts[0]+" [REDACTED:auth_header]")
				} else {
					masked = append(masked, "[REDACTED:auth_header]")
				}
			}
			sanitized[k] = masked
		} else {
			sanitized[k] = vals
		}
	}
	return sanitized
}
