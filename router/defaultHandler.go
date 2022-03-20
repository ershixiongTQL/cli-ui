package router

import (
	"io"
)

type DefaultHandler func(Input, io.StringWriter)

func UnitRegister(name string, pattern string, callback DefaultHandler) error {
	return UnitRegisterDefault(name, pattern, callback)
}

func UnitRegisterDefault(name string, pattern string, callback DefaultHandler) (err error) {
	registered, err := unitRegister(name, pattern)
	if err != nil {
		return
	}
	registered.defaulthandler = callback
	return
}

func defaultHandlerCall(unit *unit, input Input, resultIO io.StringWriter) {
	if unit != nil {
		if h := unit.defaulthandler; h != nil {
			h(input, resultIO)
		}
	}
}
