package middleware

import (
	"QueueLite/internal/config"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func loadSecret() (string, error) {
	cfg, err := config.Load()
	secret := cfg.SecretKey
	if err != nil || secret == "" {
		return "", errors.New("invalid secret something went wrong")
	}
	return secret, nil
}

func CreateToken(username string, userid string) (string, error) {
	secret, err := loadSecret()
	if err != nil {
		return "", err
	}
	var secretKey = []byte(secret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"userid":   userid,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateToken(tokenString string) (string, error) {
	secret, err := loadSecret()
	if err != nil {
		return "", err
	}
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	userID, ok := claims["userid"].(string)
	if !ok {
		return "", fmt.Errorf("userid claim missing or not a string")
	}

	return userID, nil
}

func CreateQueueToken(username string, queueid string) (string, error) {
	secret, err := loadSecret()
	if err != nil {
		return "", err
	}
	var secretKey = []byte(secret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"queueid":  queueid,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateQueueToken(tokenString string) (string, error) {
	secret, err := loadSecret()
	if err != nil {
		return "", err
	}
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	queueID, ok := claims["queueid"].(string)
	if !ok {
		return "", fmt.Errorf("queueid claim missing or not a string")
	}

	return queueID, nil
}

func CreateGuestToken(guestID string, businessID string, phoneNumber string) (string, error) {
	secret, err := loadSecret()
	if err != nil {
		return "", err
	}
	var secretKey = []byte(secret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"guestid":     guestID,
		"businessid":  businessID,
		"phonenumber": phoneNumber,
		"exp":         time.Now().Add(config.GuestExpirationTime).Unix(),
	})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateGuestToken(tokenString string) (string, string, string, error) {
	secret, err := loadSecret()
	if err != nil {
		return "", "", "", err
	}
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", "", "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", "", fmt.Errorf("invalid token")
	}

	guestID, ok := claims["guestid"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("guestid claim missing or not a string")
	}
	businessID, ok := claims["businessid"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("businessid claim missing or not a string")
	}
	phoneNumber, ok := claims["phonenumber"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("phonenumber claim missing or not a string")
	}

	return guestID, businessID, phoneNumber, nil
}
