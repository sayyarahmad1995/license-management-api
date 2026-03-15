package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"license-management-api/internal/config"
)

type TokenService interface {
	GenerateAccessToken(userID int, email, role string) (string, error)
	GenerateRefreshToken(userID int, email, role string) (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (*TokenClaims, error)
}

type TokenClaims struct {
	UserID int    `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type tokenService struct {
	jwtCfg *config.JwtConfig
}

func NewTokenService(jwtCfg *config.JwtConfig) TokenService {
	return &tokenService{
		jwtCfg: jwtCfg,
	}
}

// GenerateAccessToken generates a new access token
func (ts *tokenService) GenerateAccessToken(userID int, email, role string) (string, error) {
	claims := &TokenClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ts.jwtCfg.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "license-management-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(ts.jwtCfg.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a new refresh token
func (ts *tokenService) GenerateRefreshToken(userID int, email, role string) (string, error) {
	claims := &TokenClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ts.jwtCfg.RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "license-management-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(ts.jwtCfg.RefreshTokenSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateAccessToken validates and parses an access token
func (ts *tokenService) ValidateAccessToken(token string) (*TokenClaims, error) {
	claims := &TokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(ts.jwtCfg.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ValidateRefreshToken validates and parses a refresh token
func (ts *tokenService) ValidateRefreshToken(token string) (*TokenClaims, error) {
	claims := &TokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(ts.jwtCfg.RefreshTokenSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
