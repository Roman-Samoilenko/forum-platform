package service

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func CreateJWt(login string) (string, error) {
	secretKey := os.Getenv("JWT_SECRET_KEY")

	claims := jwt.MapClaims{
		"login": login,
		"exp":   jwt.NewNumericDate(time.Now().Add(1 * time.Hour)), // Срок действия
		"iat":   jwt.NewNumericDate(time.Now()),                    // Время выпуска
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	return tokenString, err
}
