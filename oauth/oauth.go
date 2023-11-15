package oauth

import (
	"context"
	"log"
	"os"

	oidc "github.com/coreos/go-oidc"
	"github.com/isotiropoulos/storage-api/models"
	"golang.org/x/oauth2"
	// "gopkg.in/square/go-jose.v2/jwt"
)

var Verifier *oidc.IDTokenVerifier
var oauth2Config oauth2.Config
var keyset oidc.KeySet

func Init() {

	log.Println("Starting oidc configuration")
	oidcProvider := os.Getenv("OIDC_PROVIDER")
	if oidcProvider == "" {
		oidcProvider = "http://localhost:30105/auth/realms/buildspace"
	}
	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		clientID = "minioapi"
	}
	clientSecret := os.Getenv("CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = "d4AvWhUKAZqdMnBVPR0dD5w5RrZfk9RC"
	}

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, oidcProvider)

	if err != nil {
		panic(err)
	}

	oauth2Config = oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "openid"},
	}

	oidcConfig := &oidc.Config{
		ClientID:          clientID,
		SkipClientIDCheck: true,
		SkipIssuerCheck:   true,
	}
	keyset = oidc.NewRemoteKeySet(ctx, oidcProvider)

	Verifier = provider.Verifier(oidcConfig)

}

func GetClaims(token string) (claims models.OidcClaims, err error) {
	resultCl := models.OidcClaims{}
	tokenVer, err := Verifier.Verify(context.Background(), token)
	erro := tokenVer.Claims(&resultCl)
	if erro != nil {
		log.Println("failed to parse Claims: ", erro.Error())
	}
	return resultCl, err
}
