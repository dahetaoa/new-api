package globalpassthrough

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/gemini"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const (
	ChannelName = "Global Passthrough"

	DefaultOpenAIResponsePath = "/v1/responses"
	DefaultOpenAIChatPath     = "/v1/chat/completions"
	DefaultGeminiPathTemplate = "/v1beta/models/{model}:{action}"
	DefaultClaudePath         = "/v1/messages"
	DefaultModelsPath         = "/v1/models"
)

type Adaptor struct{}

func ResolveOpenAIResponsePath(setting dto.ChannelSettings) string {
	return relaycommon.NormalizePassthroughPath(
		setting.GlobalPassthroughOpenAIResponsePath,
		DefaultOpenAIResponsePath,
	)
}

func ResolveOpenAIChatPath(setting dto.ChannelSettings) string {
	return relaycommon.NormalizePassthroughPath(
		setting.GlobalPassthroughOpenAIChatPath,
		DefaultOpenAIChatPath,
	)
}

func ResolveGeminiPathTemplate(setting dto.ChannelSettings) string {
	return relaycommon.NormalizePassthroughPath(
		setting.GlobalPassthroughGeminiPath,
		DefaultGeminiPathTemplate,
	)
}

func ResolveClaudePath(setting dto.ChannelSettings) string {
	return relaycommon.NormalizePassthroughPath(
		setting.GlobalPassthroughClaudePath,
		DefaultClaudePath,
	)
}

func ResolveModelsPath(setting dto.ChannelSettings) string {
	chatPath := ResolveOpenAIChatPath(setting)
	switch {
	case strings.HasSuffix(chatPath, "/chat/completions"):
		return strings.TrimSuffix(chatPath, "/chat/completions") + "/models"
	case strings.HasSuffix(chatPath, "/completions"):
		return strings.TrimSuffix(chatPath, "/completions") + "/models"
	default:
		return DefaultModelsPath
	}
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil {
		return "", errors.New("relay info is nil")
	}
	switch info.RelayFormat {
	case types.RelayFormatOpenAIResponses, types.RelayFormatOpenAIResponsesCompaction:
		requestPath := ResolveOpenAIResponsePath(info.ChannelSetting)
		if info.RelayMode == relayconstant.RelayModeResponsesCompact {
			requestPath = strings.TrimRight(requestPath, "/") + "/compact"
		}
		return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, requestPath, info.ChannelType), nil
	case types.RelayFormatOpenAI:
		switch info.RelayMode {
		case relayconstant.RelayModeChatCompletions:
		default:
			return "", fmt.Errorf("global passthrough channel: unsupported relay mode %d for OpenAI request", info.RelayMode)
		}
		requestPath := ResolveOpenAIChatPath(info.ChannelSetting)
		return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, requestPath, info.ChannelType), nil
	case types.RelayFormatClaude:
		requestPath := ResolveClaudePath(info.ChannelSetting)
		if info.IsClaudeBetaQuery && !strings.Contains(requestPath, "beta=") {
			separator := "?"
			if strings.Contains(requestPath, "?") {
				separator = "&"
			}
			requestPath = requestPath + separator + "beta=true"
		}
		return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, requestPath, info.ChannelType), nil
	case types.RelayFormatGemini:
		if info.RelayMode != relayconstant.RelayModeGemini {
			return "", fmt.Errorf("global passthrough channel: unsupported relay mode %d for Gemini request", info.RelayMode)
		}
		if strings.Contains(info.RequestURLPath, ":embedContent") || strings.Contains(info.RequestURLPath, ":batchEmbedContents") {
			return "", errors.New("global passthrough channel: Gemini embeddings are not supported")
		}

		action := "generateContent"
		if info.IsStream {
			action = "streamGenerateContent?alt=sse"
			info.DisablePing = true
		}
		requestPath := ResolveGeminiPathTemplate(info.ChannelSetting)
		requestPath = strings.ReplaceAll(requestPath, "{model}", info.UpstreamModelName)
		requestPath = strings.ReplaceAll(requestPath, "{action}", action)
		return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, requestPath, info.ChannelType), nil
	default:
		return "", fmt.Errorf("global passthrough channel: unsupported relay format %s", info.RelayFormat)
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		req.Set("x-api-key", info.ApiKey)
		anthropicVersion := c.Request.Header.Get("anthropic-version")
		if anthropicVersion == "" {
			anthropicVersion = "2023-06-01"
		}
		req.Set("anthropic-version", anthropicVersion)
		claude.CommonClaudeHeadersOperation(c, req, info)
	case types.RelayFormatGemini:
		req.Set("x-goog-api-key", info.ApiKey)
	default:
		req.Set("Authorization", "Bearer "+info.ApiKey)
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("global passthrough channel: endpoint not supported")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("global passthrough channel: endpoint not supported")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("global passthrough channel: endpoint not supported")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("global passthrough channel: endpoint not supported")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		claudeAdaptor := &claude.Adaptor{}
		return claudeAdaptor.DoResponse(c, resp, info)
	case types.RelayFormatGemini:
		geminiAdaptor := &gemini.Adaptor{}
		return geminiAdaptor.DoResponse(c, resp, info)
	default:
		openAIAdaptor := &openai.Adaptor{}
		return openAIAdaptor.DoResponse(c, resp, info)
	}
}

func (a *Adaptor) GetModelList() []string {
	return []string{}
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
