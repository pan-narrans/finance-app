package webapp

import "embed"

// Assets contains the built static files for the Telegram Mini App.
//
//go:embed dist/*
var Assets embed.FS
