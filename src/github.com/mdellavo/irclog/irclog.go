package main

import (
	"log"
	"net"
	"net/textproto"
	"strings"
)

const IRC_HOST = "literat.us:6667"
const IRC_NICK = "gobot"

func loggerMain() chan []string {

	out := make(chan []string, 1000)

	go func(out chan []string) {
		log.Print("Starting logger...")

		addr, err := net.ResolveUDPAddr("udp", ":5222")
		if err != nil {
			log.Fatal(err)
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			log.Fatal(err)
		}

		for {
			buf := [1024]byte{}
			n, remote, err := conn.ReadFrom(buf[0:])
			if err != nil {
				log.Printf("error: %s", err)
				return
			}

			payload := string(buf[:n])
			log.Printf("payload of %d bytes from %s: %s", n, remote, payload)

			ip := strings.SplitN(remote.String(), ":", 2)[0]
			names, err := net.LookupAddr(ip)

			var tag string
			if err != nil {
				tag = ip
			} else {
				tag = names[0]
			}

			out <- []string{tag, payload}
		}

		log.Print("logger finished.")
	}(out)

	return out
}

type IrcConn struct {
	Host string
	Nick string

	Channel string

	Incoming chan string
	Outgoing chan []string

	Conn *textproto.Conn
}

func ircMain(host, nick, channel string) IrcConn {

	ircConn := IrcConn{host, nick, channel, make(chan string, 1000), make(chan []string, 1000), nil}

	writer := func(ircConn IrcConn) {
		log.Print("writer starting...")
		for {
			msg := <-ircConn.Outgoing
			log.Printf("outgoing >>> %s", msg)

			fmt := msg[0]

			old := msg[1:]
			new := make([]interface{}, len(old))
			for i, v := range old {
				new[i] = interface{}(v)
			}

			_, err := ircConn.Conn.Cmd(fmt, new...)
			if err != nil {
				break
			}
		}

		log.Print("writer complete")
	}

	go func(ircConn IrcConn) {
		log.Print("Starting irc...")

		log.Printf("connecting to %s...", host)

		conn, err := textproto.Dial("tcp", host)

		defer ircConn.Conn.Close()

		if err != nil {
			log.Fatal(err)
			return
		}
		ircConn.Conn = conn

		go writer(ircConn)

		ircConn.Cmd("NICK %s", nick)
		ircConn.Cmd("USER %s 0 * :%s", nick, nick)

		for {
			msg, err := ircConn.Conn.ReadLine()
			if err != nil {
				log.Fatal(err)
				return
			}

			ircConn.Incoming <- msg
		}

		log.Print("irc finished.")
	}(ircConn)

	return ircConn
}

func (ircConn *IrcConn) Cmd(msg ...string) {
	ircConn.Outgoing <- msg
}

var IRC_COMMANDS = map[string]func(IrcConn, []string){
	"PING": func(conn IrcConn, params []string) {
		conn.Cmd("PONG %s", params[0])
	},
	"MODE": func(conn IrcConn, params []string) {
		conn.Cmd("JOIN %s", conn.Channel)
	},
	"PRIVMSG": func(conn IrcConn, params []string) {

		if params[1] == "hello" || params[1] == "hi" {
			conn.Cmd("PRIVMSG %s :%s", params[0], params[1])
		}

	},
	"JOIN": func(conn IrcConn, params []string) {
		conn.Cmd("PRIVMSG %s :Hello World.", params[0])
	},
}

func parseLine(s string) (string, string, []string) {

	log.Printf("parsing -> %s", s)

	prefix := ""
	trailing := ""
	args := []string{}
	var cmd string

	if s[0] == ':' {
		tmp := strings.SplitN(s[1:], " ", 2)
		prefix = tmp[0]
		s = tmp[1]
	}

	if strings.Index(s, " :") != -1 {
		tmp := strings.SplitN(s, " :", 2)
		s = tmp[0]
		trailing = tmp[1]
		args = strings.Split(s, " ")
		args = append(args, trailing)
	} else {
		args = strings.Split(s, " ")
	}

	cmd = args[0]
	args = args[1:]

	return prefix, cmd, args
}

func main() {

	loggerChan := loggerMain()
	ircConn := ircMain(IRC_HOST, IRC_NICK, "#gobot")

	for {
		select {

		case line := <-ircConn.Incoming:

			// fixme - move off to another pipeline stage
			prefix, cmd, args := parseLine(line)
			log.Printf("incoming <<< (prefix=%s, cmd=%s, args=%s)", prefix, cmd, args)

			f, ok := IRC_COMMANDS[cmd]
			if ok {
				f(ircConn, args)
			}

			break

		case line := <-loggerChan:
			log.Printf("logger >>> %s", line)
			ircConn.Cmd("PRIVMSG %s :[%s] %s", ircConn.Channel, line[0], line[1])
			break
		}
	}

}
