// +build !windows

/*
 Copyright 2016 Padduck, LLC

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 	http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package tty

import (
	"fmt"
	"github.com/creack/pty"
	"github.com/pufferpanel/apufferi/v4/logging"
	"github.com/pufferpanel/pufferd/v2"
	"github.com/pufferpanel/pufferd/v2/environments/envs"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cast"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type tty struct {
	*envs.BaseEnvironment
	mainProcess *exec.Cmd
	stdInWriter io.Writer
}

func (t *tty) ttyExecuteAsync(cmd string, args []string, env map[string]string, callback func(graceful bool)) (err error) {
	running, err := t.IsRunning()
	if err != nil {
		return
	}
	if running {
		err = pufferd.ErrProcessRunning
		return
	}
	t.Wait.Wait()

	pr := exec.Command(cmd, args...)
	pr.Dir = t.RootDirectory
	pr.Env = append(os.Environ(), "HOME="+t.RootDirectory)
	for k, v := range env {
		pr.Env = append(pr.Env, fmt.Sprintf("%s=%s", k, v))
	}

	wrapper := t.CreateWrapper()
	t.Wait.Add(1)
	pr.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}
	t.mainProcess = pr
	logging.Debug("Starting process: %s %s", t.mainProcess.Path, strings.Join(t.mainProcess.Args[1:], " "))
	tty, err := pty.Start(pr)
	if err != nil {
		return
	}

	t.stdInWriter = tty

	go func(proxy io.Writer) {
		_, _ = io.Copy(proxy, tty)
	}(wrapper)

	go t.handleClose(callback)
	return
}

func (t *tty) ExecuteInMainProcess(cmd string) (err error) {
	running, err := t.IsRunning()
	if err != nil {
		return err
	}
	if !running {
		err = pufferd.ErrServerOffline
		return
	}
	stdIn := t.stdInWriter
	_, err = io.WriteString(stdIn, cmd+"\n")
	return
}

func (t *tty) Kill() (err error) {
	running, err := t.IsRunning()
	if err != nil {
		return
	}
	if !running {
		return
	}
	return t.mainProcess.Process.Kill()
}

func (t *tty) IsRunning() (isRunning bool, err error) {
	isRunning = t.mainProcess != nil && t.mainProcess.Process != nil
	if isRunning {
		pr, pErr := os.FindProcess(t.mainProcess.Process.Pid)
		if pr == nil || pErr != nil {
			isRunning = false
		} else if pr.Signal(syscall.Signal(0)) != nil {
			isRunning = false
		}
	}
	return
}

func (t *tty) GetStats() (*pufferd.ServerStats, error) {
	running, err := t.IsRunning()
	if err != nil {
		return nil, err
	}
	if !running {
		return nil, pufferd.ErrServerOffline
	}
	pr, err := process.NewProcess(int32(t.mainProcess.Process.Pid))
	if err != nil {
		return nil, err
	}

	memMap, _ := pr.MemoryInfo()
	cpu, _ := pr.Percent(time.Second * 1)

	return &pufferd.ServerStats{
		Cpu:    cpu,
		Memory: cast.ToFloat64(memMap.RSS),
	}, nil
}

func (t *tty) Create() error {
	return os.Mkdir(t.RootDirectory, 0755)
}

func (t *tty) WaitForMainProcess() error {
	return t.WaitForMainProcessFor(0)
}

func (t *tty) WaitForMainProcessFor(timeout int) (err error) {
	running, err := t.IsRunning()
	if err != nil {
		return
	}
	if running {
		if timeout > 0 {
			var timer = time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
				err = t.Kill()
			})
			t.Wait.Wait()
			timer.Stop()
		} else {
			t.Wait.Wait()
		}
	}
	return
}

func (t *tty) SendCode(code int) error {
	running, err := t.IsRunning()

	if err != nil || !running {
		return err
	}

	return t.mainProcess.Process.Signal(syscall.Signal(code))
}

func (t *tty) handleClose(callback func(graceful bool)) {
	err := t.mainProcess.Wait()

	var success bool
	if t.mainProcess == nil || t.mainProcess.ProcessState == nil || err != nil {
		success = false
	} else {
		success = t.mainProcess.ProcessState.Success()
	}

	if t.mainProcess != nil && t.mainProcess.Process != nil {
		_ = t.mainProcess.Process.Release()
	}
	t.mainProcess = nil
	t.Wait.Done()

	if callback != nil {
		callback(success)
	}
}
