package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createLogForStatTest(t *testing.T, log *Log) {
	t.Helper()
	log.Id = 0
	require.NoError(t, LOG_DB.Create(log).Error)
}

func TestGetLogStatAggregatesTokensWithFilters(t *testing.T) {
	truncateTables(t)
	initCol()

	now := time.Now().Unix()
	base := Log{
		UserId:           1,
		Username:         "alice",
		CreatedAt:        now - 10,
		Type:             LogTypeConsume,
		ModelName:        "gpt-4o",
		TokenName:        "primary",
		ChannelId:        2,
		Group:            "default",
		RequestId:        "req-a",
		IsStream:         true,
		UseTime:          1,
		Content:          "ok",
		Ip:               "127.0.0.1",
		Other:            "{}",
		Quota:            10,
		PromptTokens:     100,
		CompletionTokens: 25,
	}

	createLogForStatTest(t, &base)

	second := base
	second.CreatedAt = now - 20
	second.Quota = 7
	second.PromptTokens = 30
	second.CompletionTokens = 9
	createLogForStatTest(t, &second)

	otherUser := base
	otherUser.UserId = 2
	otherUser.Username = "bob"
	otherUser.Quota = 50
	otherUser.PromptTokens = 500
	otherUser.CompletionTokens = 500
	createLogForStatTest(t, &otherUser)

	otherRequest := base
	otherRequest.RequestId = "req-b"
	otherRequest.Quota = 80
	otherRequest.PromptTokens = 80
	otherRequest.CompletionTokens = 80
	createLogForStatTest(t, &otherRequest)

	errorLog := base
	errorLog.Type = LogTypeError
	errorLog.Quota = 999
	errorLog.PromptTokens = 5
	errorLog.CompletionTokens = 6
	createLogForStatTest(t, &errorLog)

	stat, err := GetLogStat(LogFilters{
		LogType:        LogTypeConsume,
		StartTimestamp: now - 60,
		EndTimestamp:   now,
		ModelName:      "gpt%",
		UserId:         1,
		TokenName:      "primary",
		Channel:        2,
		Group:          "default",
		RequestId:      "req-a",
	})

	require.NoError(t, err)
	require.Equal(t, 17, stat.Quota)
	require.Equal(t, 2, stat.Rpm)
	require.Equal(t, 164, stat.Tpm)
	require.Equal(t, 130, stat.PromptTokens)
	require.Equal(t, 34, stat.CompletionTokens)
}

func TestGetLogStatEmptyResultReturnsZero(t *testing.T) {
	truncateTables(t)
	initCol()

	stat, err := GetLogStat(LogFilters{
		LogType:   LogTypeConsume,
		RequestId: "missing",
	})

	require.NoError(t, err)
	require.Equal(t, Stat{}, stat)
}
