package main

import (
	"log"
	"net"
	"os"
	"bufio"
	"strings"
)

func main() {

	conn, err := net.Dial("udp", "localhost:5222")
	defer conn.Close()
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) == 1 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			conn.Write([]byte(scanner.Text()))
		}
	} else {
		conn.Write([]byte(strings.Join(os.Args[1:], " ")))
	}

}
