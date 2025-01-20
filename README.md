# zmux

### usage
```go
func serve(l net.Listener) {
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("failed to accept tcp connection: %s\n", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	mux := zmux.New(conn)
	for {
		ch, err := mux.Accept()
		if err != nil {
			log.Printf("failed to accept channel: %s\n", err)
			continue
		}
		go handleChannel(ch)
	}
}

func handleChannel(ch *zmux.Channel) {
	for {
		b := make([]byte, 100)
		n, err := ch.Read(b)
		if err != nil {
			log.Printf("failed read frame: %s %d\n", err, n)
		}
		fmt.Println("message from client:", string(b[:n]))
	}
}

func main() {
	l, err := net.Listen("tcp", ":4000")
	if err != nil {
		fmt.Printf("failed to listen port: %s\n", err)
		return
	}
	go serve(l)

	conn, err := net.Dial("tcp", "localhost:4000")
	if err != nil {
		fmt.Printf("dial failed: %s\n", err)
		return
	}

	mux := zmux.New(conn)

	ch, err := mux.Open()
	if err != nil {
		fmt.Printf("failed to open channel: %s", err)
		return
	}

	for {
		n, err := ch.Write([]byte("Hello server!"))
		if err != nil {
			fmt.Printf("failed to write data: %s %d", err, n)
			return
		}
		time.Sleep(time.Second)
	}
}

```