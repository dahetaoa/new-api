package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const tokenRateLimitKeyPrefix = "tokenRateLimit"

var tokenRateLimitRedisScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], tonumber(ARGV[2]))
end
if current > tonumber(ARGV[1]) then
  return {0, current}
end
return {1, current}
`)

type fixedWindowCounter struct {
	Count     int64
	ExpiresAt int64
}

type fixedWindowMemoryCounter struct {
	mutex       sync.Mutex
	counters    map[string]fixedWindowCounter
	lastCleanup int64
}

var tokenRateLimitMemoryCounter = &fixedWindowMemoryCounter{
	counters: make(map[string]fixedWindowCounter),
}

func checkTokenRateLimit(c *gin.Context, modelName string) bool {
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenRateLimitEnabled) {
		return true
	}
	value, ok := common.GetContextKey(c, constant.ContextKeyTokenRateLimit)
	if !ok {
		return true
	}
	config, ok := value.(model.TokenRateLimitConfig)
	if !ok || !config.HasLimit() {
		return true
	}
	tokenId := common.GetContextKeyInt(c, constant.ContextKeyTokenId)
	if tokenId <= 0 {
		return true
	}
	if !checkTokenRateLimitRule(c, tokenId, "total", "", config.Total) {
		return false
	}
	if modelName == "" {
		return true
	}
	modelRule, found := config.GetModelRule(modelName)
	matchedModelName := modelName
	if !found {
		matchName := ratio_setting.FormatMatchingModelName(modelName)
		if matchName != modelName {
			modelRule, found = config.GetModelRule(matchName)
			if found {
				matchedModelName = matchName
			}
		}
	}
	if found {
		modelHash := common.Sha1([]byte(matchedModelName))
		if !checkTokenRateLimitRule(c, tokenId, "model", modelHash, modelRule) {
			return false
		}
	}
	return true
}

func checkTokenRateLimitRule(c *gin.Context, tokenId int, scope string, scopeKey string, rule model.TokenRateLimitRule) bool {
	if !rule.HasLimit() {
		return true
	}
	now := time.Now().UTC()
	checks := []struct {
		name       string
		limit      int
		period     string
		expiration time.Duration
	}{
		{name: "RPM", limit: rule.RPM, period: now.Format("200601021504"), expiration: 2 * time.Minute},
		{name: "RPH", limit: rule.RPH, period: now.Format("2006010215"), expiration: 2 * time.Hour},
		{name: "RPD", limit: rule.RPD, period: now.Format("20060102"), expiration: 48 * time.Hour},
	}
	for _, check := range checks {
		if check.limit <= 0 {
			continue
		}
		key := fmt.Sprintf("%s:%d:%s:%s:%s:%s", tokenRateLimitKeyPrefix, tokenId, scope, scopeKey, check.name, check.period)
		allowed, err := allowTokenRateLimitRequest(key, check.limit, check.expiration)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusInternalServerError, "rate_limit_check_failed")
			return false
		}
		if !allowed {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests, buildTokenRateLimitMessage(scope, check.name, check.limit))
			return false
		}
	}
	return true
}

func buildTokenRateLimitMessage(scope string, name string, limit int) string {
	period := "分钟"
	switch name {
	case "RPH":
		period = "小时"
	case "RPD":
		period = "天"
	}
	if scope == "model" {
		return fmt.Sprintf("当前令牌已达到该模型限速：每%s最多请求%d次", period, limit)
	}
	return fmt.Sprintf("当前令牌已达到总限速：每%s最多请求%d次", period, limit)
}

func allowTokenRateLimitRequest(key string, limit int, expiration time.Duration) (bool, error) {
	if limit <= 0 {
		return true, nil
	}
	if common.RedisEnabled {
		return allowTokenRateLimitRequestRedis(key, limit, expiration)
	}
	return tokenRateLimitMemoryCounter.allow(key, limit, expiration), nil
}

func allowTokenRateLimitRequestRedis(key string, limit int, expiration time.Duration) (bool, error) {
	result, err := tokenRateLimitRedisScript.Run(
		context.Background(),
		common.RDB,
		[]string{key},
		limit,
		int64(expiration.Seconds()),
	).Result()
	if err != nil {
		return false, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) == 0 {
		return false, fmt.Errorf("unexpected redis rate limit result: %v", result)
	}
	allowed, ok := values[0].(int64)
	if !ok {
		return false, fmt.Errorf("unexpected redis rate limit allowed value: %v", values[0])
	}
	return allowed == 1, nil
}

func (counter *fixedWindowMemoryCounter) allow(key string, limit int, expiration time.Duration) bool {
	now := time.Now().Unix()
	expireAt := time.Now().Add(expiration).Unix()
	counter.mutex.Lock()
	defer counter.mutex.Unlock()
	if counter.counters == nil {
		counter.counters = make(map[string]fixedWindowCounter)
	}
	if now-counter.lastCleanup > 60 {
		for k, item := range counter.counters {
			if item.ExpiresAt <= now {
				delete(counter.counters, k)
			}
		}
		counter.lastCleanup = now
	}
	item := counter.counters[key]
	if item.ExpiresAt <= now {
		item = fixedWindowCounter{ExpiresAt: expireAt}
	}
	item.Count++
	counter.counters[key] = item
	return item.Count <= int64(limit)
}
