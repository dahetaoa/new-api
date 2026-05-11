package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestListModelsGeminiMatchesOfficialEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousSelfUseMode := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = true
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = previousSelfUseMode
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1beta/models?pageSize=1", nil)
	common.SetContextKey(c, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(c, constant.ContextKeyTokenModelLimit, map[string]bool{
		"gpt-4o-mini":            true,
		"text-embedding-3-small": true,
	})

	ListModels(c, constant.ChannelTypeGemini)

	require.Equal(t, 200, w.Code)

	var resp struct {
		Models        []dto.GeminiModel `json:"models"`
		NextPageToken string            `json:"nextPageToken"`
	}
	err := common.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Len(t, resp.Models, 1)
	require.Equal(t, "1", resp.NextPageToken)
	require.NotEmpty(t, resp.Models[0].Name)
	require.Equal(t, geminiModelResourceName(resp.Models[0].BaseModelId), resp.Models[0].Name)
	require.NotEmpty(t, resp.Models[0].SupportedGenerationMethods)
}

func TestRetrieveModelGeminiReturnsModelResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NotEmpty(t, openAIModels)

	modelID := openAIModels[0].Id
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "model", Value: modelID}}
	c.Request = httptest.NewRequest("GET", "/v1beta/models/"+modelID, nil)

	RetrieveModel(c, constant.ChannelTypeGemini)

	require.Equal(t, 200, w.Code)

	var resp dto.GeminiModel
	err := common.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, geminiModelResourceName(modelID), resp.Name)
	require.Equal(t, modelID, resp.BaseModelId)
	require.Equal(t, modelID, resp.DisplayName)
	require.NotEmpty(t, resp.SupportedGenerationMethods)
}
