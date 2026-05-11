package globalpassthrough

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
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
