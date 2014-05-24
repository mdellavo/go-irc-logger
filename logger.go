package main

import (
	"log"
	"net"
	"os"
	"bufio"
	"strings"
)

func write (conn net.Conn, s string) {
	s = strings.Replace(s, "\t", "        ", -1)
	conn.Write([]byte(s))
}

func main() {

	conn, err := net.Dial("udp", "localhost:5222")
	defer conn.Close()
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 && os.Args[1] == "-" {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			write(conn, scanner.Text())
		}
	} else {
		write(conn, strings.Join(os.Args[1:], " "))
	}

}
