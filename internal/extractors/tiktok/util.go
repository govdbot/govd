package tiktok

import "net/url"

func RequestParams(videoID string) url.Values {
	return url.Values{
		"item_ids":                  {videoID},
		"language":                  {"en-US"},
		"aid":                       {"1284"},
		"app_name":                  {"tiktok_web"},
		"device_platform":           {"web_pc"},
		"referer":                   {""},
		"screen_width":              {"1366"},
		"screen_height":             {"768"},
		"browser_language":          {"en-US"},
		"browser_platform":          {"Linux x86_64"},
		"browser_name":              {"Mozilla"},
		"browser_version":           {"5.0 (X11)"},
		"browser_online":            {"false"},
		"app_language":              {"en"},
		"timezone_name":             {"Etc/Utc"},
		"is_page_visible":           {"true"},
		"focus_state":               {"true"},
		"is_fullscreen":             {"false"},
		"history_len":               {"2"},
		"security_verification_aid": {""},
	}
}
