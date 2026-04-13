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
	baseURL := cfg.GetPluginString("applemusic", "base_url")
	timeoutSec := cfg.GetPluginInt("applemusic", "timeout")
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	storefront := cfg.GetPluginString("applemusic", "storefront")
	language := cfg.GetPluginString("applemusic", "language")
	apiToken := cfg.GetPluginString("applemusic", "api_token")
	client := NewClient(baseURL, apiToken, storefront, language, time.Duration(timeoutSec)*time.Second, logger)
	client.persistFunc = func(pairs map[string]string) error {
		return cfg.PersistPluginConfig("applemusic", pairs)
	}
	if err := client.SetAPIProxy(cfg.ResolveAPIProxyConfig("applemusic")); err != nil {
		return nil, err
	}
	return &platformplugins.Contribution{Platform: NewPlatform(client)}, nil
}
