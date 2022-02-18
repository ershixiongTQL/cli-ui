// Package telnet provides simple interface for interacting with Telnet
// connection.
package protocol

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"time"
	"unicode"
)

const (
	CR = byte('\r')
	LF = byte('\n')
)

const (
	cmdSE   = 240
	cmdNOP  = 241
	cmdData = 242

	cmdBreak = 243
	cmdEl    = 248
	cmdGA    = 249
	cmdSB    = 250

	cmdWill = 251
	cmdWont = 252
	cmdDo   = 253
	cmdDont = 254

	cmdIAC = 255
)

const (
	OptEcho            = 1
	OptSuppressGoAhead = 3
	OptTerminalType    = 24
	OptLineMode        = 34
	OptNAWS            = 31
	OptNAOFFD          = 13
)

type Conn struct {
	net.Conn
	r *bufio.Reader

	unixWriteMode bool

	cliSuppressGoAhead bool
	cliEcho            bool
	cliLineMode        bool
}

func NewConn(conn net.Conn) (*Conn, error) {
	c := Conn{
		Conn: conn,
		r:    bufio.NewReaderSize(conn, 256),
	}
	return &c, nil
}

func Dial(network, addr string) (*Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return NewConn(conn)
}

func DialTimeout(network, addr string, timeout time.Duration) (*Conn, error) {
	conn, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}
	return NewConn(conn)
}

// SetUnixWriteMode sets flag that applies only to the Write method.
// If set, Write converts any '\n' (LF) to '\r\n' (CR LF).
func (c *Conn) SetUnixWriteMode(uwm bool) {
	c.unixWriteMode = uwm
}

func (c *Conn) do(option byte) error {
	_, err := c.Conn.Write([]byte{cmdIAC, cmdDo, option})
	return err
}

func (c *Conn) dont(option byte) error {
	_, err := c.Conn.Write([]byte{cmdIAC, cmdDont, option})
	return err
}

func (c *Conn) Will(option byte) error {
	_, err := c.Conn.Write([]byte{cmdIAC, cmdWill, option})
	return err
}

func (c *Conn) wont(option byte) error {
	_, err := c.Conn.Write([]byte{cmdIAC, cmdWont, option})
	return err
}

func (c *Conn) sub(opt byte, data ...byte) error {
	if _, err := c.Conn.Write([]byte{cmdIAC, cmdSB, opt}); err != nil {
		return err
	}
	if _, err := c.Conn.Write(data); err != nil {
		return err
	}
	_, err := c.Conn.Write([]byte{cmdIAC, cmdSE})
	return err
}

func (c *Conn) deny(cmd, opt byte) (err error) {
	switch cmd {
	case cmdDo:
		err = c.wont(opt)
	case cmdDont:
		// nop
	case cmdWill, cmdWont:
		err = c.dont(opt)
	}
	return
}

func (c *Conn) skipSubneg() error {
	for {
		if b, err := c.r.ReadByte(); err != nil {
			return err
		} else if b == cmdIAC {
			if b, err = c.r.ReadByte(); err != nil {
				return err
			} else if b == cmdSE {
				return nil
			}
		}
	}
}

func (c *Conn) cmd(cmd byte) error {

	switch cmd {
	case cmdGA:
		return nil
	case cmdDo, cmdDont, cmdWill, cmdWont:
		// Process cmd after this switch.
	case cmdSB:
		return c.skipSubneg()
	case cmdEl:
		c.Conn.Write([]byte{cmdEl})
		return nil
	case cmdNOP:
		return nil
	default:
		return fmt.Errorf("unknown command: %d", cmd)
	}
	// Read an option
	o, err := c.r.ReadByte()
	if err != nil {
		return err
	}

	switch o {
	case OptEcho:
		// Accept any echo configuration.
		switch cmd {
		case cmdDo:
			if !c.cliEcho {
				c.cliEcho = true
				err = c.Will(o)
			}
		case cmdDont:
			if c.cliEcho {
				c.cliEcho = false
				err = c.wont(o)
			}
		case cmdWill:
			if !c.cliEcho {
				c.cliEcho = true
				err = c.do(o)
			}
		case cmdWont:
			if c.cliEcho {
				c.cliEcho = false
				err = c.dont(o)
			}
		}
	case OptSuppressGoAhead:
		// We don't use GA so can allways accept every configuration
		switch cmd {
		case cmdDo:
			if !c.cliSuppressGoAhead {
				c.cliSuppressGoAhead = true
				err = c.Will(o)
			}
		case cmdDont:
			if c.cliSuppressGoAhead {
				c.cliSuppressGoAhead = false
				err = c.wont(o)
			}
		case cmdWill:
			if !c.cliSuppressGoAhead {
				c.cliSuppressGoAhead = true
				err = c.do(o)
			}
		case cmdWont:
			if c.cliSuppressGoAhead {
				c.cliSuppressGoAhead = false
				err = c.dont(o)
			}
		}
	case OptNAWS:
		if cmd != cmdDo {
			err = c.deny(cmd, o)
			break
		}
		if err = c.Will(o); err != nil {
			break
		}
		// Reply with max window size: 65535x65535
		err = c.sub(o, 255, 255, 255, 255)
	case OptLineMode:
		switch cmd {
		case cmdDo:
			if !c.cliLineMode {
				c.cliLineMode = true
				err = c.Will(o)
			}
		case cmdDont:
			if c.cliLineMode {
				c.cliLineMode = false
				err = c.wont(o)
			}
		case cmdWill:
			if !c.cliLineMode {
				c.cliLineMode = true
				err = c.do(o)
			}
		case cmdWont:
			if c.cliLineMode {
				c.cliLineMode = false
				err = c.dont(o)
			}
		}
	default:
		// Deny any other option
		err = c.deny(cmd, o)
	}
	return err
}

