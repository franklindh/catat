// api/auth.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (s *Server) googleOAuthLogin(ctx *gin.Context) {
	state := uuid.New().String()

	googleOauthConfig := &oauth2.Config{
		ClientID:     s.config.GoogleOAuthClientID,
		ClientSecret: s.config.GoogleOAuthClientSecret,
		RedirectURL:  s.config.GoogleOAuthRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	authURL := googleOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	fmt.Printf("DEBUG: Raw auth URL: %s\n", authURL)

	ctx.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

func (s *Server) googleOAuthCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	if code == "" {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("code parameter is required")))
		return
	}

	googleOauthConfig := &oauth2.Config{
		ClientID:     s.config.GoogleOAuthClientID,
		ClientSecret: s.config.GoogleOAuthClientSecret,
		RedirectURL:  s.config.GoogleOAuthRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(fmt.Errorf("failed to exchange code: %w", err)))
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(fmt.Errorf("failed to get user info: %w", err)))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(fmt.Errorf("failed to read user info response: %w", err)))
		return
	}

	var userInfo googleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(fmt.Errorf("failed to parse user info: %w", err)))
		return
	}

	user, err := s.store.GetUserByGoogleID(ctx, userInfo.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var balance pgtype.Numeric
			_ = balance.Scan("0.0000")

			arg := db.CreateUserParams{
				GoogleID:  userInfo.ID,
				Email:     userInfo.Email,
				Name:      pgtype.Text{String: userInfo.Name, Valid: true},
				AvatarUrl: pgtype.Text{String: userInfo.Picture, Valid: true},
				Balance:   balance,
			}

			newUser, createErr := s.store.CreateUser(ctx, arg)
			if createErr != nil {
				if pqErr, ok := createErr.(*pgconn.PgError); ok && pqErr.Code == "23505" {
					ctx.JSON(http.StatusForbidden, util.ErrorResponse(errors.New("email already registered")))
					return
				}
				ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(createErr))
				return
			}
			user = newUser
		} else {
			ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
			return
		}
	}

	accessToken, err := s.tokenMaker.CreateToken(
		util.PgxUUIDToGoogleUUID(user.ID),
		s.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := gin.H{
		"access_token": accessToken,
		"user":         newUserResponse(user),
	}

	fmt.Println(user)
	ctx.JSON(http.StatusOK, rsp)
}
