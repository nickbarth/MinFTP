package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
)

func getTransferConn() *net.TCPConn {
	server, _ := net.Listen("tcp", ":1500")
	defer server.Close()
	listener := server.(*net.TCPListener)
	conn, _ := listener.AcceptTCP()
	return conn
}

func validLogin(user string, password string) bool {
	return user == "admin" && password == "password"
}

func authRequired(command string) bool {
	return command == "DELE" || command == "STOR" || command == "SIZE" || command == "LIST" || command == "RETR"
}

func handleConn(conn net.Conn) {
	user := "anonymous"
	password := "anonymous"

	transferConn := (*net.TCPConn)(nil)
	buff := bufio.NewReader(conn)

	fmt.Fprintf(conn, "200 FTP Server ready.\n")
	fmt.Print("Connected.\n")

	for {
		message, err := buff.ReadString('\n')
		command := strings.Split(strings.TrimSpace(message), " ")[0]

		if err != nil {
			break
		}

		if !validLogin(user, password) && authRequired(command) {
			fmt.Fprintf(conn, "550 Not authorized.\n")
			continue
		}

		fmt.Print(message)

		switch command {
		case "USER":
			user = strings.Split(strings.TrimSpace(message), " ")[1]
			fmt.Fprintf(conn, "331 User okay. Please specify the password.\n")
		case "PASS":
			arg := strings.Split(strings.TrimSpace(message), " ")
			if len(arg) == 2 {
				password = arg[1]
			}

			if validLogin(user, password) {
				fmt.Fprintf(conn, "230 Login successful.\n")
			} else {
				fmt.Fprintf(conn, "530 Authentication failed.\n")
			}
		case "SYST":
			fmt.Fprintf(conn, "215 UNIX Type: L8\n")
		case "FEAT":
			fmt.Fprintf(conn, "211-Supported:\n SIZE\n ESPV\n UTF8\n211 End\n")
		case "CWD":
			fmt.Fprintf(conn, "250 CWD successful.\n")
		case "PWD":
			fmt.Fprintf(conn, "257 \"/\" is the remote directory.\n")
		case "TYPE":
			fmt.Fprintf(conn, "200 Type set to: Binary.\n")
		case "SIZE":
			arg := strings.Split(strings.TrimSpace(message), " ")[1]
			file, _ := os.Open(arg)
			stats, _ := file.Stat()
			fmt.Fprintf(conn, "213 %d\n", stats.Size())
		case "DELE":
			arg := strings.Split(strings.TrimSpace(message), " ")[1]
			os.Remove(arg)
			fmt.Fprintf(conn, "250 250 File removed.\n")
		case "STOR":
			fmt.Fprintf(conn, "125 Transfer starting.\n")
			func(tc *net.TCPConn) {
				arg := strings.Split(strings.TrimSpace(message), " ")[1]
				data, _ := ioutil.ReadAll(tc)
				ioutil.WriteFile(arg, data, 0644)
				tc.CloseRead()
			}(transferConn)
			transferConn = (*net.TCPConn)(nil)
			fmt.Fprintf(conn, "226 Transfer complete.\n")
		case "RETR":
			fmt.Fprintf(conn, "125 Transfer starting.\n")
			func(tc *net.TCPConn) {
				arg := strings.Split(strings.TrimSpace(message), " ")[1]
				data, _ := ioutil.ReadFile(arg)
				fmt.Fprintf(tc, string(data))
				tc.CloseWrite()
			}(transferConn)
			transferConn = (*net.TCPConn)(nil)
			fmt.Fprintf(conn, "226 Transfer complete.\n")
		case "LIST":
			fmt.Fprintf(conn, "125 Transfer starting.\n")
			func(tc *net.TCPConn) {
				output, _ := exec.Command("ls").Output()
				fmt.Fprintf(tc, "%s", strings.Replace(string(output), "\n", "\r\n", -1))
				tc.CloseWrite()
			}(transferConn)
			fmt.Fprintf(conn, "226 Transfer complete.\n")
		case "EPSV":
			fmt.Fprintf(conn, "229 Entering Passive Mode (|||1500|).\n")
			transferConn = getTransferConn()
		case "QUIT":
			fmt.Fprintf(conn, "221 Goodbye.\n")
		default:
			fmt.Printf("Not Found `%s`.\n", command)
			fmt.Fprintf(conn, "502 Command not implemented.\n")
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
