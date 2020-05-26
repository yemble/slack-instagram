package service

import (
	"encoding/json"
)

type Config struct {
	QueueURL     string               `json:"queue_url"`
	SlackTeams   map[string]*TeamInfo `json:"slack_teams"`
	CookieString string               `json:"cookies"`
}

type TeamInfo struct {
	Name       string `json:"name"`
	OauthToken string `json:"oauth_token"`
}

func NewConfigFromJSON(j []byte) (*Config, error) {
	cfg := &Config{}

	if err := json.Unmarshal(j, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) TeamByRequestToken(t string) *TeamInfo {
	if t, ok := c.SlackTeams[t]; ok {
		return t
	}
	return nil
}
