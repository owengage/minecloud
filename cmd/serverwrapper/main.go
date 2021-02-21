package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/owengage/minecloud/pkg/serverwrapper"
)

/*

Example of joining
[08:28:47] [User Authenticator #1/INFO]: UUID of player NeroGage is a87fddc1-dc61-4c11-8472-f49001a15d21
[08:28:47] [Server thread/INFO]: NeroGage[/127.0.0.1:33062] logged in with entity id 366 at (-144.4019021007376, 64.0, -157.54162185876902)
[08:28:47] [Server thread/INFO]: NeroGage joined the game

Example of leaving
[08:28:57] [Server thread/INFO]: NeroGage left the game

*/

// CommandRequest represents HTTP request to perform a command.
type CommandRequest struct {
	Command string `json:"command"`
}

// MaybeErrResponse returned from requests.
type MaybeErrResponse struct {
	Error error `json:"error"`
}

func main() {
	address := flag.String("address", "", "IP address to bind to")
	serverJar := flag.String("jar", "", "Minecraft server JAR file")
	worldDir := flag.String("world-dir", "", "Directory containing world files")
	serverDir := flag.String("server-dir", "", "Directory containing server files")
	snapshotDir := flag.String("snapshot-dir", "", "Path to write world snapshot to")
	jvmMem := flag.String("server-memory", "", "amount of memory to run server with, defaults to 80% of available. eg 10G")
	flag.Parse()

	wrapper := NewWrapper(WrapperOpts{
		Jar:       *serverJar,
		WorldDir:  *worldDir,
		ServerDir: *serverDir,
		JVMMemory: *jvmMem,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go wrapper.Run(ctx)

	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req CommandRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = wrapper.Send(req.Command)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(MaybeErrResponse{Error: err})

	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {

		response := serverwrapper.StatusResponse{}

		response.Status = string(wrapper.Status())

		enc := json.NewEncoder(w)
		err := enc.Encode(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		result := make(chan error)
		saveTask := &SnapshotTask{
			wrapper:     wrapper,
			snapshotDir: *snapshotDir,
			worldDir:    *worldDir,
			result:      result,
		}

		wrapper.Execute(saveTask)
		err := <-result

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(MaybeErrResponse{Error: err})
	})

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		result := make(chan error)
		stopTask := &StopTask{
			wrapper: wrapper,
			result:  result,
		}

		wrapper.Execute(stopTask)
		err := <-result

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(MaybeErrResponse{Error: err})
	})

	server := &http.Server{Addr: *address, Handler: nil}
	go server.ListenAndServe()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	fmt.Println("Waiting for signal")
	s := <-c
	fmt.Println("Got signal:", s)

	cancel()
	server.Close()
	wrapper.Stop()
}
