package main

import (
	"github.com/ziutek/telnet"
	"log"
	"os"
	"time"
)

const timeout = 10 * time.Second

func checkErr(err error) {
	if err != nil {
		log.Fatalln("Error:", err)
	}
}

func expect(t *telnet.Conn, d ...string) {
	checkErr(t.SetReadDeadline(time.Now().Add(timeout)))
	checkErr(t.SkipUntil(d...))
}

func sendln(t *telnet.Conn, s string) {
	checkErr(t.SetWriteDeadline(time.Now().Add(timeout)))
	buf := make([]byte, len(s)+1)
	copy(buf, s)
	buf[len(s)] = '\n'
	_, err := t.Write(buf)
	checkErr(err)
}

func main() {
	if len(os.Args) != 5 {
		log.Printf("Usage: %s {unix|cisco} HOST:PORT USER PASSWD", os.Args[0])
		return
	}
	typ, dst, user, passwd := os.Args[1], os.Args[2], os.Args[3], os.Args[4]

	t, err := telnet.Dial("tcp", dst)
	checkErr(err)
	t.SetUnixWriteMode(true)

	var data []byte
	switch typ {
	case "unix":
		expect(t, "login: ")
		sendln(t, user)
		expect(t, "ssword: ")
		sendln(t, passwd)
		expect(t, "$")
		sendln(t, "ls -l")
		data, err = t.ReadBytes('$')
	case "cisco":
		expect(t, "name: ")
		sendln(t, user)
		expect(t, "ssword: ")
		sendln(t, passwd)
		expect(t, ">")
		sendln(t, "sh ver")
		data, err = t.ReadBytes('>')
	default:
		log.Fatalln("bad host type: " + typ)
	}
	checkErr(err)
	os.Stdout.Write(data)
	os.Stdout.WriteString("\n")
}
