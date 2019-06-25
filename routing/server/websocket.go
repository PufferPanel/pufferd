package server

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/pufferpanel/apufferi"
	"github.com/pufferpanel/apufferi/logging"
	"github.com/pufferpanel/pufferd/messages"
	"github.com/pufferpanel/pufferd/programs"
	"reflect"
	"strings"
)

func listenOnSocket(conn *websocket.Conn, server programs.Program, scopes []string) {
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			logging.Exception("error on reading from websocket", err)
			return
		}
		if msgType != websocket.TextMessage {
			continue
		}
		mapping := make(map[string]interface{})

		err = json.Unmarshal(data, &mapping)
		if err != nil {
			logging.Exception("error on decoding websocket message", err)
			continue
		}

		messageType := mapping["type"]
		if message, ok := messageType.(string); ok {
			switch strings.ToLower(message) {
			case "stat":
				{
					if apufferi.ContainsValue(scopes, "server.stats") {
						results, err := server.GetEnvironment().GetStats()
						msg := messages.StatMessage{}
						if err != nil {
							msg.Cpu = 0
							msg.Memory = 0
						} else {
							msg.Cpu, _ = results["cpu"].(float64)
							msg.Memory, _ = results["memory"].(float64)
						}
						_ = messages.Write(conn, msg)
					}
				}
			case "start":
				{
					if apufferi.ContainsValue(scopes, "server.start") {
						_ = server.Start()
					}
					break
				}
			case "stop":
				{
					if apufferi.ContainsValue(scopes, "server.stop") {
						_ = server.Stop()
					}
				}
			case "install":
				{
					if apufferi.ContainsValue(scopes, "server.install") {
						_ = server.Install()
					}
				}
			case "kill":
				{
					if apufferi.ContainsValue(scopes, "server.kill") {
						_ = server.Kill()
					}
				}
			case "reload":
				{
					if apufferi.ContainsValue(scopes, "server.reload") {
						_ = programs.Reload(server.Id())
					}
				}
			case "ping":
				{
					_ = messages.Write(conn, messages.PongMessage{})
				}
			case "console":
				{
					cmd, ok := mapping["command"].(string)
					if ok {
						if run, _ := server.IsRunning(); run {
							_ = server.GetEnvironment().ExecuteInMainProcess(cmd)
						}
					}
				}
			case "file":
				{
					if !apufferi.ContainsValue(scopes, "server.files") {
						break
					}

					action, ok := mapping["action"].(string)
					if !ok {
						break
					}
					path, ok := mapping["path"].(string)
					if !ok {
						break
					}

					switch strings.ToLower(action) {
					case "get":
						{
							//can we get a file here?
							file, list, err := server.GetItem(path)
							if err != nil {
								_ = messages.Write(conn, messages.FileListMessage{Error: err.Error()})
								break
							}

							//we are not going to send it via websocket since it's a mess at this stage
							_ = file.Close()

							if list != nil {
								_ = messages.Write(conn, messages.FileListMessage{FileList: list})
							} else if file != nil {
								_ = messages.Write(conn, messages.FileListMessage{Url: path})
							}
						}
					case "delete":
						{
							if !apufferi.ContainsValue(scopes, "server.files.delete") {
								break
							}

							err := server.DeleteItem(path)
							if err != nil {
								_ = messages.Write(conn, messages.FileListMessage{Error: err.Error()})
							}
						}
					case "create":
						{
							if !apufferi.ContainsValue(scopes, "server.files.put") {
								break
							}

							err := server.CreateFolder(path)

							if err != nil {
								_ = messages.Write(conn, messages.FileListMessage{Error: err.Error()})
							}
						}
					}
				}
			default:
				_ = conn.WriteJSON(map[string]string{"error": "unknown command"})
			}
		} else {
			logging.Error("message type is not a string, but was %s", reflect.TypeOf(messageType))
		}
	}
}
