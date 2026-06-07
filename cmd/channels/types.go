package main

import "context"

type InboundMessage struct {
	ChannelType string `json:"channel_type"`
	ChannelID   string `json:"channel_id"`
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	Content     string `json:"content"`
	ThreadID    string `json:"thread_id,omitempty"`
	Timestamp   string `json:"timestamp"`
}

type OutboundMessage struct {
	Content  string `json:"content"`
	ThreadID string `json:"thread_id,omitempty"`
	Format   string `json:"format,omitempty"`
}

type Target struct {
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id,omitempty"`
}

type AdapterInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

type ChannelAdapter interface {
	Start(ctx context.Context) error
	Stop() error
	SendMessage(ctx context.Context, target Target, msg OutboundMessage) error
	OnMessage(handler func(ctx context.Context, msg InboundMessage) error)
	Info() AdapterInfo
}

type ChannelConfig struct {
	Name        string            `json:"name"`
	AdapterType string            `json:"adapter_type"`
	AgentName   string            `json:"agent_name"`
	Credentials map[string]string `json:"credentials,omitempty"`
	Overrides   map[string]string `json:"overrides,omitempty"`
	Policies    ChannelPolicies   `json:"policies"`
}

type ChannelPolicies struct {
	DM            string `json:"dm"`
	Group         string `json:"group"`
	RateLimitPM   int    `json:"rate_limit_per_minute"`
	RateLimitUser int    `json:"rate_limit_per_user"`
}
