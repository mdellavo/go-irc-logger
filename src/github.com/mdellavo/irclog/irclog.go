package main

import (
	"log"
	"net"
	"flag"
	"time"
	"strconv"
	"net/textproto"
	"strings"
)

var Port = flag.String("p", "5222", "listen port")
var Host = flag.String("h", "", "irc host")
var Nick = flag.String("n", "", "irc nick")
var Channel = flag.String("c", "", "irc channel")

func now() (int) {
	return int(time.Now().Unix())
}

func getRemote(remote net.Addr) (ip, name string) {

	ip = strings.SplitN(remote.String(), ":", 2)[0]
	names, err := net.LookupAddr(ip)

	if err != nil {
		return ip, ""
	}

	return ip, names[0]

}

func getRemoteTag(remote net.Addr) (tag string) {
	ip, name := getRemote(remote)

	if name != "" {
		return name
	}

	return ip
}

func udpLoggerMain() chan []string {

	out := make(chan []string, 1000)

	reader := func(out chan []string) {
		log.Print("Starting logger...")

		addr, err := net.ResolveUDPAddr("udp", ":" + *Port)
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

			go func() {

				payload := string(buf[:n])
				log.Printf("payload of %d bytes from %s: %s", n, remote, payload)

				out <- []string{getRemoteTag(remote), payload}

			}()
		}

		log.Print("logger finished.")
	}

	go reader(out)

	return out
}

func tcpLoggerMain() chan []string {
	out := make(chan []string, 1000)

	reader := func(c net.Conn) {
		defer c.Close()
		stream := textproto.NewConn(c)
		tag := getRemoteTag(c.RemoteAddr())
		for {

			line, err := stream.ReadLine()

			if err != nil {
				log.Print("closing reader")
				break
			}


			out <- []string{tag, line}
		}
	}

	listener := func(out chan []string) {
		l, err := net.Listen("tcp", ":" + *Port)
		if err != nil {
			log.Fatal(err)
		}
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}
			go reader(conn)
		}
	}

	go listener(out)

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

	reader := func(ircConn IrcConn) {
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
	}

	go reader(ircConn)

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
		} else if params[1] == "ping" {
			conn.Cmd("PRIVMSG %s CTCP PING %s %s", *Channel, strconv.Itoa(now()), "1")
		}

	},
	"JOIN": func(conn IrcConn, params []string) {
		conn.Cmd("PRIVMSG %s :Hello World.", params[0])
	},
}

func parseLine(s string) (string, string, []string) {
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

	flag.Parse()

	udpLoggerChan := udpLoggerMain()
	tcpLoggerChan := tcpLoggerMain()
	ircConn := ircMain(*Host, *Nick, *Channel)

	echo := func(msg []string) {
		ircConn.Cmd("PRIVMSG %s :[%s] %s", ircConn.Channel, msg[0], msg[1])
	}

	for {
		select {

		case msg := <-ircConn.Incoming:

			// FIXME - move off to another pipeline stage
			prefix, cmd, args := parseLine(msg)
			log.Printf("incoming <<< (prefix=%s, cmd=%s, args=%s)", prefix, cmd, args)

			f, ok := IRC_COMMANDS[cmd]
			if ok {
				f(ircConn, args)
			}

			break

		case msg := <- tcpLoggerChan:
			log.Printf("tcp logger >>> %s", msg)
			echo(msg)
			break

		case msg := <- udpLoggerChan:
			log.Printf("udp logger >>> %s", msg)
			echo(msg)
			break
		}
	}

}
