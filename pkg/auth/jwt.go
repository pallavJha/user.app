package auth

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gopkg.in/hlandau/passlib.v1"
	"gopkg.in/hlandau/passlib.v1/abstract"
	"gopkg.in/hlandau/passlib.v1/hash/argon2"
	"strconv"
	"time"
	"user.app/message"
	"user.app/models"
	"user.app/pkg/conn"
	"user.app/pkg/constants"
	"user.app/pkg/query"
)

var (
	ErrInvalidCredentialsError = errors.New("Invalid credentials")
	PasslibCtx                 = passlib.Context{
		Schemes: []abstract.Scheme{
			argon2.New(1, 32*1024, 4),
		},
	}
	// Authenticator is global authenticator.
	Authenticator *JWT
)

const (
	// Internal is the claim type used in direct login JWT
	Internal ClaimType = "Internal"
	// DefaultLifetime is the expiry time in hours for the MissionControl JWT
	DefaultLifetime int = 72
	SigningKey          = "&7!iArdsCHST$%NgH52KLRGa2CTmG0ik"
)

type (

	// Token response body for JWT
	Token struct {
		AccessToken string `json:"access_token"`
	}

	// UserClaims struct as base for JWT payload
	UserClaims struct {
		ID        string
		Username  string
		ClaimType ClaimType
		SessionID string
		jwt.StandardClaims
	}

	// JWT implements auth.IAuth, and used for encoding and validating JWT Token.
	JWT struct {
		registry SessionRegistry
	}

	// UserSessionDetail holds the session details for a user
	UserSessionDetail struct {
		User      *models.User
		ClaimType ClaimType
		SessionID string
	}
)

// UnaryServerInterceptor registers as an unary server interceptor
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return grpc_auth.UnaryServerInterceptor(nil)
}


// NewJWTAuth is the constructor for the JWT
func NewJWTAuth() (*JWT, error) {
	jwtLifeTimeInHours := DefaultLifetime

	sessionProvider, err := NewInMemSessionRegistry(jwtLifeTimeInHours)
	if err != nil {
		return nil, err
	}

	return &JWT{
		registry: sessionProvider,
	}, nil
}

// EncodeToken creates a JWT token for a user argument for subsequent call verification.
func (j *JWT) EncodeToken(user *models.User, claimType ClaimType, sessionID string) (*Token, error) {
	claims := j.prepareClaims(user, claimType, sessionID)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(SigningKey))

	if err != nil {
		return nil, errors.WithMessage(err, "unable to generate signed string from token.")
	}

	return &Token{
		AccessToken: accessToken,
	}, nil
}

// prepareClaims creates a UserClaim object with expiry time
func (j *JWT) prepareClaims(user *models.User, claimType ClaimType, sessionID string) UserClaims {
	currentTime := time.Now()
	jwtLifeTimeInHours := DefaultLifetime
	expireTime := currentTime.Add(time.Hour * time.Duration(jwtLifeTimeInHours)).Unix()

	// Create the UserClaims
	claims := UserClaims{
		ID:        user.ID,
		Username:  user.Username,
		ClaimType: claimType,
		SessionID: sessionID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime,
			Issuer:    "bob.dylan",
		},
	}
	return claims
}

// decodeJWT decodes a JWT token argument into UserClaims object
func (j *JWT) decodeJWT(tokenString string) (*UserClaims, error) {
	userClaims := UserClaims{}
	// Parse the token
	tokenType, err := jwt.ParseWithClaims(tokenString, &userClaims, func(token *jwt.Token) (interface{}, error) {
		return []byte(SigningKey), nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "error while parsing jwt token")
	}

	// Validate the token and return the custom user claims
	if claims, ok := tokenType.Claims.(*UserClaims); ok && tokenType.Valid {
		return claims, nil
	}
	return nil, err
}

// VerifyCredentials called on each and every request made.
// It decodes the JWT token and puts user_id, name and email as principal in the context.
func (j *JWT) VerifyCredentials(grpcCtx context.Context) (context.Context, error) {
	var (
		err         error
		tokenString string
		userClaims  *UserClaims
	)

	tokenString, err = grpc_auth.AuthFromMD(grpcCtx, "bearer")
	if err != nil {
		return nil, errors.Wrap(err, "unable to extract token from request header")
	}

	userClaims, err = j.decodeJWT(tokenString)
	if err != nil {
		return nil, ErrInvalidCredentialsError
	}

	session, err := j.registry.GetSessionRegistry(grpcCtx, *userClaims)
	if err != nil {
		return nil, err
	}

	if session.ClaimType != Internal {
		// handle other ways of authentication
	}

	md, _ := metadata.FromIncomingContext(grpcCtx)
	md.Set(constants.MDKeyUserID, userClaims.ID)
	md.Set(constants.MDKeyUsername, userClaims.Username)
	md.Set(constants.MDKeySuperUser, strconv.FormatBool(session.SuperUser))
	return metadata.NewIncomingContext(grpcCtx, md), nil
}

// InvalidateSession evicts the current session from the session registry
func (j *JWT) InvalidateSession(ctx context.Context) error {
	var (
		err         error
		tokenString string
		userClaims  *UserClaims
	)

	tokenString, err = grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		log.Error().Err(err).Msg("unable to extract token from request header")
		return ErrInvalidCredentialsError
	}

	userClaims, err = j.decodeJWT(tokenString)
	if err != nil {
		log.Error().Err(err).Msg("unable to decode JWT")
		return ErrInvalidCredentialsError
	}

	return j.registry.EvictFromSessionRegistry(ctx, *userClaims)
}

// Authenticate creates a session for a user whose authentication request is present in the parameter
// the authentication process is delegated to function handleUsernamePasswordAuthentication
func (j *JWT) Authenticate(ctx context.Context, req *message.AuthRequest) (UserSessionDetail, error) {
	var user *models.User
	var err error
	var sessionObj Session

	user, err = j.handleUsernamePasswordAuthentication(ctx, req.Username, req.Password)
	if err != nil {
		return UserSessionDetail{}, err
	}
	sessionObj.ClaimType = Internal
	sessionObj.UserID = user.ID
	if user.IsSuperuser.Bool {
		sessionObj.SuperUser = true
	}

	sessionID, err := j.registry.AddToSessionRegistry(ctx, sessionObj)
	if err != nil {
		return UserSessionDetail{}, err
	}
	return UserSessionDetail{
		User:      user,
		ClaimType: sessionObj.ClaimType,
		SessionID: sessionID,
	}, nil
}

func (j *JWT) handleUsernamePasswordAuthentication(ctx context.Context, username, password string) (*models.User, error) {
	var err error

	user, err := query.FindUser(ctx, conn.Instance, &query.UserFilter{
		Username: username,
	})
	if err != nil {
		switch err {
		case query.ErrNoRowsFound:
			return nil, ErrInvalidCredentialsError
		}
		return nil, err
	}

	if err = PasslibCtx.VerifyNoUpgrade(password, user.Password); err != nil {
		log.Error().Err(err).Msg("supplied password does not match stored password")
		return nil, ErrInvalidCredentialsError
	}
	return user, nil
}
