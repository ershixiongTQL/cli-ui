package router

import (
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"sync"
)

type Input interface {
	GetSegment(int) (string, error)
	SegmentExist(int) bool
	GetRaw() string
	GetUnitName() string
}

type CommandHandler func(Input, io.StringWriter)
type CommandProgressHandler func(input Input, resultIO io.StringWriter, progressUpdate func(ratio float32)) error

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

	handler         CommandHandler
	progressHandler CommandProgressHandler
}

type progressResultIOWrapper struct {
	io       io.StringWriter
	progress *float32
}

func (io *progressResultIOWrapper) WriteString(str string) (n int, err error) {
	io.io.WriteString(fmt.Sprintf("\n%6.2f%%    ", *io.progress*100))
	n, err = io.io.WriteString(str)
	return
}

func (u *Unit) Call(input Input, resultIO io.StringWriter) {

	if u.progressHandler != nil {
		progress := float32(0)
		wrap := &progressResultIOWrapper{io: resultIO, progress: &progress}
		err := u.progressHandler(input, wrap, func(ratio float32) {

			if ratio == progress {
				return
			}

			progress = ratio
			resultIO.WriteString(fmt.Sprintf("\r%6.2f%%    ", progress*100))

			progressInt := int(math.Min(float64(progress*50), float64(50)))

			resultIO.WriteString(strings.Repeat("=", progressInt))
			resultIO.WriteString(strings.Repeat("_", 50-progressInt))
		})

		if err == nil {
			resultIO.WriteString(fmt.Sprintf("\r%6.2f%%    ", float32(100)))
			resultIO.WriteString(strings.Repeat("=", 50))
		}

	} else if u.handler != nil {

		u.handler(input, resultIO)

	}

}

var commandSubscribersNew = make(map[string]Unit)
var subscribersMutex sync.Mutex

func UnitRegister(name string, pattern string, callback interface{}) (err error) {

	subscribersMutex.Lock()
	defer subscribersMutex.Unlock()

	var unit Unit

	if callback == nil {
		return fmt.Errorf("invalid callback function")
	}

	if _, ok := callback.(func(Input, io.StringWriter)); ok {
		unit.handler = callback.(func(Input, io.StringWriter))
	} else if _, ok := callback.(func(Input, io.StringWriter, func(float32)) error); ok {
		unit.progressHandler = callback.(func(Input, io.StringWriter, func(float32)) error)
	} else {
		return fmt.Errorf("invalid callback type")
	}

	unit.compiled = regexp.MustCompile(pattern)

	unit.name = name
	unit.pattern = pattern

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
			unit.Call(
				&CmdInput{raw: command, subMatches: found[1:], unitName: unit.name},
				resultIO,
			)
			handlerCnt++
		}
	}

	if handlerCnt == 0 {
		resultIO.WriteString("command invalid: " + command + "\n")
		err = fmt.Errorf("mux nothing")
	}

	return
}
