package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type SnapshotTask struct {
	wrapper     *Wrapper
	worldDir    string
	snapshotDir string
	result      chan error
}

func (t *SnapshotTask) Init() TaskStep {
	err := t.wrapper.Send("save-off")
	if err != nil {
		t.result <- fmt.Errorf("save-on failed: %w", err)
		return TaskDone
	}

	err = t.wrapper.Send("save-all flush")
	if err != nil {
		t.result <- fmt.Errorf("save-on failed: %w", err)
		return TaskDone
	}

	return TaskContinue
}

func (t *SnapshotTask) OnOutput(out string) TaskStep {
	if strings.Contains(out, "[Server thread/INFO]: Saved the game") {
		cmd := exec.Command("cp", "-r", "-a", t.worldDir, t.snapshotDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.result <- fmt.Errorf("cp failed: %w: %s", err, out)
			return TaskDone
		}

		err = t.wrapper.Send("save-on")
		if err != nil {
			t.result <- fmt.Errorf("save-on failed: %w", err)
			return TaskDone
		}

		return TaskContinue
	}

	if strings.Contains(out, "[Server thread/INFO]: Automatic saving is now enabled") {
		close(t.result)
		return TaskDone
	}

	return TaskContinue
}
