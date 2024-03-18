package aztokenprovider

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func Test_OBORetrieverGetExpiry(t *testing.T) {
	t.Run("returns 0 time value if retriever is nil", func(t *testing.T) {
		retriever := onBehalfOfTokenRetriever{}
		expiry := retriever.GetExpiry()

		require.Equal(t, expiry, &time.Time{})
	})
	t.Run("returns 0 time value if idToken is empty", func(t *testing.T) {
		retriever := onBehalfOfTokenRetriever{idToken: ""}
		expiry := retriever.GetExpiry()

		require.Equal(t, expiry, &time.Time{})
	})
	t.Run("returns 0 time value if it is not possible to parse JWT", func(t *testing.T) {
		retriever := onBehalfOfTokenRetriever{idToken: "fake-string"}
		expiry := retriever.GetExpiry()

		require.Equal(t, expiry, &time.Time{})
	})
	t.Run("returns 0 time value if there is no expiration time", func(t *testing.T) {
		var secretKey = []byte("secret-key")
		token := jwt.NewWithClaims(jwt.SigningMethodHS256,
			jwt.MapClaims{
				"test-claim": "test",
			})
		tokenString, err := token.SignedString(secretKey)
		if err != nil {
			t.Error(err)
		}
		retriever := onBehalfOfTokenRetriever{idToken: tokenString}
		expiry := retriever.GetExpiry()

		require.Equal(t, expiry, &time.Time{})
	})
	t.Run("returns 0 time value if expiration time is invalid", func(t *testing.T) {
		var secretKey = []byte("secret-key")
		token := jwt.NewWithClaims(jwt.SigningMethodHS256,
			jwt.MapClaims{
				"test-claim": "test",
				"exp":        "string not valid",
			})
		tokenString, err := token.SignedString(secretKey)
		if err != nil {
			t.Error(err)
		}
		retriever := onBehalfOfTokenRetriever{idToken: tokenString}
		expiry := retriever.GetExpiry()

		require.Equal(t, expiry, &time.Time{})
	})
	t.Run("returns time value if expiration time is valid", func(t *testing.T) {
		// Truncate to match the library
		expiryTime := time.Now().Truncate(time.Second)
		var secretKey = []byte("secret-key")
		token := jwt.NewWithClaims(jwt.SigningMethodHS256,
			jwt.MapClaims{
				"test-claim": "test",
				"exp":        expiryTime.Unix(),
			})
		tokenString, err := token.SignedString(secretKey)
		if err != nil {
			t.Error(err)
		}
		retriever := onBehalfOfTokenRetriever{idToken: tokenString}
		expiry := retriever.GetExpiry()

		require.Equal(t, expiry, &expiryTime)
	})
}
