package main

import "github.com/charmbracelet/lipgloss"

var (
	bg      = lipgloss.Color("#16161e")
	accentC = lipgloss.Color("#00CED1")
	blueC   = lipgloss.Color("#5B8DEF")
	greenC  = lipgloss.Color("#4CCD6B")
	redC    = lipgloss.Color("#E06C75")
	muteC   = lipgloss.Color("#5c6370")
	borderC = lipgloss.Color("#2c313a")
	sbarBg  = lipgloss.Color("#16161e")
	dialogBg = lipgloss.Color("#1a1a2e")
	tabBg   = lipgloss.Color("#0f0f1a")
)

var agentColors = []lipgloss.Color{
	lipgloss.Color("#00CED1"), lipgloss.Color("#5B8DEF"),
	lipgloss.Color("#4CCD6B"), lipgloss.Color("#E6C06C"),
	lipgloss.Color("#C678DD"), lipgloss.Color("#E06C75"),
	lipgloss.Color("#56B6C2"), lipgloss.Color("#D19A66"),
}

var providers = []provider{
	{"anthropic", "Anthropic"}, {"openai", "OpenAI"},
	{"gemini", "Gemini"}, {"mistral", "Mistral"},
	{"deepseek", "DeepSeek"}, {"xai", "xAI"}, {"nvidia", "NVIDIA"},
	{"groq", "Groq"}, {"together", "Together"},
	{"openrouter", "OpenRouter"}, {"cohere", "Cohere"},
}

var keyPrefixes = map[string]string{
	"anthropic":  "sk-ant-",
	"openai":     "sk-",
	"groq":       "gsk_",
	"deepseek":   "sk-",
	"nvidia":     "",
	"together":   "",
	"openrouter": "sk-or-",
	"cohere":     "",
}

func keyWarning(provider, key string) string {
	prefix, ok := keyPrefixes[provider]
	if !ok || prefix == "" || key == "" {
		return ""
	}
	if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
		return ""
	}
	return "Key format looks unusual — expected prefix: " + prefix
}

var baseURLs = map[string]string{
	"anthropic":  "https://api.anthropic.com",
	"openai":     "https://api.openai.com",
	"gemini":     "https://generativelanguage.googleapis.com",
	"mistral":    "https://api.mistral.ai",
	"deepseek":   "https://api.deepseek.com",
	"xai":        "https://api.x.ai",
	"nvidia":     "https://integrate.api.nvidia.com",
	"groq":       "https://api.groq.com/openai",
	"together":   "https://api.together.xyz",
	"openrouter": "https://openrouter.ai/api",
	"cohere":     "https://api.cohere.com",
}
