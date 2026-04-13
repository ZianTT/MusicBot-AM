package applemusic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/liuran001/MusicBot-Go/bot/platform"
)

func maskAppleMusicToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

func (a *AppleMusicPlatform) tokenStatusSummary(ctx context.Context) string {
	if a == nil || a.client == nil {
		return "- 状态: 插件未初始化"
	}
	token := strings.TrimSpace(a.client.Token())
	if token == "" {
		return "- 状态: 未配置 API Token"
	}
	lines := []string{
		"- 状态: 已配置 API Token",
		"- Token: " + maskAppleMusicToken(token),
	}
	checkCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if _, err := a.client.SearchSongs(checkCtx, "IOSYS", 1); err != nil {
		lines = append(lines, "- 鉴权校验: 失败", "- 错误: "+strings.TrimSpace(err.Error()))
	} else {
		lines = append(lines, "- 鉴权校验: 通过")
	}
	return strings.Join(lines, "\n")
}

func (a *AppleMusicPlatform) accountCheckMessage(ctx context.Context) (string, error) {
	if a == nil || a.client == nil {
		return "", fmt.Errorf("applemusic client unavailable")
	}
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if _, err := a.client.SearchSongs(checkCtx, "IOSYS", 1); err != nil {
		return fmt.Sprintf("Apple Music Token 校验失败: %v", err), nil
	}
	return "Apple Music Token 校验通过", nil
}

var (
	_ platform.AccountStatusProvider = (*AppleMusicPlatform)(nil)
	_ platform.CookieImporter        = (*AppleMusicPlatform)(nil)
	_ platform.CookieChecker         = (*AppleMusicPlatform)(nil)
	_ platform.LoginMethodProvider   = (*AppleMusicPlatform)(nil)
)
