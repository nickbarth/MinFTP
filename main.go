package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func getTransferConn(port string) *net.TCPConn {
	server, _ := net.Listen("tcp", ":"+port)
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

func getArg(message string) string {
	args := strings.Split(strings.TrimSpace(message), " ")
	return args[len(args)-1]
}

func getFilename(arg string) string {
	reg, _ := regexp.Compile("[^a-z0-9.]+")
	path := strings.Split(strings.ToLower(arg), "/")
	return reg.ReplaceAllString(path[len(path)-1], "")
}

func handleConn(conn net.Conn) {
	user := "anonymous"
	password := "anonymous"

	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	port := strconv.Itoa(6000 + rand.Intn(1000))

	transferConn := (*net.TCPConn)(nil)
	buff := bufio.NewReader(conn)

	fmt.Fprintf(conn, "200 FTP Server ready.\n")
	fmt.Print("Connected.\n")

	for {
		message, err := buff.ReadString('\n')
		command := strings.Split(strings.TrimSpace(message), " ")[0]

		arg := getArg(message)
		filename := getFilename(arg)
		stats, fileErr := os.Stat(filename)

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
			user = arg
			fmt.Fprintf(conn, "331 User okay. Please specify the password.\n")
		case "PASS":
			password = arg
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
			fmt.Fprintf(conn, "250 \"/\" is the current directory.\n")
		case "PWD":
			fmt.Fprintf(conn, "257 \"/\" is the remote directory.\n")
		case "TYPE":
			fmt.Fprintf(conn, "200 Type set to: Binary.\n")
		case "SIZE":
			size := int64(0)
			if fileErr == nil {
				size = stats.Size()
			}
			fmt.Fprintf(conn, "213 %d\n", size)
		case "DELE":
			os.Remove(filename)
			fmt.Fprintf(conn, "250 File removed.\n")
		case "STOR":
			fmt.Fprintf(conn, "125 Transfer starting.\n")
			func(tc *net.TCPConn) {
				data, _ := ioutil.ReadAll(tc)
				ioutil.WriteFile(filename, data, 0644)
				tc.CloseRead()
			}(transferConn)
			transferConn = (*net.TCPConn)(nil)
			fmt.Fprintf(conn, "226 Transfer complete.\n")
		case "RETR":
			fmt.Fprintf(conn, "125 Transfer starting.\n")
			func(tc *net.TCPConn) {
				data, _ := ioutil.ReadFile(filename)
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
			fmt.Fprintf(conn, "229 Entering Passive Mode (|||"+port+"|).\n")
			transferConn = getTransferConn(port)
			ipcmp, _, _ := net.SplitHostPort(transferConn.RemoteAddr().String())
			if ip != ipcmp {
				fmt.Fprintf(conn, "550 Not authorized.\n")
				conn.Close()
				transferConn.Close()
			}
		case "QUIT":
			fmt.Fprintf(conn, "221 Goodbye.\n")
		default:
			fmt.Printf("Not Found `%s`.\n", command)
			fmt.Fprintf(conn, "502 Command not implemented.\n")
		}
	}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	server, _ := net.Listen("tcp", ":2121")
	defer server.Close()

	for {
		conn, _ := server.Accept()
		go handleConn(conn)
	}
}
