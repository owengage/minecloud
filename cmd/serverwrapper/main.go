package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
)

type StatusResponse struct {
	Status string
}

func main() {
	address := flag.String("address", "", "IP address to bind to")
	serverJar := flag.String("jar", "", "Minecraft server JAR file")
	serverDir := flag.String("server-dir", "", "Directory containing server files and world")

	flag.Parse()

	server := NewServer(*serverJar, *serverDir)

	go func() {
		err := server.Run()
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		for message := range server.Output() {
			fmt.Println(message)
		}
	}()

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := server.RequestStop()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {

		response := StatusResponse{}

		response.Status = string(server.Status())

		enc := json.NewEncoder(w)
		err := enc.Encode(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	log.Fatal(http.ListenAndServe(*address, nil))
}

// Server is a Minecraft server.
type Server struct {
	output        chan string
	input         chan string
	inputResponse chan error
	done          chan struct{}

	jar string
	dir string

	finishedStarting bool
	stopRequested    bool
}

// NewServer prepares a new Minecraft server for launch.
func NewServer(jar, dir string) *Server {
	out := make(chan string, 0)
	in := make(chan string, 0)
	inResponse := make(chan error, 0)

	done := make(chan struct{}, 0)

	return &Server{
		output:           out,
		input:            in,
		inputResponse:    inResponse,
		done:             done,
		jar:              jar,
		dir:              dir,
		finishedStarting: false,
	}
}

type Status string

const StatusStarting = "starting"
const StatusRunning = "running"
const StatusStopped = "stopped"

// Output from server console.
func (server *Server) Output() <-chan string {
	return server.output
}

// RequestStop sends a request for the server to stop.
func (server *Server) RequestStop() error {
	server.input <- "/stop\n"
	return <-server.inputResponse
}

// Status of the server
func (server *Server) Status() Status {
	select {
	case <-server.done:
		return StatusStopped
	default:
	}

	if server.finishedStarting {
		return StatusRunning
	}

	return StatusStarting
}

// Run the server. Blocks until server is closed. Use `go server.Run()`.
func (server *Server) Run() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("server run: %w", err)
		}
	}()

	// Get absolute path to JAR since the command will be running from a
	// different directory.
	jar, err := filepath.Abs(server.jar)
	if err != nil {
		return
	}

	cmd := exec.Command("java", "-jar", jar)
	cmd.Dir = server.dir

	in, err := cmd.StdinPipe()
	if err != nil {
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	readToChan := func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			server.output <- line

			if strings.Contains(line, "[Server thread/INFO]: Done") {
				server.finishedStarting = true
			}
		}
		if scanner.Err() != nil {
			log.Println(scanner.Err())
		}
	}

	go readToChan(stdout)
	go readToChan(stderr)

	go func() {
		for command := range server.input {
			fmt.Printf("Sending: %s\n", command)
			_, err := in.Write([]byte(command))
			server.inputResponse <- err // might be nil.
		}
	}()

	err = cmd.Start()
	if err != nil {
		return
	}

	err = cmd.Wait()

	if err != nil {
		return
	}

	close(server.done)
	return nil
}
