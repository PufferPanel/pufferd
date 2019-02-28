package oauth2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pufferpanel/apufferi/common"
	"github.com/pufferpanel/apufferi/config"
	pufferdHttp "github.com/pufferpanel/apufferi/http"
	"github.com/pufferpanel/apufferi/logging"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

func ValidateToken(accessToken string, gin *gin.Context) bool {
	return validateToken(accessToken, gin, true)
}

func validateToken(accessToken string, gin *gin.Context, recurse bool) bool {
	authUrl := config.GetString("infoServer")
	data := url.Values{}
	data.Set("token", accessToken)
	encodedData := data.Encode()
	request, _ := http.NewRequest("POST", authUrl, bytes.NewBufferString(encodedData))

	RefreshIfStale()

	atLocker.RLock()
	request.Header.Add("Authorization", "Bearer "+daemonToken)
	atLocker.RUnlock()
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", strconv.Itoa(len(encodedData)))
	response, err := client.Do(request)
	if err != nil {
		logging.Error("Error talking to auth server", err)
		pufferdHttp.Respond(gin).Message(err.Error()).Fail().Status(500).Send()
		gin.Abort()
		return false
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		if response.StatusCode == 401 {
			//refresh token and repeat call
			//if we didn't refresh, then there's no reason to try again
			if recurse && RefreshToken() {
				response.Body.Close()
				return validateToken(accessToken, gin, false)
			}
		}

		logging.Error("Unexpected response code from auth server", response.StatusCode)
		pufferdHttp.Respond(gin).Message(fmt.Sprintf("unexpected response code %d", response.StatusCode)).Fail().Status(500).Send()
		gin.Abort()
		return false
	}

	var respArr map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&respArr)

	if err != nil {
		logging.Error("Error parsing response from auth server", err)
		pufferdHttp.Respond(gin).Message(err.Error()).Fail().Status(500).Send()
		gin.Abort()
		return false
	} else if respArr["error"] != nil {
		errStr, ok := respArr["error"].(string)
		if !ok {
			err = errors.New(fmt.Sprintf("error is %s instead of string", reflect.TypeOf(respArr["error"])))
			logging.Error("Error parsing response from auth server", err)
		} else {
			err = errors.New(errStr)
		}
		pufferdHttp.Respond(gin).Message(err.Error()).Fail().Status(500).Send()
		gin.Abort()
		return false
	}

	active, ok := respArr["active"].(bool)

	if !ok || !active {
		gin.AbortWithStatus(401)
		return false
	}

	serverMapping, ok := respArr["servers"].(map[string]interface{})
	if !ok {
		err = errors.New(fmt.Sprintf("auth server did not respond in the format expected, got %s instead of map[string]interface{} for servers", reflect.TypeOf(respArr["servers"])))
		logging.Error("Error parsing response from auth server", err)
		pufferdHttp.Respond(gin).Message(err.Error()).Fail().Status(500).Send()
		gin.Abort()
		return false
	}

	mapping := make(map[string][]string)

	for k, v := range serverMapping {
		mapping[k] = common.ToStringArray(v)
	}

	gin.Set("serverScopes", mapping)
	return true
}