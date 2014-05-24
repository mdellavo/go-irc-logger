package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"flag"
	"strings"
)

func write(conn net.Conn, s string) {
	s = strings.Replace(s, "\t", "        ", -1)
	conn.Write([]byte(s))
}

var host = flag.String("h", "localhost", "target log host")
var port = flag.String("p", "5222", "log port")

func main() {
	flag.Parse()

	conn, err := net.Dial("udp", *host + ":" + *port)
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
