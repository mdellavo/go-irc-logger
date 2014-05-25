package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"flag"
	"strings"
)

var mode = flag.String("m", "tcp", "udp/tcp mode")
var host = flag.String("h", "localhost", "target log host")
var port = flag.String("p", "5222", "log port")

func write(conn net.Conn, s string) {
	s = strings.Replace(s, "\t", "        ", -1) + "\r\n"
	conn.Write([]byte(s))
}

func dialTcp() (conn net.Conn, err error) {

	addr, err := net.ResolveTCPAddr("tcp", *host + ":" + *port)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	tcpConn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	tcpConn.SetNoDelay(true)
	tcpConn.SetLinger(-1)

	return tcpConn, err
}

func dialUdp() (conn net.Conn, err error) {
	return net.Dial(*mode, *host + ":" + *port)
}

func main() {
	flag.Parse()

	var conn net.Conn
	var err error
	if *mode == "tcp" {
		conn, err = dialTcp()
	} else {
		conn, err = dialUdp()
	}

	defer conn.Close()
	if err != nil {
		log.Fatal(err)
	}

	args := flag.Args()

	if len(args) > 0 && args[0] == "-" {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			write(conn, scanner.Text())
		}
	} else {
		write(conn, strings.Join(args, " "))
	}
}
