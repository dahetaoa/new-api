package globalpassthrough

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveModelsPath(t *testing.T) {
	require.Equal(t, "/v1/models", ResolveModelsPath(dto.ChannelSettings{}))
	require.Equal(t, "/openai/v1/models", ResolveModelsPath(dto.ChannelSettings{
		GlobalPassthroughOpenAIChatPath: "/openai/v1/chat/completions",
	}))
}

func TestGetRequestURLResponsesCompact(t *testing.T) {
	adaptor := &Adaptor{}
	url, err := adaptor.GetRequestURL(&relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponsesCompaction,
		RelayMode:   relayconstant.RelayModeResponsesCompact,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeGlobalPassthrough,
			ChannelBaseUrl: "https://example.com",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/v1/responses/compact", url)
}

func TestGetRequestURLGeminiStream(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatGemini,
		RelayMode:   relayconstant.RelayModeGemini,
		IsStream:    true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-2.5-pro",
			ChannelType:       constant.ChannelTypeGlobalPassthrough,
			ChannelBaseUrl:    "https://example.com",
		},
	}
	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/v1beta/models/gemini-2.5-pro:streamGenerateContent?alt=sse", url)
	require.True(t, info.DisablePing)
}

func TestGetRequestURLClaudeBetaQuery(t *testing.T) {
	adaptor := &Adaptor{}
	url, err := adaptor.GetRequestURL(&relaycommon.RelayInfo{
		RelayFormat:       types.RelayFormatClaude,
		RelayMode:         relayconstant.RelayModeChatCompletions,
		IsClaudeBetaQuery: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeGlobalPassthrough,
			ChannelBaseUrl: "https://example.com",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/v1/messages?beta=true", url)
}

func TestGetRequestURLUsesOriginalPathAndQuery(t *testing.T) {
	adaptor := &Adaptor{}
	url, err := adaptor.GetRequestURL(&relaycommon.RelayInfo{
		RelayFormat:    types.RelayFormatRerank,
		RelayMode:      relayconstant.RelayModeRerank,
		RequestURLPath: "/v1/rerank?top_n=3&return_documents=true",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeGlobalPassthrough,
			ChannelBaseUrl: "https://example.com/upstream",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/upstream/v1/rerank?top_n=3&return_documents=true", url)
}

func TestGetRequestURLGeminiStripsGatewayKeyQuery(t *testing.T) {
	adaptor := &Adaptor{}
	url, err := adaptor.GetRequestURL(&relaycommon.RelayInfo{
		RelayFormat:    types.RelayFormatGemini,
		RelayMode:      relayconstant.RelayModeGemini,
		RequestURLPath: "/v1beta/models/gemini-2.5-pro:generateContent?key=client-token&alt=sse&foo=bar",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeGlobalPassthrough,
			ChannelBaseUrl: "https://example.com",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/v1beta/models/gemini-2.5-pro:generateContent?alt=sse&foo=bar", url)
}

func TestSetupRequestHeaderCopiesSafeHeadersAndRewritesAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("User-Agent", "codex-cli")
	ctx.Request.Header.Set("originator", "codex_cli_rs")
	ctx.Request.Header.Set("session_id", "sess-123")
	ctx.Request.Header.Set("conversation_id", "conv-456")
	ctx.Request.Header.Set("OpenAI-Beta", "responses=v1")
	ctx.Request.Header.Set("OpenAI-Organization", "org-123")
	ctx.Request.Header.Set("OpenAI-Project", "proj-123")
	ctx.Request.Header.Set("Idempotency-Key", "idem-123")
	ctx.Request.Header.Set("x-client-request-id", "client-123")
	ctx.Request.Header.Set("x-codex-turn-state", "turn-state")
	ctx.Request.Header.Add("X-Custom", "one")
	ctx.Request.Header.Add("X-Custom", "two")
	ctx.Request.Header.Set("Authorization", "Bearer client-key")
	ctx.Request.Header.Set("Cookie", "session=secret")
	ctx.Request.Header.Set("Accept-Encoding", "gzip")
	ctx.Request.Header.Set("Content-Length", "123")
	ctx.Request.Header.Set("Sec-WebSocket-Key", "ws-key")
	ctx.Request.Header.Set("X-Api-Key", "client-api-key")

	header := http.Header{}
	err := (&Adaptor{}).SetupRequestHeader(ctx, &header, &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponses,
		IsStream:    true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "channel-key",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "application/json", header.Get("Content-Type"))
	require.Equal(t, "text/event-stream", header.Get("Accept"))
	require.Equal(t, "codex-cli", header.Get("User-Agent"))
	require.Equal(t, "codex_cli_rs", header.Get("originator"))
	require.Equal(t, "sess-123", header.Get("session_id"))
	require.Equal(t, "conv-456", header.Get("conversation_id"))
	require.Equal(t, "responses=v1", header.Get("OpenAI-Beta"))
	require.Equal(t, "org-123", header.Get("OpenAI-Organization"))
	require.Equal(t, "proj-123", header.Get("OpenAI-Project"))
	require.Equal(t, "idem-123", header.Get("Idempotency-Key"))
	require.Equal(t, "client-123", header.Get("x-client-request-id"))
	require.Equal(t, "turn-state", header.Get("x-codex-turn-state"))
	require.ElementsMatch(t, []string{"one", "two"}, header.Values("X-Custom"))
	require.Equal(t, "Bearer channel-key", header.Get("Authorization"))
	require.Empty(t, header.Get("Cookie"))
	require.Empty(t, header.Get("Accept-Encoding"))
	require.Empty(t, header.Get("Content-Length"))
	require.Empty(t, header.Get("Sec-WebSocket-Key"))
	require.Empty(t, header.Get("X-Api-Key"))
}
