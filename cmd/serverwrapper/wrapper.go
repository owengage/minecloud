package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/owengage/minecloud/pkg/serverwrapper"
)

// Wrapper is a Minecraft server.
type Wrapper struct {
	output        chan string
	input         chan string
	inputResponse chan error
	done          chan struct{}

	jar       string
	serverDir string
	worldDir  string

	finishedStarting bool
	stopRequested    bool
}

// WrapperOpts are the options for creating a server.
type WrapperOpts struct {
	Jar       string
	WorldDir  string
	ServerDir string
}

// NewWrapper prepares a new Minecraft server for launch.
func NewWrapper(opts WrapperOpts) *Wrapper {
	out := make(chan string, 0)
	in := make(chan string, 0)
	inResponse := make(chan error, 0)

	done := make(chan struct{}, 0)

	return &Wrapper{
		output:           out,
		input:            in,
		inputResponse:    inResponse,
		done:             done,
		jar:              opts.Jar,
		serverDir:        opts.ServerDir,
		worldDir:         opts.WorldDir,
		finishedStarting: false,
	}
}

// Output from server console.
func (wrapper *Wrapper) Output() <-chan string {
	return wrapper.output
}

// RequestStop sends a request for the server to stop.
func (wrapper *Wrapper) RequestStop() error {
	wrapper.input <- "/stop\n"
	return <-wrapper.inputResponse
}

// Send command to server. New line automatically appended. eg Send("/stop")
func (wrapper *Wrapper) Send(cmd string) error {
	wrapper.input <- cmd + "\n"
	return <-wrapper.inputResponse
}

// Status of the server
func (wrapper *Wrapper) Status() serverwrapper.Status {
	select {
	case <-wrapper.done:
		return serverwrapper.StatusStopped
	default:
	}

	if wrapper.finishedStarting {
		return serverwrapper.StatusRunning
	}

	return serverwrapper.StatusStarting
}

// Run the server. Blocks until server is closed. Use `go server.Run()`.
func (wrapper *Wrapper) Run() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("wrapper run: %w", err)
		}
	}()

	// Get some absolute paths since the command will be running from a
	// different directory.
	jar, err := filepath.Abs(wrapper.jar)
	if err != nil {
		return
	}

	worldDir, err := filepath.Abs(wrapper.worldDir)
	if err != nil {
		return
	}

	universe, world, err := getUniverseAndWorld(worldDir)
	if err != nil {
		return
	}

	cmd := exec.Command("java",
		"-jar", jar,
		"--universe", universe,
		"--world", world)

	cmd.Dir = wrapper.serverDir

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
			wrapper.output <- line

			if strings.Contains(line, "[Wrapper thread/INFO]: Done") {
				wrapper.finishedStarting = true
			}
		}
		if scanner.Err() != nil {
			log.Println(scanner.Err())
		}
	}

	go readToChan(stdout)
	go readToChan(stderr)

	go func() {
		for command := range wrapper.input {
			fmt.Printf("Sending: %s\n", command)
			_, err := in.Write([]byte(command))
			wrapper.inputResponse <- err // might be nil.
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

	close(wrapper.done)
	fmt.Printf("Minecraft Wrapper terminated")
	return nil
}
