package wrapper

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var ImageName = "minecloud/server-wrapper"

func getGitRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")

	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(out))
}

func GoRunCmd(t *testing.T, pkg string, args ...string) (cmd *exec.Cmd, kill func()) {
	newArgs := append([]string{}, "run", pkg)
	newArgs = append(newArgs, args...)
	cmd = exec.Command("go", newArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	kill = func() {
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		require.NoError(t, err)
		err = syscall.Kill(-pgid, 15) // note the minus sign
		require.NoError(t, err)
	}
	return
}

func TestStartAndStop(t *testing.T) {
	jarPath := "system_tests/resources/server.jar"
	serverDir := "system_tests/resources/basic-server-files"
	worldDir := "system_tests/resources/basic-world"
	snapPath := "system_tests/resources/snap.tar"

	cmd, kill := GoRunCmd(t, "github.com/owengage/minecloud/cmd/serverwrapper",
		"-address", "0.0.0.0:8080",
		"-jar", jarPath,
		"-server-dir", serverDir,
		"-world-dir", worldDir,
		"-snapshot-path", snapPath)
	cmd.Dir = getGitRoot()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()

	time.Sleep(1 * time.Second)

	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/stop", nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	ioutil.ReadAll(res.Body)

	kill()
	err = cmd.Wait()
	require.NoError(t, err)

	// Does this actually test anything?

	defer res.Body.Close()
}
