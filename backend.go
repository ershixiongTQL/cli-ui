package cliui

import (
	"io"
	"log"

	"github.com/ershixiongTQL/cli-ui/completer"
	"github.com/ershixiongTQL/cli-ui/router"
)

type uiBackend struct {
	completer completer.Completer
}

func (be *uiBackend) Completer(input string) (completions []string) {
	return be.completer.GetCompletes(input)
}

func (be *uiBackend) Helps(input string) (help string) {
	return be.completer.GetHelps(input)
}

func (be *uiBackend) CommandHandler(command string, resultIO io.StringWriter) error {
	return router.Mux(command, resultIO)
}

func (be *uiBackend) UserAuth(username string, passwd string) bool {
	//TODO: Auth system
	return true
}

func backendPrepare(configFilePath string) (be *uiBackend) {
	be = new(uiBackend)

	if err := be.completer.Setup(configFilePath); err != nil {
		log.Println(err.Error())
		return nil
	}

	return
}
