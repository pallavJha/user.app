package query

import (
	"context"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/reiver/go-pqerror"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"time"
	"user.app/models"
)

var (
	ErrMultipleUsersWithSameUsernameEmail = errors.New("multiple users with the same user and Email found")
	ErrNoRowsFound                        = errors.New("no rows were found")
)

func CreateUser(ctx context.Context, exec boil.ContextExecutor, user *models.User) error {
	err := user.Insert(ctx, exec, boil.Infer())
	if err != nil {
		log.Error().Err(err).Msg("insertion failed")

		pqError, ok := errors.Cause(err).(*pq.Error)
		if ok {
			switch pqError.Code {
			case pqerror.CodeIntegrityConstraintViolationUniqueViolation:
				log.Error().Str("name", "already exists").Str("Email", "already exists").Msg("insertion failed")
				return errors.Wrap(err, "unique constraint violation for name and Email")
			}
		}
		return err
	}

	log.Info().Msg("inserted successfully")
	return nil
}

type UserFilter struct {
	Username string
	Email    string
	ID       string
}

func (f *UserFilter) apply(mod []qm.QueryMod) []qm.QueryMod {
	if f == nil {
		return mod
	}
	if len(f.Username) > 0 {
		mod = append(mod, models.UserWhere.Username.EQ(f.Username))
	}
	if len(f.ID) > 0 {
		mod = append(mod, models.UserWhere.ID.EQ(f.ID))
	}
	if len(f.Email) > 0 {
		mod = append(mod, models.UserWhere.Username.EQ(f.Email))
	}

	return mod
}

func FindUser(ctx context.Context, exec boil.ContextExecutor, filter *UserFilter) (*models.User, error) {
	var queryMod = []qm.QueryMod{
		models.UserWhere.DeletedAt.EQ(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	if filter != nil {
		queryMod = filter.apply(queryMod)
	}

	userSlice, err := models.Users(
		queryMod...,
	).All(ctx, exec)

	if err != nil {
		log.Error().Msg("retrieval failed")
		return nil, err
	}

	if len(userSlice) == 0 {
		log.Error().Msg("no results")
		return nil, ErrNoRowsFound
	}

	if len(userSlice) > 1 {
		log.Error().Msg("multiple users found")
		return nil, ErrMultipleUsersWithSameUsernameEmail
	}

	log.Info().Msg("retrieval successful")
	return userSlice[0], nil
}

func UpdateUser(ctx context.Context, exec boil.ContextExecutor, user *models.User, columnsUpdated []string) error {
	_, err := user.Update(ctx, exec, boil.Whitelist(append(columnsUpdated, "updated_at")...))
	if err != nil {
		log.Error().Err(err).Msg("update failed")

		pqError, ok := errors.Cause(err).(*pq.Error)
		if ok {
			switch pqError.Code {
			case pqerror.CodeIntegrityConstraintViolationUniqueViolation:
				log.Error().Str("name", "already exists").Str("Email", "already exists").Msg("insertion failed")
				return errors.Wrap(err, "unique constraint violation for name and Email")
			}
		}
		return err
	}

	log.Info().Msg("updated successfully")
	return nil
}
