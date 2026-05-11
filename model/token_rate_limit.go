package model

import (
	"errors"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	MaxTokenRateLimitValue = 1000000000
	maxRateLimitModelName  = 191
)

type TokenRateLimitRule struct {
	RPM int `json:"rpm"`
	RPH int `json:"rph"`
	RPD int `json:"rpd"`
}

type TokenModelRateLimitRule struct {
	ModelName string `json:"model_name"`
	TokenRateLimitRule
}

type TokenRateLimitConfig struct {
	Total  TokenRateLimitRule        `json:"total"`
	Models []TokenModelRateLimitRule `json:"models"`
}

func (rule TokenRateLimitRule) HasLimit() bool {
	return rule.RPM > 0 || rule.RPH > 0 || rule.RPD > 0
}

func (config TokenRateLimitConfig) HasLimit() bool {
	if config.Total.HasLimit() {
		return true
	}
	for _, rule := range config.Models {
		if strings.TrimSpace(rule.ModelName) != "" && rule.TokenRateLimitRule.HasLimit() {
			return true
		}
	}
	return false
}

func (config TokenRateLimitConfig) GetModelRule(modelName string) (TokenRateLimitRule, bool) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return TokenRateLimitRule{}, false
	}
	for _, rule := range config.Models {
		if rule.ModelName == modelName && rule.TokenRateLimitRule.HasLimit() {
			return rule.TokenRateLimitRule, true
		}
	}
	return TokenRateLimitRule{}, false
}

func (token *Token) GetRateLimitConfig() (TokenRateLimitConfig, error) {
	if token == nil || strings.TrimSpace(token.RateLimits) == "" {
		return TokenRateLimitConfig{}, nil
	}
	var config TokenRateLimitConfig
	if err := common.UnmarshalJsonStr(token.RateLimits, &config); err != nil {
		return TokenRateLimitConfig{}, err
	}
	return NormalizeTokenRateLimitConfig(config)
}

func NormalizeTokenRateLimitConfig(config TokenRateLimitConfig) (TokenRateLimitConfig, error) {
	total, err := normalizeRateLimitRule(config.Total)
	if err != nil {
		return TokenRateLimitConfig{}, err
	}
	modelsByName := make(map[string]TokenModelRateLimitRule)
	for _, modelRule := range config.Models {
		modelName := strings.TrimSpace(modelRule.ModelName)
		if modelName == "" {
			continue
		}
		if len(modelName) > maxRateLimitModelName {
			return TokenRateLimitConfig{}, errors.New("模型名称过长")
		}
		rule, err := normalizeRateLimitRule(modelRule.TokenRateLimitRule)
		if err != nil {
			return TokenRateLimitConfig{}, err
		}
		if !rule.HasLimit() {
			continue
		}
		modelsByName[modelName] = TokenModelRateLimitRule{
			ModelName:          modelName,
			TokenRateLimitRule: rule,
		}
	}
	modelNames := make([]string, 0, len(modelsByName))
	for modelName := range modelsByName {
		modelNames = append(modelNames, modelName)
	}
	sort.Strings(modelNames)
	models := make([]TokenModelRateLimitRule, 0, len(modelNames))
	for _, modelName := range modelNames {
		models = append(models, modelsByName[modelName])
	}
	return TokenRateLimitConfig{
		Total:  total,
		Models: models,
	}, nil
}

func normalizeRateLimitRule(rule TokenRateLimitRule) (TokenRateLimitRule, error) {
	if rule.RPM < 0 || rule.RPH < 0 || rule.RPD < 0 {
		return TokenRateLimitRule{}, errors.New("限速值不能为负数")
	}
	if rule.RPM > MaxTokenRateLimitValue || rule.RPH > MaxTokenRateLimitValue || rule.RPD > MaxTokenRateLimitValue {
		return TokenRateLimitRule{}, errors.New("限速值过大")
	}
	return rule, nil
}
