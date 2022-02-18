package interfaces

import (
	"io"
)

type UI_AGENT_FE_TYPE uint

const (
	UI_AGENT_FE_TYPE_TELNET = iota
)

type UIAgentInterface interface {
	Start() error
	Stop()
}

type BackEndInterface interface {
	Completer(input string) (completions []string)
	Helps(input string) (help string)
	CommandHandler(command string, resultIO io.StringWriter) error
	UserAuth(username string, passwd string) bool
}
