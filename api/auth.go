package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	db "github.com/franklindh/catat/db/sqlc"
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
	frontendURL := s.config.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	redirectWithError := func(message string) {
		redirectURL := fmt.Sprintf("%s/auth/callback?error=%s", frontendURL, url.QueryEscape(message))
		ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}

	code := ctx.Query("code")
	if code == "" {
		redirectWithError("Kode autentikasi tidak ditemukan")
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
		redirectWithError("Gagal memverifikasi dengan Google")
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		redirectWithError("Gagal mengambil data dari Google")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		redirectWithError("Gagal membaca data user")
		return
	}

	var userInfo googleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		redirectWithError("Gagal memproses data user")
		return
	}

	googleAuthID := pgtype.Text{String: userInfo.ID, Valid: true}

	user, err := s.store.GetUserByGoogleAuthID(ctx, googleAuthID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			arg := db.CreateUserParams{
				Email:        userInfo.Email,
				Name:         pgtype.Text{String: userInfo.Name, Valid: true},
				AvatarUrl:    pgtype.Text{String: userInfo.Picture, Valid: true},
				GoogleAuthID: googleAuthID,
			}

			_, createErr := s.store.CreateUser(ctx, arg)
			if createErr != nil {
				if pqErr, ok := createErr.(*pgconn.PgError); ok && pqErr.Code == "23505" {
					redirectWithError("Email sudah terdaftar")
					return
				}
				redirectWithError("Gagal membuat akun")
				return
			}
			user, err = s.store.GetUserByGoogleAuthID(ctx, googleAuthID)
			if err != nil {
				redirectWithError("Gagal mengambil data akun")
				return
			}

			// Create default categories for new user
			if seedErr := s.store.CreateDefaultCategories(ctx, user.ID); seedErr != nil {
				// Log error but don't fail registration
				fmt.Printf("Warning: failed to create default categories for user %d: %v\n", user.ID, seedErr)
			}
		} else {
			redirectWithError("Gagal mengambil data akun")
			return
		}
	}

	accessToken, err := s.tokenMaker.CreateToken(user.ID, user.Role, s.config.AccessTokenDuration)
	if err != nil {
		redirectWithError("Gagal membuat sesi")
		return
	}

	redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s", frontendURL, url.QueryEscape(accessToken))
	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
