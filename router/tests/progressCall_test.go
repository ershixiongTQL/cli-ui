package router

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/ershixiongTQL/cli-ui/router"
)

type testBuf struct {
	writeTest func(str string, self *testBuf)
}

func (b *testBuf) WriteString(str string) (l int, err error) {
	l = len(str)
	b.writeTest(str, b)
	return
}

func progressCallSim(uniqName string, progressGet func() (float32, error)) (err error) {

	router.UnitRegister(uniqName, uniqName, func(input router.Input, result io.StringWriter, progressUpdate func(float32)) error {

		for {
			p, e := progressGet()
			if e != nil {
				if strings.Contains(e.Error(), "stop") {
					return nil
				} else {
					return e
				}
			}
			progressUpdate(p)
		}
	})

	var testError error

	buf := testBuf{
		writeTest: func(str string, self *testBuf) {

			if testError != nil {
				//some error happened
				return
			}

			//TEST: format error
			if strings.Contains(str, "!(NOVERB)") {
				testError = fmt.Errorf("format error: NOVERB")
			}

			//TEST: invalid charactor
			founds := regexp.MustCompile(`[^\d\.%_= \n\r]`).FindAllString(str, -1)

			if len(founds) != 0 {
				testError = fmt.Errorf("invalid charactor found: %s", founds[:])
			}

			//TODO: other tests

			return
		},
	}

	router.Mux(uniqName, &buf)

	return testError
}

func TestNormal(t *testing.T) {

	progress := float32(0)

	err := progressCallSim("testNormal", func() (p float32, err error) {
		p = progress
		if progress >= 1 {
			return 0, fmt.Errorf("stop")
		}
		progress += 0.1
		return
	})

	if err != nil {
		t.Fatal(err)
	}
}
