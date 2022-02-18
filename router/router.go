package router

import (
	"fmt"
	"io"
	"regexp"
	"sync"
)

type Input interface {
	GetSegment(int) (string, error)
	SegmentExist(int) bool
	GetRaw() string
	GetUnitName() string
}

type CommandHandlerNew func(input Input, response io.StringWriter)

type CmdInput struct {
	subMatches []string
	raw        string
	unitName   string
}

func (c *CmdInput) GetSegment(index int) (seg string, err error) {
	if index >= len(c.subMatches) {
		return "", fmt.Errorf("segment index out of range")
	}
	return c.subMatches[index], nil
}

func (c *CmdInput) SegmentExist(index int) bool {
	if index >= len(c.subMatches) {
		return false
	}
	return c.subMatches[index] != ""
}

func (c *CmdInput) GetRaw() string {
	return c.raw
}

func (c *CmdInput) GetUnitName() string {
	return c.unitName
}

type Unit struct {
	name     string
	pattern  string
	compiled *regexp.Regexp
	handler  CommandHandlerNew
}

var commandSubscribersNew = make(map[string]Unit)
var subscribersMutex sync.Mutex

func UnitRegister(name string, pattern string, callback CommandHandlerNew) (err error) {

	subscribersMutex.Lock()
	defer subscribersMutex.Unlock()

	var unit Unit

	if callback == nil {
		return fmt.Errorf("invalid callback function")
	}

	unit.compiled = regexp.MustCompile(pattern)

	unit.name = name
	unit.pattern = pattern
	unit.handler = callback

	if _, exist := commandSubscribersNew[pattern]; exist {
		return fmt.Errorf("pattern registered multiple times")
	}

	commandSubscribersNew[pattern] = unit
	return
}

func Mux(command string, resultIO io.StringWriter) (err error) {

	handlerCnt := 0

	for _, unit := range commandSubscribersNew {
		if found := unit.compiled.FindStringSubmatch(command); found != nil {
			input := CmdInput{raw: command, subMatches: found[1:], unitName: unit.name}
			unit.handler(&input, resultIO)
			handlerCnt++
		}
	}

	if handlerCnt == 0 {
		resultIO.WriteString("command invalid: " + command + "\n")
	}

	return
}
