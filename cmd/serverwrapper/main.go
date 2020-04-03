package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

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
	flag.Parse()

	wrapper := NewWrapper(WrapperOpts{
		Jar:       *serverJar,
		WorldDir:  *worldDir,
		ServerDir: *serverDir,
	})

	go func() {
		err := wrapper.Run()
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		for message := range wrapper.Output() {
			fmt.Println(message)
		}
	}()

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := wrapper.RequestStop()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

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

	log.Fatal(http.ListenAndServe(*address, nil))
}
