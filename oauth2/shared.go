/*
 Copyright 2019 Padduck, LLC

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

package oauth2

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/pufferpanel/apufferi/v4/logging"
	"github.com/pufferpanel/pufferd/v2/commons"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

var atLocker = &sync.RWMutex{}
var daemonToken string
var lastRefresh time.Time
var expiresIn time.Duration
var client = &http.Client{}

func RefreshToken() bool {
	atLocker.Lock()
	defer atLocker.Unlock()

	//if we just refreshed in the last minute, don't refresh the token
	if lastRefresh.Add(1 * time.Minute).After(time.Now()) {
		return false
	}

	clientId := viper.GetString("auth.clientId")
	if clientId == "" {
		logging.Exception("error talking to auth server", errors.New("client id not specified"))
		return false
	}

	clientSecret := viper.GetString("auth.clientSecret")
	if clientSecret == "" {
		logging.Exception("error talking to auth server", errors.New("client secret not specified"))
		return false
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	encodedData := data.Encode()

	request := createRequest(encodedData)

	request.Header.Add("Authorization", "Bearer "+daemonToken)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", strconv.Itoa(len(encodedData)))
	response, err := client.Do(request)
	defer commons.CloseResponse(response)
	if err != nil {
		logging.Exception("error talking to auth server", err)
		return false
	}

	var responseData requestResponse
	err = json.NewDecoder(response.Body).Decode(&responseData)

	if responseData.Error != "" {
		return false
	}

	daemonToken = responseData.AccessToken
	lastRefresh = time.Now()
	expiresIn = responseData.ExpiresIn

	return true
}

func RefreshIfStale() {
	//we know the token only lasts about an hour,
	//so we'll check to see if we know the cache is old
	atLocker.RLock()
	oldCache := lastRefresh.Add(expiresIn).Before(time.Now())
	atLocker.RUnlock()
	if oldCache {
		RefreshToken()
	}
}

func createRequest(encodedData string) (request *http.Request) {
	authUrl := viper.GetString("auth.url")

	logging.Debug("Using standard http connection")
	request, _ = http.NewRequest("POST", authUrl+"/oauth2/token", bytes.NewBufferString(encodedData))
	return
}

type requestResponse struct {
	AccessToken string        `json:"access_token"`
	ExpiresIn   time.Duration `json:"expires_in"`
	Error       string        `json:"error"`
}
