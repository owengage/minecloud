package main

type TaskStep int

const (
	TaskContinue TaskStep = iota
	TaskDone
)

type Task interface {
	Init() TaskStep
	OnOutput(string) TaskStep
}

type TaskTerminatable interface {
	OnTerminate()
}
