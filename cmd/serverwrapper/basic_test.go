package main

import (
	"os/exec"
)

type CmdChan interface {
	CombinedOutput() <-chan string
	Input() chan<- string
	Wait() error
}

type cmdChan struct {
	inner *exec.Cmd
}

func (cmd *cmdChan) Wait() {

}

// func TestThing(t *testing.T) {
// 	innerCmd := exec.Command("ls")
// 	cmd := cmdChan{innerCmd}
// 	require.NoError(t, innerCmd.Start())

// 	require.NoError(t, cmd.Wait())
// }
