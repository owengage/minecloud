package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"reflect"
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

	tasks chan Task
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
		tasks:            make(chan Task),
	}
}

func (wrapper *Wrapper) Execute(task Task) {
	wrapper.tasks <- task
}

// RequestStop sends a request for the server to stop.
func (wrapper *Wrapper) RequestStop() error {
	wrapper.input <- "/stop\n"
	return <-wrapper.inputResponse
}

func (wrapper *Wrapper) Save() error {
	return nil
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

func (wrapper *Wrapper) Stop() {
	close(wrapper.done)
}

func (wrapper *Wrapper) Run(ctx context.Context) {
	var currentTask Task = &WaitForStartedTask{wrapper}
	var tasks chan Task

	go func() {
		err := wrapper.runServer(ctx)
		if err != nil {
			panic(err)
		}
	}()

	for {
		select {
		case line := <-wrapper.output:
			claimedMsg := "NoTask"
			if currentTask != nil {
				claimedMsg = getTaskName(currentTask)
				if currentTask.OnOutput(line) == TaskDone {
					currentTask = nil
					tasks = wrapper.tasks
				}
			}
			fmt.Printf("%s %s\n", claimedMsg, line)
		case task := <-tasks:
			if task.Init() == TaskContinue {
				currentTask = task
				tasks = nil
			}
		case <-wrapper.done:
			if currentTask == nil {
				return
			}

			if t, ok := currentTask.(TaskTerminatable); ok {
				t.OnTerminate()
			}
			return
		}
	}
}

func getTaskName(task Task) string {
	t := reflect.TypeOf(task)
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	return t.Name()
}

type WaitForStartedTask struct {
	wrapper *Wrapper
}

func (task *WaitForStartedTask) Init() TaskStep { return TaskContinue }
func (task *WaitForStartedTask) OnOutput(line string) TaskStep {
	if strings.Contains(line, "[Server thread/INFO]: Done") {
		task.wrapper.finishedStarting = true
		return TaskDone
	}
	return TaskContinue
}

// runServer. Blocks until server is closed. Use `go server.Run()`.
func (wrapper *Wrapper) runServer(ctx context.Context) (err error) {

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

	availableMiB, err := serverwrapper.AvailableMiB()
	if err != nil {
		return
	}

	claimMiB := (availableMiB * 80) / 100

	jvmOptions := []string{
		// Mostly from https://aikar.co/2018/07/02/tuning-the-jvm-g1gc-garbage-collector-flags-for-minecraft/.
		fmt.Sprintf("-Xms%dM", claimMiB),
		fmt.Sprintf("-Xmx%dM", claimMiB),
		"-XX:+UseG1GC",
		"-XX:+ParallelRefProcEnabled",
		"-XX:MaxGCPauseMillis=200",
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+DisableExplicitGC",
		"-XX:-OmitStackTraceInFastThrow",
		"-XX:+AlwaysPreTouch",
		"-XX:G1NewSizePercent=30",
		"-XX:G1MaxNewSizePercent=40",
		"-XX:G1HeapRegionSize=8M",
		"-XX:G1ReservePercent=20",
		"-XX:G1HeapWastePercent=5",
		"-XX:G1MixedGCCountTarget=8",
		"-XX:InitiatingHeapOccupancyPercent=15",
		"-XX:G1MixedGCLiveThresholdPercent=90",
		"-XX:G1RSetUpdatingPauseTimePercent=5",
		"-XX:SurvivorRatio=32",
		"-XX:MaxTenuringThreshold=1"}

	minecraftOptions := []string{"-jar", jar,
		"--universe", universe,
		"--world", world,
		"nogui"}

	opts := []string{}
	opts = append(opts, jvmOptions...)
	opts = append(opts, minecraftOptions...)

	log.Println("java command options:", opts)

	cmd := exec.CommandContext(ctx, "java", opts...)

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
		}
		if scanner.Err() != nil {
			log.Println(scanner.Err())
		}
	}

	go readToChan(stdout)
	go readToChan(stderr)

	go func() {
		for command := range wrapper.input {
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
