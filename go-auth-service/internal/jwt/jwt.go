package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	RequestLimit int    `json:"request_limit"`
	Challenge    string `json:"challenge"`
	jwt.RegisteredClaims
}

func Generate(secret string, challenge string, expMinutes int, requestLimit int) (string, time.Time, error) {
	exp := time.Now().Add(time.Duration(expMinutes) * time.Minute)
	claims := Claims{
		RequestLimit: requestLimit,
		Challenge:    challenge,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        challenge,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	return signed, exp, err
}

// Verify validates a JWT token and returns the claims if valid
func Verify(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}
