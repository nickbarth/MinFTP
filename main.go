package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func getTransferConn() *net.TCPConn {
	server, _ := net.Listen("tcp", ":1500")
	defer server.Close()
	listener := server.(*net.TCPListener)
	conn, _ := listener.AcceptTCP()
	return conn
}

func handleConn(conn net.Conn) {
	var transferConn = (*net.TCPConn)(nil)
	var buff = bufio.NewReader(conn)

	fmt.Fprintf(conn, "200 FTP Server ready.\n")
	fmt.Print("Connected.\n")

	for {
		message, err := buff.ReadString('\n')
		command := strings.Split(strings.TrimSpace(message), " ")[0]

		if err != nil {
			break
		}

		fmt.Print(message)

		switch command {
		case "USER":
			fmt.Fprintf(conn, "331 User okay. Please specify the password.\n")
		case "PASS":
			fmt.Fprintf(conn, "230 Login successful.\n")
		case "SYST":
			fmt.Fprintf(conn, "215 UNIX Type: L8\n")
		case "FEAT":
			fmt.Fprintf(conn, "200\n")
		case "CWD":
			fmt.Fprintf(conn, "250 CWD successful.\n")
		case "PWD":
			fmt.Fprintf(conn, "257 \"/\" is the remote directory.\n")
		case "TYPE":
			fmt.Fprintf(conn, "200 Type set to: Binary.\n")
		case "SIZE":
			fmt.Fprintf(conn, "213 28\n")
		case "STOR":
			fmt.Fprintf(conn, "125 Transfer starting.\n")
			func(tc *net.TCPConn) {
				message, _ := bufio.NewReader(tc).ReadString('\n')
				fmt.Printf("DATA:\n%s", message)
				tc.CloseRead()
			}(transferConn)
			transferConn = (*net.TCPConn)(nil)
			fmt.Fprintf(conn, "226 Transfer complete.\n")
		case "RETR":
			fmt.Fprintf(conn, "125 Transfer starting.\n")
			func(tc *net.TCPConn) {
				fmt.Fprintf(tc, "EXAMPLE DATA\r\n")
				tc.CloseWrite()
			}(transferConn)
			transferConn = (*net.TCPConn)(nil)
			fmt.Fprintf(conn, "226 Transfer complete.\n")
		case "LIST":
			fmt.Fprintf(conn, "125 Transfer starting.\n")
			func(tc *net.TCPConn) {
				fmt.Fprintf(tc, "example_file.txt\r\n")
				tc.CloseWrite()
			}(transferConn)
			fmt.Fprintf(conn, "226 Transfer complete.\n")
		case "EPSV":
			fmt.Fprintf(conn, "229 Entering Passive Mode (|||1500|).\n")
			transferConn = getTransferConn()
		case "QUIT":
			fmt.Fprintf(conn, "221\n")
		default:
			fmt.Printf("Not Found `%s`.\n", command)
		}
	}
}

func main() {
	server, _ := net.Listen("tcp", ":2121")
	defer server.Close()

	for {
		conn, _ := server.Accept()
		go handleConn(conn)
	}
}
