package frontendtelnet

import (
	"bytes"
	"strings"

	"github.com/ershixiongTQL/cli-ui/frontendtelnet/protocol"
	"github.com/ershixiongTQL/cli-ui/history"
)

type client struct {
	server       *Server
	conn         *protocol.Conn
	inLineBuffer *bytes.Buffer
	lineCursor   int
	history      *history.HRing
}

func newClient(s *Server, conn *protocol.Conn) (c *client) {
	c = new(client)
	c.server = s
	c.conn = conn
	c.inLineBuffer = bytes.NewBuffer(make([]byte, 0, 2048))
	c.history = history.NewHRing(1024)
	return
}

func (c *client) WriteString(str string) (n int, err error) {
	c.conn.Write([]byte(str))
	return len(str), nil
}

func (c *client) negotiate() {
	c.conn.ClearScreen()
	c.conn.SetUnixWriteMode(true)
	c.conn.Will(protocol.OptSuppressGoAhead)
	c.conn.Will(protocol.OptEcho)
}

func (c *client) print(text string) {
	c.conn.Write([]byte(text))
}

func (c *client) printPrompt() {
	c.conn.Write([]byte(c.server.config.GetPrompt()))
	c.conn.Write([]byte("# "))
}

func (c *client) close() {
	c.conn.Close()
}

func (c *client) clientCursorBack(num int) {
	c.print(strings.Repeat("\b", num))
}

//unsafe
func (c *client) clientCursorForward(num int) {
	c.conn.Write(c.inLineBuffer.Bytes()[c.lineCursor : c.lineCursor+num])
}

func (c *client) cursorMoveCheck(move int) (moved int) {

	dst := c.lineCursor + move
	if dst < 0 || dst > c.getLineLen() {
		return 0
	}
	return move
}

func (c *client) cursorMove(move int) (ok bool) {

	if c.cursorMoveCheck(move) != move {
		return false
	}

	if move < 0 {
		c.clientCursorBack(-move)
	} else if move > 0 {
		c.clientCursorForward(move)
	}

	c.lineCursor += move
	return true
}

func (c *client) cursorHome() {
	c.cursorMove(-c.lineCursor)
}

func (c *client) cursorEnd() {
	c.cursorMove(c.inLineBuffer.Len() - c.lineCursor)
}

func (c *client) inLineBs() {

	if c.cursorMoveCheck(-1) == -1 {

		if !c.isCursorAtTheEnd() {
			c.lineCursor -= 1
			endPos := c.lineCursor
			goback := c.inLineBuffer.Len() - endPos - 1

			newLine := append(c.inLineBuffer.Bytes()[:endPos], c.inLineBuffer.Bytes()[endPos+1:]...)
			c.inLineBuffer.Reset()
			c.inLineBuffer.Write(newLine)
			c.print("\b" + string(newLine[endPos:]) + " \b")
			c.clientCursorBack(goback)
		} else {
			c.print("\b \b") //wipe display
			line := c.inLineBuffer.Bytes()
			lineLen := c.inLineBuffer.Len()
			c.inLineBuffer.Reset()
			c.inLineBuffer.Write(line[:lineLen-1])
			c.lineCursor -= 1
		}

	}
}

func (c *client) inlineDel() {
	if c.cursorMoveCheck(1) == 1 {
		if c.lineCursor == c.inLineBuffer.Len()-1 {
			newLine := c.inLineBuffer.Bytes()[:c.lineCursor]
			c.inLineBuffer.Reset()
			c.inLineBuffer.Write(newLine)
			c.print(" \b")
		} else {
			newLine := append(c.inLineBuffer.Bytes()[:c.lineCursor], c.inLineBuffer.Bytes()[c.lineCursor+1:]...)
			c.inLineBuffer.Reset()
			c.inLineBuffer.Write(newLine)
			c.print(string(newLine[c.lineCursor:]) + " ")
			c.clientCursorBack(len(newLine[c.lineCursor:]) + 1)
		}
	}
}

func (c *client) exec() error {
	line := c.getLine()
	c.inLineClear()

	re := regexp.MustCompile(`^\s*(exit|quit)\s*$`)

	if re.FindString(line) != "" {
		return errors.New("exit")
	}

	if handler := c.server.config.Backend.CommandHandler; handler != nil {
		handler(line, c)
	}
	c.WriteString("\n")
	c.history.Append(line)

	return nil
}

func (c *client) getCompletions() {

	if !c.isCursorAtTheEnd() {
		return
	}

	var completions []string

	if completer := c.server.config.Backend.Completer; completer != nil {
		completions = completer(c.getLine())
	}

	if len(completions) == 1 {
		//single option, simply print
		c.lineAppend([]byte(completions[0]))
	} else if len(completions) > 1 {

		c.print("\n")
		c.print(strings.Join(completions, " "))
		c.print("\n")
		c.printPrompt()
		c.print(c.getLine())
	} else {
		//no completion
		//TODO: <ENTER> hint
		return
	}
}

func (c *client) getHelp() {

	if helper := c.server.config.Backend.Helps; helper != nil {
		help := helper(c.getLine())

		if help != "" {
			c.print("\n" + help + "\n")
			c.printPrompt()
			c.print(c.getLine())
		}
	}
}

func (c *client) isCursorAtTheEnd() bool {
	return c.lineCursor >= c.inLineBuffer.Len()
}

func (c *client) lineAppend(chars []byte) {

	if c.isCursorAtTheEnd() {
		//Append

		c.inLineBuffer.Write(chars)
		c.lineCursor += len(chars)
		c.print(string(chars))

	} else {
		//Insert

		startPos := c.lineCursor

		seg2 := append([]byte{}, c.inLineBuffer.Bytes()[startPos:]...)
		newLine := append(c.inLineBuffer.Bytes()[:startPos], chars...)
		newLine = append(newLine, seg2...)

		c.inLineBuffer.Reset()
		c.inLineBuffer.Write(newLine)

		c.print(string(chars))
		c.print(string(newLine[startPos+len(chars):]))
		c.clientCursorBack(len(newLine[startPos+len(chars):]))
		c.lineCursor += len(chars)
	}
}

func (c *client) allLineClear() {
	for {
		if c.lineCursor > 0 {
			c.print("\b \b") //wipe display
			c.lineCursor -= 1
		} else {
			c.inLineBuffer.Reset()
			break
		}
	}
}

func (c *client) inLineClear() {
	c.inLineBuffer.Reset()
	c.lineCursor = 0
}

func (c *client) getLine() string {
	return c.inLineBuffer.String()
}

func (c *client) getLineLen() int {
	len := c.inLineBuffer.Len()
	return len
}

func (c *client) historyCheckout(previous bool) {

	hisCnt := c.history.Cnt()
	if hisCnt == 0 {
		return
	}

	var ok bool
	if previous {
		ok = c.history.PosBack()
	} else {
		ok = c.history.PosForward()
	}

	currLineLen := c.getLineLen()
	if currLineLen != 0 {
		for i := 0; i < currLineLen; i++ {
			c.conn.Write([]byte("\b \b"))
		}
	}

	c.inLineClear()

	if ok {
		content := []byte(c.history.Read())
		c.lineAppend(content)
	}

}
