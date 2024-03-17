package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

func main() {
	var (
		host  = flag.String("host", "localhost", "Host to connect to")
		port  = flag.String("port", "8080", "Port to connect to")
		mode  = flag.String("mode", "client", "Mode to run in client/server")
		shell = flag.Bool("shell", false, "Enable a reverse shell")
	)
	flag.Parse()

	switch *mode {
	case "client":
		runClient(*host, *port)
	case "server":
		runServer(*port, *shell)
	default:
		fmt.Println("Invalid mode")
		os.Exit(1)
	}
}

func runClient(host, port string) {
	fmt.Println("Connecting to", host, "on port", port)
	conn, err := net.Dial("tcp", net.JoinHostPort(host, port))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to %s:%s, %v\n", host, port, err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Connected to", host, "on port", port)
	go io.Copy(os.Stdout, conn)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprintln(conn, scanner.Text())
	}

	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", scanner.Err())
	}

}

func runServer(port string, isShell bool) {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server on port %s: %v\n", port, err)
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Starting server on port", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
			os.Exit(1)
		}

		if isShell {
			go handleShell(conn)
		} else {
			go handleConn(conn)
		}
	}
}

func handleShell(conn net.Conn) {
	defer conn.Close()

	// Start a new shell with a PTY
	c := exec.Command("/bin/bash")
	ptmx, err := pty.Start(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start shell: %v\n", err)
		return
	}
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Set the PTY size to standard values, adjust as needed
	pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80, X: 0, Y: 0})

	// Copy input and output between the PTY and the connection
	go func() { io.Copy(ptmx, conn) }()
	io.Copy(conn, ptmx)
}

func handleConn(conn net.Conn) {
	io.Copy(conn, conn)
	conn.Close()
}
