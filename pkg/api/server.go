package api

import (
	"context"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
	"user.app/message"
	"user.app/models"
	"user.app/pkg/auth"
	"user.app/pkg/conn"
	"user.app/pkg/constants"
	"user.app/pkg/query"
)

var (
	ErrInternalServer     = status.Error(codes.Internal, "internal server error")
	ErrUserIDNotAvailable = status.Error(codes.Internal, "user id not available")
)

// MDGet returns the metadata object present in the incoming context
func MDGet(ctx context.Context) (metadata.MD, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		return md, nil
	}
	log.Error().Msg("unable to get metadata from the context")
	return nil, status.Error(codes.Internal, "Internal server error")
}

// MDGetUserIsServiceAccount returns the user, is service account or not
// present in the context which is supposed to be the logged-in user's
func MDIsSuperUser(ctx context.Context) (bool, error) {
	is, err := MDGetValue(ctx, constants.MDKeySuperUser)
	if err != nil {
		return false, err
	}

	isBool, err := strconv.ParseBool(is)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse is_service_account metadata")
		return false, err
	}

	return isBool, nil
}

// MDGetUserID returns the user id present in the context which is supposed to be the logged-in user's id
func MDGetUserID(ctx context.Context) (string, error) {
	return MDGetValue(ctx, constants.MDKeyUserID)
}

// MDGetValue find the value for which key has been passed in the arguments from the provided context
func MDGetValue(ctx context.Context, key string) (string, error) {
	md, err := MDGet(ctx)
	if err != nil {
		return "", err
	}

	if values := md.Get(key); len(values) > 0 {
		return values[0], nil
	}
	log.Error().Msgf("error while getting %s from metadata", key)

	return "", status.Error(codes.Internal, "Internal server error")
}

type Server struct {
}

func (*Server) SignIn(ctx context.Context, req *message.AuthRequest) (*message.AuthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request sent")
	}
	userDetails, err := auth.Authenticator.Authenticate(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("cannot authenticate the user")
		if err == auth.ErrInvalidCredentialsError {
			return nil, status.Error(codes.Unauthenticated, "Invalid credentials error")
		}
		return nil, status.Error(codes.Internal, "internal server error")
	}

	token, err := auth.Authenticator.EncodeToken(userDetails.User, userDetails.ClaimType, userDetails.SessionID)
	if err != nil {
		log.Error().Err(err).Msg("unable to encode jwt")
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &message.AuthResponse{
		Username: userDetails.User.Username,
		Token:    token.AccessToken,
	}, nil
}

func (*Server) SignOut(ctx context.Context, _ *message.Empty) (*message.Empty, error) {
	err := auth.Authenticator.InvalidateSession(ctx)
	if err != nil {
		log.Error().Err(err).Msg("cannot invalidate the session")
		return nil, ErrInternalServer
	}

	return &message.Empty{}, err
}

func (*Server) CreateUser(ctx context.Context, req *message.CreateUserRequest) (*message.CreateUserResponse, error) {
	hashedPW, err := auth.PasslibCtx.Hash(req.Password)
	if err != nil {
		log.Error().Err(err).Msg("cannot hash the password")
		return nil, ErrInternalServer
	}

	transaction, err := conn.Instance.Begin()
	if err != nil {
		log.Error().Err(err).Msg("cannot begin the transaction")
		return nil, ErrInternalServer
	}

	newUser := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPW,
	}

	err = query.CreateUser(ctx, transaction, newUser)
	if err != nil {
		_ = transaction.Rollback()
		log.Error().Err(err).Msg("cannot create the user")
		return nil, ErrInternalServer
	}
	_ = transaction.Commit()

	return &message.CreateUserResponse{
		UserId: newUser.ID,
	}, err
}

func (*Server) UpdateUser(ctx context.Context, req *message.UpdateUserRequest) (*message.Empty, error) {
	userID, err := MDGetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "userID is not present in the request")
	}

	if len(req.Username) == 0 && len(req.Email) == 0 {
		return nil, status.Error(codes.InvalidArgument, "noting to update")
	}

	transaction, err := conn.Instance.Begin()
	if err != nil {
		log.Error().Err(err).Msg("cannot begin the transaction")
		return nil, ErrInternalServer
	}

	user, err := query.FindUser(ctx, transaction, &query.UserFilter{
		ID: userID,
	})
	if err != nil {
		log.Error().Err(err).Msg("cannot retrieve the user")
		_ = transaction.Rollback()
		return nil, ErrInternalServer
	}

	var columnsUpdated []string

	if req.Email != user.Email {
		user.Email = req.Email
		columnsUpdated = append(columnsUpdated, models.UserColumns.Email)
	}

	if req.Username != user.Username {
		user.Username = req.Username
		columnsUpdated = append(columnsUpdated, models.UserColumns.Username)
	}

	err = query.UpdateUser(ctx, transaction, user, columnsUpdated)
	if err != nil {
		_ = transaction.Rollback()
		log.Error().Err(err).Msg("cannot update the user")
		return nil, ErrInternalServer
	}
	_ = transaction.Commit()

	return &message.Empty{}, err
}

func (*Server) DeleteUser(ctx context.Context, _ *message.Empty) (*message.Empty, error) {
	userID, err := MDGetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "userID is not present in the request")
	}

	transaction, err := conn.Instance.Begin()
	if err != nil {
		log.Error().Err(err).Msg("cannot begin the transaction")
		return nil, ErrInternalServer
	}

	user, err := query.FindUser(ctx, transaction, &query.UserFilter{
		ID: userID,
	})
	if err != nil {
		log.Error().Err(err).Msg("cannot retrieve the user")
		_ = transaction.Rollback()
		return nil, ErrInternalServer
	}

	var columnsUpdated []string

	user.DeletedAt = time.Now()
	columnsUpdated = append(columnsUpdated, models.UserColumns.DeletedAt)

	err = query.UpdateUser(ctx, transaction, user, columnsUpdated)
	if err != nil {
		_ = transaction.Rollback()
		log.Error().Err(err).Msg("cannot update the user")
		return nil, ErrInternalServer
	}
	_ = transaction.Commit()

	err = auth.Authenticator.InvalidateSession(ctx)
	if err != nil {
		log.Error().Err(err).Msg("cannot invalidate the session")
		return nil, ErrInternalServer
	}

	return &message.Empty{}, err
}

// AuthFuncOverride This will bypass on method matching allowedFunc
func (s *Server) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	if fullMethodName != "/message.UserApp/CreateUser" && fullMethodName != "/message.UserApp/SignIn" {
		return auth.Authenticator.VerifyCredentials(ctx)
	}
	return ctx, nil
}
