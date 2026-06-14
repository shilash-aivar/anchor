package config

import "strings"

// NormalizeTier maps aliases (prod, PRD, dev, …) to canonical tier names.
func NormalizeTier(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "prod", "production", "prd", "live":
		return "production"
	case "stg", "staging", "stage":
		return "staging"
	case "dev", "development", "devel":
		return "development"
	case "":
		return "development"
	default:
		return strings.ToLower(strings.TrimSpace(tier))
	}
}

// TierAbbrev returns PRD / STG / DEV for prompts and whoami.
func TierAbbrev(tier string) string {
	switch NormalizeTier(tier) {
	case "production":
		return "PRD"
	case "staging":
		return "STG"
	default:
		return "DEV"
	}
}

// IsProductionTier reports whether tier is production after normalization.
func IsProductionTier(tier string) bool {
	return NormalizeTier(tier) == "production"
}
