package frontendtelnet

import (
	"fmt"
	"net"

	"github.com/ershixiongTQL/cli-ui/frontendtelnet/protocol"

	"github.com/ershixiongTQL/cli-ui/interfaces"
)

//特殊字符定义
const (
	NULL   uint8 = 0x00
	CR     uint8 = protocol.CR //'\r'
	LF     uint8 = protocol.LF //'\n'
	TAB    uint8 = '\t'
	BS     uint8 = '\b'
	DEL    uint8 = 0x7f
	ETX    uint8 = 0x03
	EOT    uint8 = 0x04
	SUB    uint8 = 0x1a
	ESC    uint8 = 0x1b
	QM     uint8 = 0x3f
	CTRL_A uint8 = 'A' - '@'
	CTRL_E uint8 = 'E' - '@'
	CTRL_U uint8 = 'U' - '@'
)

var triggerMap = map[uint8]uint8{
	CR:     CR,
	TAB:    TAB,
	BS:     BS,
	DEL:    DEL,
	ETX:    ETX,
	EOT:    EOT,
	SUB:    SUB,
	ESC:    ESC,
	QM:     QM,
	CTRL_A: CTRL_A,
	CTRL_E: CTRL_E,
	CTRL_U: CTRL_U,
	NULL:   NULL,
}

func triggerCheck(char uint8) (isTrigger bool) {
	_, isTrigger = triggerMap[char]
	return
}

type Config struct {
	GetPrompt func() string
	GetBanner func() string
	Backend   interfaces.BackEndInterface
	ListenOn  string
}

type Server struct {
	config   Config
	listener net.Listener
}

func (s *Server) Init(cfg Config) error {
	s.config = cfg
	return nil
}

func (s *Server) Start() (err error) {

	if s.listener != nil {
		return fmt.Errorf("server already started")
	}

	s.listener, err = net.Listen("tcp", s.config.ListenOn)
	if err != nil {
		return fmt.Errorf("unable to listen on %s, %s", s.config.ListenOn, err.Error())
	}

	go serverRoutine(s)

	return
}

func (s *Server) Stop() {
	s.listener.Close()
	s.listener = nil
}

func serverRoutine(s *Server) {
	for {

		if listener := s.listener; listener != nil {
			connRaw, err := listener.Accept()
			if err != nil {
				listener.Close()
				return
			}

			telnetConn, err := protocol.NewConn(connRaw)
			if err != nil {
				continue
			}

			//TODO: new connection hook

			go telnetConnRoutine(telnetConn, s)
		} else {
			return
		}

	}
}

func telnetConnRoutine(conn *protocol.Conn, s *Server) {

	client := newClient(s, conn)

	client.negotiate()
	client.print(s.config.GetBanner())
	client.printPrompt()

	for {

		char, err := conn.ReadByte()

		if err != nil {
			//TODO: detach hook
			conn.Close()
			return
		}

		if !triggerCheck(char) {
			if char == LF {
				continue
			}
			client.lineAppend([]byte{char})
			continue
		}

		switch char {
		case CR:
			client.print("\n")
			if client.getLineLen() > 0 {
				client.exec()
			}
			client.printPrompt()
		case TAB:
			client.getCompletions()
		case QM:
			client.getHelp()
		case BS:
			fallthrough
		case DEL:
			client.inLineBs()
		case ESC:

			var err error
			var next uint8

			next, err = conn.ReadByte()
			if err != nil || next == ESC {
				client.close()
				return
			}

			switch next {
			case ESC:
				client.close()
				return

			case 0x5b:

				next, err = conn.ReadByte()
				if err != nil {
					client.close()
					return
				}

				switch next {
				case 'A':
					client.historyCheckout(true)
				case 'B':
					client.historyCheckout(false)
				case 'C':
					client.cursorMove(1) //right
				case 'D':
					client.cursorMove(-1) //left
				case '1': //home
					next, err = conn.ReadByte()
					if err != nil {
						client.close()
						return
					}

					//fmt.Printf("escape 0x5b: 0x31: 0x%x\n", next)

					switch next {
					case 0x7e: //home
						client.cursorHome()
					}

				case '4': //end
					next, err = conn.ReadByte()
					if err != nil {
						client.close()
						return
					}

					//fmt.Printf("escape 0x5b: 0x34: 0x%x\n", next)

					switch next {
					case 0x7e: //home
						client.cursorEnd()
					}

				case '3':
					next, err = conn.ReadByte()
					if err != nil {
						client.close()
						return
					}

					//fmt.Printf("escape 0x5b: 0x33: 0x%x\n", next)

					switch next {
					case 0x7e: //del ahead
						client.inlineDel()
					}
				default:

				}

			default:
				fmt.Printf("unknown ESC next: %d(0x%x)\n", next, next)
				client.close()
				return
			}

		case ETX:
			fallthrough
		case EOT:
			fallthrough
		case SUB:
			client.close()
			return
		case CTRL_A:
			client.cursorHome()
		case CTRL_E:
			client.cursorEnd()
		case CTRL_U:
			client.allLineClear()
		case NULL:
			continue
		default:
			continue
		}

	}
}
