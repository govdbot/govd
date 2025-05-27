package redgifs

import (
	"fmt"
	"net/http"
	"time"

	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"

	"github.com/bytedance/sonic"
)

var accessToken *Token

func GetAccessToken(
	client models.HTTPClient,
	cookies []*http.Cookie,
) (*Token, error) {
	if accessToken == nil || time.Now().Unix() >= accessToken.ExpiresIn {
		if err := RefreshAccessToken(client, cookies); err != nil {
			return nil, err
		}
	}
	return accessToken, nil
}

func RefreshAccessToken(
	client models.HTTPClient,
	cookies []*http.Cookie,
) error {
	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		tokenEndpoint,
		nil,
		nil,
		cookies,
	)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get access token: %s", resp.Status)
	}
	var token Token
	err = sonic.ConfigFastest.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	token.ExpiresIn = time.Now().Add(23 * time.Hour).Unix()
	accessToken = &token
	return nil
}
