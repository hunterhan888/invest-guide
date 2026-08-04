package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type JWTIssuer struct {
	secret []byte
	issuer string
	expiry time.Duration
}

func NewJWTIssuer(secret, issuer string, expiry time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), issuer: issuer, expiry: expiry}
}

func (j *JWTIssuer) Issue(userID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTIssuer) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	parser := jwt.NewParser(jwt.WithIssuer(j.issuer))
	token, err := parser.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return j.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
