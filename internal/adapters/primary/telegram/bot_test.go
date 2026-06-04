package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/telebot.v3"
)

func TestGetCleanedText(t *testing.T) {
	adapter := &TelegramAdapter{
		teleBot: &telebot.Bot{
			Me: &telebot.User{
				Username: "MyBot",
			},
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No mention",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "Mention at start",
			input:    "@MyBot Hello",
			expected: "Hello",
		},
		{
			name:     "Mention at end",
			input:    "Hello @MyBot",
			expected: "Hello",
		},
		{
			name:     "Mention in middle",
			input:    "Hello @MyBot world",
			expected: "Hello world",
		},
		{
			name:     "Multiple mentions",
			input:    "@MyBot Hello @MyBot",
			expected: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				c := &mockContext{text: tt.input}
				result := adapter.getCleanedText(c)
				assert.Equal(t, tt.expected, result)
			},
		)
	}
}

type mockContext struct {
	telebot.Context
	text string
}

func (c *mockContext) Text() string {
	return c.text
}
