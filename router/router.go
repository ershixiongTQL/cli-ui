package router

import (
	"fmt"
	"io"
	"regexp"
	"sync"
)

var commandSubscribs = make(map[string]*unit)
var lock sync.Mutex

type unit struct {
	name     string
	pattern  string
	compiled *regexp.Regexp

	//Handlers
	progressHandler ProgressHandler
	defaulthandler  DefaultHandler
}

func unitRegister(name string, pattern string) (registered *unit, err error) {
	lock.Lock()
	defer lock.Unlock()

	if _, exist := commandSubscribs[pattern]; exist {
		return nil, fmt.Errorf("pattern registered multiple times")
	}

	registered = &unit{
		name:     name,
		pattern:  pattern,
		compiled: regexp.MustCompile(pattern),
	}

	commandSubscribs[pattern] = registered

	return
}

func (u *unit) Call(input Input, resultIO io.StringWriter) {
	progressHandlerCall(u, input, resultIO)
	defaultHandlerCall(u, input, resultIO)
}

func Mux(command string, resultIO io.StringWriter) (err error) {

	handlerCnt := 0

	for _, unit := range commandSubscribs {
		if found := unit.compiled.FindStringSubmatch(command); found != nil {
			unit.Call(createInput(command, found[1:], unit.name), resultIO)
			handlerCnt++
		}
	}

	if handlerCnt == 0 {
		resultIO.WriteString("No handler for the command \"" + command + "\"!")
		err = fmt.Errorf("mux nothing")
	}

	return
}
