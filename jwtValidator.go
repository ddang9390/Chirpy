package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type customClaims struct {
	jwt.RegisteredClaims
}

func jwtValidate(r *http.Request, secret string) (*customClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header is required")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return nil, fmt.Errorf("bearer token required")
	}
	claims := &customClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if time.Now().UTC().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf(("token expired"))
	}

	return claims, nil
}

func jwtCreation(user User, secret string) string {
	claims := customClaims{
		jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "chirpy",
			Subject:  fmt.Sprint(user.ID),
		},
	}
	expiresInSeconds := user.Expires_in_seconds

	if expiresInSeconds > 0 {
		maxExpiration := int64(24 * time.Hour.Seconds())
		if expiresInSeconds > maxExpiration {
			expiresInSeconds = maxExpiration
		}
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Duration(expiresInSeconds) * time.Second))
	} else {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(24 * time.Hour))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Println(err)
		return err.Error()
	}

	return signedToken
}

func generateRefreshToken() string {
	b := make([]byte, 32)
	ranData, err := rand.Read(b)
	if err != nil {
		fmt.Println("error:", err)
		return ""
	}

	bs := []byte(strconv.Itoa(ranData))
	secret := hex.EncodeToString(bs)
	return secret
}
