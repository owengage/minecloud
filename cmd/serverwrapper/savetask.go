package main

import "strings"

type SaveTask struct {
	wrapper *Wrapper
	result  chan error
}

func (t *SaveTask) Init() TaskStep {
	err := t.wrapper.Send("save-all")
	if err != nil {
		t.result <- err
		return TaskDone
	}
	return TaskContinue
}

func (t *SaveTask) OnOutput(out string) TaskStep {
	if strings.Contains(out, "[Server thread/INFO]: Saved the game") {
		close(t.result)
		return TaskDone
	}
	return TaskContinue
}
