package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"9itchi-account-api/models"
	"9itchi-account-api/utils"

	"go.uber.org/zap"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool { return uuidRe.MatchString(s) }

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func badRequest(msg string) *models.ApiError {
	return &models.ApiError{Code: http.StatusBadRequest, Error: errors.New(msg)}
}

func apiError(code int, msg string) *models.ApiError {
	return &models.ApiError{Code: code, Error: errors.New(msg)}
}

// internalError logs the cause and returns a generic 500.
func internalError(op string, err error) *models.ApiError {
	utils.Logger.Error(op, zap.Error(err))
	return apiError(http.StatusInternalServerError, "internal error")
}

// validatePassword enforces min 8 chars (bcrypt caps at 72 bytes) with upper, lower and digit.
func validatePassword(p string) error {
	if len(p) < 8 || len(p) > 72 {
		return errors.New("password must be between 8 and 72 characters")
	}
	var upper, lower, digit bool
	for _, r := range p {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !upper || !lower || !digit {
		return errors.New("password must contain an uppercase letter, a lowercase letter and a digit")
	}
	return nil
}

// hashToken is the sha256 hex digest stored for refresh tokens.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// hashCode keys the digest of a 6-digit reset code with the server secret and the user id.
func hashCode(secret, userID, code string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(userID + ":" + code))
	return hex.EncodeToString(m.Sum(nil))
}