func (c *Conn) tryReadByte() (b byte, retry bool, err error) {
	b, err = c.r.ReadByte()
	if err != nil || b != cmdIAC {
		return
	}
	b, err = c.r.ReadByte()
	if err != nil {
		return
	}
	if b != cmdIAC {
		err = c.cmd(b)
		if err != nil {
			fmt.Printf("telnet cmd error: %s\n", err.Error())
			return
		}
		retry = true
	}
	return
}

func (c *Conn) ClearScreen() {
	c.Write([]byte{12})
}

// SetEcho tries to enable/disable echo on server side. Typically telnet
// servers doesn't support this.
func (c *Conn) SetEcho(echo bool) error {
	if echo {
		return c.do(OptEcho)
	}
	return c.dont(OptEcho)
}

// ReadByte works like bufio.ReadByte
func (c *Conn) ReadByte() (b byte, err error) {
	retry := true
	for retry && err == nil {
		b, retry, err = c.tryReadByte()
	}
	return
}

// ReadRune works like bufio.ReadRune
func (c *Conn) ReadRune() (r rune, size int, err error) {
loop:
	r, size, err = c.r.ReadRune()
	if err != nil {
		return
	}
	if r != unicode.ReplacementChar || size != 1 {
		// Properly readed rune
		return
	}
	// Bad rune
	err = c.r.UnreadRune()
	if err != nil {
		return
	}
	// Read telnet command or escaped IAC
	_, retry, err := c.tryReadByte()
	if err != nil {
		return
	}
	if retry {
		// This bad rune was a begining of telnet command. Try read next rune.
		goto loop
	}
	// Return escaped IAC as unicode.ReplacementChar
	return
}

// Read is for implement an io.Reader interface
func (c *Conn) Read(buf []byte) (int, error) {
	var n int
	for n < len(buf) {
		b, retry, err := c.tryReadByte()
		if err != nil {
			return n, err
		}
		if !retry {
			buf[n] = b
			n++
		}
		if n > 0 && c.r.Buffered() == 0 {
			// Don't block if can't return more data.
			return n, err
		}
	}
	return n, nil
}

// ReadBytes works like bufio.ReadBytes
func (c *Conn) ReadBytes(delim byte) ([]byte, error) {
	var line []byte
	for {
		b, err := c.ReadByte()
		if err != nil {
			return nil, err
		}
		line = append(line, b)
		if b == delim {
			break
		}
	}
	return line, nil
}

// SkipBytes works like ReadBytes but skips all read data.
func (c *Conn) SkipBytes(delim byte) error {
	for {
		b, err := c.ReadByte()
		if err != nil {
			return err
		}
		if b == delim {
			break
		}
	}
	return nil
}

// ReadString works like bufio.ReadString
func (c *Conn) ReadString(delim byte) (string, error) {
	bytes, err := c.ReadBytes(delim)
	return string(bytes), err
}

func (c *Conn) readUntil(read bool, delims ...string) ([]byte, int, error) {
	if len(delims) == 0 {
		return nil, 0, nil
	}
	p := make([]string, len(delims))
	for i, s := range delims {
		if len(s) == 0 {
			return nil, 0, nil
		}
		p[i] = s
	}
	var line []byte
	for {
		b, err := c.ReadByte()
		if err != nil {
			return nil, 0, err
		}
		if read {
			line = append(line, b)
		}
		for i, s := range p {
			if s[0] == b {
				if len(s) == 1 {
					return line, i, nil
				}
				p[i] = s[1:]
			} else {
				p[i] = delims[i]
			}
		}
	}
}

// ReadUntilIndex reads from connection until one of delimiters occurs. Returns
// read data and an index of delimiter or error.
func (c *Conn) ReadUntilIndex(delims ...string) ([]byte, int, error) {
	return c.readUntil(true, delims...)
}

// ReadUntil works like ReadUntilIndex but don't return a delimiter index.
func (c *Conn) ReadUntil(delims ...string) ([]byte, error) {
	d, _, err := c.readUntil(true, delims...)
	return d, err
}

// SkipUntilIndex works like ReadUntilIndex but skips all read data.
func (c *Conn) SkipUntilIndex(delims ...string) (int, error) {
	_, i, err := c.readUntil(false, delims...)
	return i, err
}

// SkipUntil works like ReadUntil but skips all read data.
func (c *Conn) SkipUntil(delims ...string) error {
	_, _, err := c.readUntil(false, delims...)
	return err
}

// Write is for implement an io.Writer interface
func (c *Conn) Write(buf []byte) (int, error) {
	search := "\xff"
	if c.unixWriteMode {
		search = "\xff\n"
	}
	var (
		n   int
		err error
	)
	for len(buf) > 0 {
		var k int
		i := bytes.IndexAny(buf, search)
		if i == -1 {
			k, err = c.Conn.Write(buf)
			n += k
			break
		}
		k, err = c.Conn.Write(buf[:i])
		n += k
		if err != nil {
			break
		}
		switch buf[i] {
		case LF:
			k, err = c.Conn.Write([]byte{CR, LF})
		case cmdIAC:
			k, err = c.Conn.Write([]byte{cmdIAC, cmdIAC})
		}
		n += k
		if err != nil {
			break
		}
		buf = buf[i+1:]
	}
	return n, err
}
