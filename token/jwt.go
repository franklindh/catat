// token/jwt_maker.go
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTMaker struct {
	signingKey []byte
}

func NewJWTMaker(signingKey string) (Maker, error) {
	if len(signingKey) == 0 {
		return nil, errors.New("signing key cannot be empty")
	}

	return &JWTMaker{
		signingKey: []byte(signingKey),
	}, nil
}

type jwtPayload struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func (maker *JWTMaker) CreateToken(userID uuid.UUID, duration time.Duration) (string, error) {
	payload, err := NewPayload(userID, duration)
	if err != nil {
		return "", err
	}

	jwtClaims := &jwtPayload{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{Time: payload.ExpiredAt},
			IssuedAt:  &jwt.NumericDate{Time: payload.IssuedAt},
			ID:        payload.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	tokenString, err := token.SignedString(maker.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (maker *JWTMaker) VerifyToken(tokenString string) (*Payload, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwtPayload{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return maker.signingKey, nil
		})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtPayload)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID.String())
	if err != nil {
		return nil, ErrInvalidToken
	}

	payload := &Payload{
		ID:        uuid.MustParse(claims.ID),
		UserID:    userID,
		IssuedAt:  claims.IssuedAt.Time,
		ExpiredAt: claims.ExpiresAt.Time,
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}
