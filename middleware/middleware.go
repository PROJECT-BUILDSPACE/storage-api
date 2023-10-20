package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	auth "github.com/isotiropoulos/storage-api/oauth"
	"github.com/isotiropoulos/storage-api/utils"
)

type IAuth interface {
	AuthMiddleware(h http.HandlerFunc) http.HandlerFunc
	NaiveAuthMiddleware(h http.HandlerFunc) http.HandlerFunc
}

type AuthImplementation struct {
}

func (a *AuthImplementation) AuthMiddleware(h http.HandlerFunc) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiAll := r.Header.Get("Authorization")
		if apiAll == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized.", "No Authorization header.", "MID0001")
			return
		}
		apiKeyAr := strings.Split(apiAll, " ")
		authType := apiKeyAr[0]
		if authType != "Bearer" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized.", "No Bearer token.", "MID0002")
			return
		}

		apiKey := apiKeyAr[1]
		// log.Println(apiKey)
		claims, err := auth.GetClaims(apiKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Unable to resolve claims.", err.Error(), "MID0003")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		ctx = context.WithValue(ctx, "claims", claims)
		_, err = auth.Verifier.Verify(ctx, apiKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Unable to initialize verifier.", err.Error(), "MID0004")
			return
		}

		groupID := r.Header.Get("X-Group-Id")
		fmt.Println(groupID, claims.Groups)
		if len(groupID) == 0 {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unable to resolve Group.", "No group present.", "MID0005")
			return
		} else {
			if !utils.ItemInArray(claims.Groups, groupID) {
				utils.RespondWithError(w, http.StatusForbidden, "Permission Denied.", "No permission rights for user in group.", "MID0006")
				return
			}
		}

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *AuthImplementation) NaiveAuthMiddleware(h http.HandlerFunc) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiAll := r.Header.Get("Authorization")
		if apiAll == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized.", "No Authorization header.", "MID0007")
			return
		}
		apiKeyAr := strings.Split(apiAll, " ")
		authType := apiKeyAr[0]
		if authType != "Bearer" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized.", "No Bearer token.", "MID0008")
			return
		}

		apiKey := apiKeyAr[1]
		// log.Println(apiKey)
		claims, err := auth.GetClaims(apiKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Unable to resolve claims.", err.Error(), "MID0009")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		ctx = context.WithValue(ctx, "claims", claims)
		_, err = auth.Verifier.Verify(ctx, apiKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Unable to initialize verifier.", err.Error(), "MID0010")
			return
		}

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
