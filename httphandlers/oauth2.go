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

package httphandlers

import (
	"github.com/gin-gonic/gin"
	"github.com/pufferpanel/apufferi/v4"
	"github.com/pufferpanel/apufferi/v4/response"
	"github.com/pufferpanel/apufferi/v4/scope"
	"github.com/pufferpanel/pufferd/v2"
	"github.com/pufferpanel/pufferd/v2/programs"
	"net/http"
	"strings"
)

func OAuth2Handler(requiredScope scope.Scope, requireServer bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		failure := true
		defer func() {
			if failure && !c.IsAborted() {
				c.Abort()
			}
		}()

		authHeader := c.Request.Header.Get("Authorization")
		var authToken string
		if authHeader == "" {
			authToken = c.Query("accessToken")
			if authToken == "" {
				response.HandleError(c, pufferd.ErrMissingAccessToken, http.StatusBadRequest)
				return
			}
		} else {
			authArr := strings.SplitN(authHeader, " ", 2)
			if len(authArr) < 2 || authArr[0] != "Bearer" {
				response.HandleError(c, pufferd.ErrNotBearerToken, http.StatusBadRequest)
				return
			}
			authToken = authArr[1]
		}

		var err error
		key := pufferd.GetPublicKey()
		if key == nil {
			key, err = pufferd.LoadPublicKey()
			if response.HandleError(c, err, http.StatusInternalServerError) {
				return
			}
		}

		token, err := apufferi.ParseToken(key, authToken)
		if response.HandleError(c, err, http.StatusForbidden) {
			return
		}

		serverId := c.Param("id")
		scopes := make([]scope.Scope, 0)
		if token.Claims.PanelClaims.Scopes[serverId] != nil {
			scopes = append(scopes, token.Claims.PanelClaims.Scopes[serverId]...)
		}
		if token.Claims.PanelClaims.Scopes[""] != nil {
			scopes = append(scopes, token.Claims.PanelClaims.Scopes[""]...)
		}

		if !apufferi.ContainsScope(scopes, requiredScope) {
			response.HandleError(c, pufferd.CreateErrMissingScope(requiredScope), http.StatusForbidden)
			return
		}

		if requireServer {
			program, _ := programs.Get(serverId)
			if program == nil {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			c.Set("server", program)
		}

		c.Set("scopes", scopes)

		failure = false
	}
}
