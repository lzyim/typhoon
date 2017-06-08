package typhoon

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
)

func Run() {
	ln, _ := net.Listen("tcp", ":8080")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalln(err)
			return
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	defer conn.Close()
	for {
		if line, _, err := reader.ReadLine(); err == nil {
			requestLine := bytes.Split(line, []byte(" "))
			if bytes.Equal(requestLine[0], []byte("GET")) {
				path := parseUri(requestLine[1])
				if isStatic(path[len(path)-1]) {
					serveStatic(requestLine[1], writer)
				} else {
					serveDynamic(requestLine[1], writer)
				}
			}
		} else {
			return
		}
	}
}

func parseUri(uri []byte) [][]byte {
	return bytes.Split(uri, []byte("/"))
}

func getArgs(path []byte) ([][]byte, bool) {
	args := bytes.Split(path, []byte("?"))
	if len(args) < 2 {
		return args, false
	}
	return args, true
}

func isStatic(filename []byte) bool {
	isStatic := false
	ext := bytes.Split(filename, []byte("."))
	if len(ext) < 2 {
		return isStatic
	}
	switch {
	case bytes.Equal(ext[1], []byte("css")):
		isStatic = true
	case bytes.Equal(ext[1], []byte("js")):
		isStatic = true
	case bytes.Equal(ext[1], []byte("gif")):
		isStatic = true
	case bytes.Equal(ext[1], []byte("ico")):
		isStatic = true
	case bytes.Equal(ext[1], []byte("jpg")):
		isStatic = true
	case bytes.Equal(ext[1], []byte("jpeg")):
		isStatic = true
	case bytes.Equal(ext[1], []byte("png")):
		isStatic = true
	}
	return isStatic
}

func getFileType(filename []byte) string {
	switch {
	case bytes.Contains(filename, []byte(".html")):
		return "text/html"
	case bytes.Contains(filename, []byte(".css")):
		return "text/css"
	case bytes.Contains(filename, []byte(".js")):
		return "text/javascript"
	case bytes.Contains(filename, []byte(".gif")):
		return "text/gif"
	case bytes.Contains(filename, []byte(".png")):
		return "text/png"
	case bytes.Contains(filename, []byte(".jpg")):
		return "text/jepg"
	default:
		return "text/plain"
	}
}

func getInterpreter(filename []byte) string {
	switch {
	case bytes.Contains(filename, []byte(".js")):
		return "node"
	case bytes.Contains(filename, []byte(".php")):
		return "php"
	case bytes.Contains(filename, []byte(".py")):
		return "python"
	case bytes.Contains(filename, []byte(".rb")):
		return "ruby"
	default:
		return ""
	}
}

func handleErr(err error, writer *bufio.Writer) {
	writer.Write([]byte("HTTP/1.1 404 Not Found\r\nServer: Typhoon\r\n"))
	writer.Write([]byte("Connection: close\r\n\r\n"))
	writer.Flush()
	log.Print(err)
}

func serveStatic(file []byte, writer *bufio.Writer) {
	stFile, err := os.Open(string(file[:]))
	defer stFile.Close()
	if err != nil {
		handleErr(err, writer)
		return
	}
	fi, err := stFile.Stat()
	if err != nil {
		handleErr(err, writer)
		return
	}
	writer.Write([]byte("HTTP/1.1 200 OK\r\nServer: Typhoon\r\n"))
	length := fmt.Sprintf("Content-Length: %d\r\n", fi.Size())
	writer.Write([]byte(length))
	fType := fmt.Sprintf("Content-Type: %s\r\n", getFileType(file))
	writer.Write([]byte(fType))
	writer.Write([]byte("Connection: keep-alive\r\n\r\n"))
	buf := make([]byte, 1024)
	for {
		n, err := stFile.Read(buf)
		if n > 0 {
			writer.Write(buf)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("read %d bytes: %v", n, err)
			break
		}
	}
	writer.Write([]byte("\r\n"))
	writer.Flush()
}

func serveDynamic(file []byte, writer *bufio.Writer) {
	args, ok := getArgs(file)
	if ok {
		os.Setenv("QUERY_STRING", string(args[1]))
	}
	pwd, _ := os.Getwd()
	out, err := exec.Command(getInterpreter(args[0]), fmt.Sprintf("%s/%s", pwd, string(args[0]))).Output()
	if err != nil {
		handleErr(err, writer)
		return
	}
	writer.Write([]byte("HTTP/1.1 200 OK\r\nServer: Typhoon\r\n"))
	writer.Write([]byte("Content-Type: text/html\r\n"))
	length := fmt.Sprintf("Content-Length: %d\r\n", len(out))
	writer.Write([]byte(length))
	writer.Write([]byte("Connection: keep-alive\r\n\r\n"))
	writer.Write(out)
	writer.Write([]byte("\r\n"))
	writer.Flush()
}
