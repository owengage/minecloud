package serverwrapper

// StatusResponse is the response from the status endpoint
type StatusResponse struct {
	Status string
}

type Status string

const StatusStarting = "starting"
const StatusRunning = "running"
const StatusStopped = "stopped"
