package applemusic

import (
	"fmt"
	"time"

	"github.com/liuran001/MusicBot-Go/bot/config"
	logpkg "github.com/liuran001/MusicBot-Go/bot/logger"
	platformplugins "github.com/liuran001/MusicBot-Go/bot/platform/plugins"
)

func init() {
	if err := platformplugins.Register("applemusic", buildContribution); err != nil {
		panic(err)
	}
}

func buildContribution(cfg *config.Config, logger *logpkg.Logger) (*platformplugins.Contribution, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config required")
	}
	token := cfg.GetPluginString("applemusic", "token")
	if token == "" {
		return nil, fmt.Errorf("applemusic plugin requires token")
	}
	baseURL := cfg.GetPluginString("applemusic", "base_url")
	if baseURL == "" {
		baseURL = "https://am.1641263.xyz"
	}
	timeoutSec := cfg.GetPluginInt("applemusic", "timeout")
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	client := NewClient(baseURL, token, time.Duration(timeoutSec)*time.Second, logger)
	if err := client.SetAPIProxy(cfg.ResolveAPIProxyConfig("applemusic")); err != nil {
		return nil, err
	}
	return &platformplugins.Contribution{Platform: NewPlatform(client)}, nil
}
