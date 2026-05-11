package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func IsGlobalPassthroughChannelType(channelType int) bool {
	return channelType == constant.ChannelTypeGlobalPassthrough
}

func IsGlobalPassthroughChannel(info *RelayInfo) bool {
	if info == nil {
		return false
	}
	return IsGlobalPassthroughChannelType(info.ChannelType)
}

func ShouldDirectPassthrough(info *RelayInfo) bool {
	return IsGlobalPassthroughChannel(info)
}

func ShouldPassThroughRequestBody(info *RelayInfo) bool {
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled {
		return true
	}
	if info == nil {
		return false
	}
	if ShouldDirectPassthrough(info) {
		return true
	}
	return info.ChannelSetting.PassThroughBodyEnabled
}

func NormalizePassthroughPath(path string, defaultPath string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return defaultPath
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}
