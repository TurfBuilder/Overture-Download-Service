package downloader

import (
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// fetchJWKS fetches and parses a standard (RFC 7517) JSON Web Key Set from the
// given URL. The returned Keyfunc handles kid lookup and refreshes the key set
// in the background, so key rotation is picked up automatically.
func fetchJWKS(url string) (keyfunc.Keyfunc, error) {
	return keyfunc.NewDefault([]string{url})
}

func isAccessTokenValid(tokenString string, jwks keyfunc.Keyfunc) bool {
	logger.Info("Checking Access Token")

	token, err := jwt.Parse(tokenString, jwks.Keyfunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		logger.Errorw("token validation failed", "error", err)
		return false
	}

	return token.Valid
}
