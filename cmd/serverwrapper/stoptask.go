package main

type StopTask struct {
	wrapper *Wrapper
	result  chan error
}

func (t *StopTask) Init() TaskStep {
	err := t.wrapper.Send("stop")
	if err != nil {
		t.result <- err
		return TaskDone
	}
	return TaskContinue
}

func (t *StopTask) OnOutput(out string) TaskStep {
	return TaskContinue
}

func (t *StopTask) OnTerminate() {
	close(t.result)
}
