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

package environments

import (
	"errors"
	"github.com/pufferpanel/apufferi/cache"
	"github.com/pufferpanel/pufferd/utils"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"fmt"
	"github.com/pufferpanel/apufferi/logging"
	ppError "github.com/pufferpanel/pufferd/errors"
	"github.com/shirou/gopsutil/process"
	"strings"
)

type standard struct {
	*BaseEnvironment
	mainProcess *exec.Cmd
	stdInWriter io.Writer
}

func (s *standard) standardExecuteAsync(cmd string, args []string, env map[string]string, callback func(graceful bool)) (err error) {
	running, err := s.IsRunning()
	if err != nil {
		return
	}
	if running {
		err = errors.New("process is already running (" + strconv.Itoa(s.mainProcess.Process.Pid) + ")")
		return
	}
	s.wait.Wait()
	s.mainProcess = exec.Command(cmd, args...)
	s.mainProcess.Dir = s.RootDirectory
	s.mainProcess.Env = append(os.Environ(), "HOME="+s.RootDirectory)
	for k, v := range env {
		s.mainProcess.Env = append(s.mainProcess.Env, fmt.Sprintf("%s=%s", k, v))
	}
	wrapper := s.createWrapper()
	s.mainProcess.Stdout = wrapper
	s.mainProcess.Stderr = wrapper
	pipe, err := s.mainProcess.StdinPipe()
	if err != nil {
		logging.Error("Error creating process", err)
	}
	s.stdInWriter = pipe
	s.wait.Add(1)
	logging.Debugf("Starting process: %s %s", s.mainProcess.Path, strings.Join(s.mainProcess.Args, " "))
	err = s.mainProcess.Start()
	go func() {
		err := s.mainProcess.Wait()
		s.wait.Done()
		if callback != nil {
			if s.mainProcess == nil || s.mainProcess.ProcessState == nil || err != nil  {
				callback(false)
			} else {
				callback(s.mainProcess.ProcessState.Success())
			}
		}
	}()
	if err != nil && err.Error() != "exit status 1" {
		logging.Error("Error starting process", err)
	} else {
		logging.Debug("Process started (" + strconv.Itoa(s.mainProcess.Process.Pid) + ")")
	}
	return
}

func (s *standard) ExecuteInMainProcess(cmd string) (err error) {
	running, err := s.IsRunning()
	if err != nil {
		return err
	}
	if !running {
		err = errors.New("main process has not been started")
		return
	}
	stdIn := s.stdInWriter
	_, err = io.WriteString(stdIn, cmd+"\n")
	return
}

func (s *standard) Kill() (err error) {
	running, err := s.IsRunning()
	if err != nil {
		return err
	}
	if !running {
		return
	}
	err = s.mainProcess.Process.Kill()
	err2 := s.mainProcess.Process.Release()
	if err == nil {
		err = err2
	}
	s.mainProcess = nil
	return
}

func (s *standard) IsRunning() (isRunning bool, err error) {
	isRunning = s.mainProcess != nil && s.mainProcess.Process != nil
	if isRunning {
		pr, pErr := os.FindProcess(s.mainProcess.Process.Pid)
		if pr == nil || pErr != nil {
			isRunning = false
		} else if runtime.GOOS != "windows" && pr.Signal(syscall.Signal(0)) != nil{
			isRunning = false
		}
	}
	return
}

func (s *standard) GetStats() (map[string]interface{}, error) {
	running, err := s.IsRunning()
	if err != nil {
		return nil, err
	}
	if !running {
		return nil, ppError.NewServerOffline()
	}
	pr, err := process.NewProcess(int32(s.mainProcess.Process.Pid))
	if err != nil {
		return nil, err
	}
	resultMap := make(map[string]interface{})
	memMap, _ := pr.MemoryInfo()
	resultMap["memory"] = memMap.RSS
	cpu, _ := pr.Percent(time.Second * 1)
	resultMap["cpu"] = cpu
	return resultMap, nil
}

func (e *standard) Create() error {
	return os.Mkdir(e.RootDirectory, 0755)
}

func (e *standard) WaitForMainProcess() error {
	return e.WaitForMainProcessFor(0)
}

func (e *standard) WaitForMainProcessFor(timeout int) (err error) {
	running, err := e.IsRunning()
	if err != nil {
		return
	}
	if running {
		if timeout > 0 {
			var timer = time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
				err = e.Kill()
			})
			e.wait.Wait()
			timer.Stop()
		} else {
			e.wait.Wait()
		}
	}
	return
}

func (e *standard) SendCode(code int) error {
	running, err := e.IsRunning()

	if err != nil || !running {
		return err
	}

	return e.mainProcess.Process.Signal(syscall.Signal(code))
}

type StandardFactory struct {
	EnvironmentFactory
}

func (sf StandardFactory) Create(folder, id string, environmentSection map[string]interface{}, rootDirectory string, cache cache.Cache, wsManager utils.WebSocketManager) Environment {
	s := &standard{BaseEnvironment: &BaseEnvironment{Type: "standard"}}
	s.BaseEnvironment.executeAsync = s.standardExecuteAsync
	s.BaseEnvironment.waitForMainProcess = s.WaitForMainProcess
	s.wait = sync.WaitGroup{}

	s.RootDirectory = rootDirectory
	s.ConsoleBuffer = cache
	s.WSManager = wsManager
	return s
}

func (sf StandardFactory) Key() string {
	return "standard"
}