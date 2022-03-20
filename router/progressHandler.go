package router

import (
	"fmt"
	"io"
	"math"
	"strings"
)

const PROGRESS_BAR_LENGTH int = 36
const PROGRESS_CHAR_DONE string = "="
const PROGRESS_CHAR_TODO string = "_"

type ProgressHandler func(input Input, resultIO io.StringWriter, progressUpdate func(ratio float32)) error

func UnitRegisterProgress(name string, pattern string, callback ProgressHandler) (err error) {
	registered, err := unitRegister(name, pattern)
	if err != nil {
		return
	}
	registered.progressHandler = callback
	return
}

type progressResultIOWrapper struct {
	io       io.StringWriter
	progress *float32
}

func (io *progressResultIOWrapper) WriteString(str string) (n int, err error) {

	if *io.progress >= 1 {

		n, err = io.io.WriteString(str)
		return

	}

	str = strings.TrimSpace(str)

	if strings.ContainsAny(str, "\n\r") {

		lines := strings.Split(str, "\n")

		for _, l := range lines {
			l = strings.TrimSpace(l)
			io.io.WriteString(fmt.Sprintf("    %s\n", l))
			printProgressBar(io.io, *io.progress)
		}

	} else {
		printProgressBar(io.io, *io.progress)
		io.io.WriteString(fmt.Sprintf("    %s\n", str))
		printProgressBar(io.io, *io.progress)
	}

	return
}

func progressHandlerCall(unit *unit, input Input, resultIO io.StringWriter) {

	if unit == nil {
		return
	}

	handler := unit.progressHandler

	if handler == nil {
		return
	}

	progress := float32(-1)
	wrap := &progressResultIOWrapper{io: resultIO, progress: &progress}

	err := handler(input, wrap, func(ratio float32) {
		if ratio < 0 || ratio == progress {
			return
		}
		progress = ratio
		printProgressBar(resultIO, ratio)
	})

	if err == nil {
		progress = 1
		// printProgressBar(resultIO, 1)
	}
}

func printProgressBar(writer io.StringWriter, ratio float32) {

	if ratio > 1 {
		ratio = 1
	}

	progressChars := int(math.Min(float64(ratio*float32(PROGRESS_BAR_LENGTH)), float64(PROGRESS_BAR_LENGTH)))
	writer.WriteString(fmt.Sprintf("\r%6.2f%%    [%s%s]", ratio*100,
		strings.Repeat(PROGRESS_CHAR_DONE, progressChars),
		strings.Repeat(PROGRESS_CHAR_TODO, PROGRESS_BAR_LENGTH-progressChars)))

	if ratio >= 1 {
		writer.WriteString("\n")
	}
}
