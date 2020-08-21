package auth

import (
	"context"
	"encoding/json"
	"github.com/allegro/bigcache"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"time"
)

type (
	// ClaimType is a string type that will be used to store the type of the Claim present in the Mission Control JWT
	ClaimType string
	// Session stores the session information for a user
	Session struct {
		ID        string
		UserID    string
		ClaimType ClaimType
		SuperUser bool
	}

	// SessionRegistry handles the entire session's life cycle
	SessionRegistry interface {

		// AddToSessionRegistry adds a session to the session registry
		AddToSessionRegistry(ctx context.Context, session Session) (string, error)

		// UpdateSessionRegistry updates a session present in the session registry
		UpdateSessionRegistry(ctx context.Context, session Session) error

		// GetSessionRegistry retrieves a session from the registry using the claim
		GetSessionRegistry(ctx context.Context, claim UserClaims) (Session, error)

		// EvictFromSessionRegistry removes a session from the session registry
		EvictFromSessionRegistry(ctx context.Context, claim UserClaims) error
	}

	// InMemSessionRegistry is an implementation of SessionRegistry that uses In memory cache, BigCache, to handle the
	// sessions
	InMemSessionRegistry struct {
		sessionRegistry *bigcache.BigCache
	}
)

// NewInMemSessionRegistry is the constructor for the InMemorySessionRegistry
func NewInMemSessionRegistry(jwtLifeTimeInHours int) (SessionRegistry, error) {
	sessionRegistry, err := bigcache.NewBigCache(
		bigcache.DefaultConfig(time.Duration(jwtLifeTimeInHours) * time.Hour),
	)
	if err != nil {
		return nil, err
	}
	return &InMemSessionRegistry{
		sessionRegistry: sessionRegistry,
	}, nil
}

// AddToSessionRegistry add a session object to the registry. It returns the session ID of the session that has been
// added.
func (j *InMemSessionRegistry) AddToSessionRegistry(ctx context.Context, session Session) (string, error) {
	sessionID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "cannot create the UUID")
	}
	session.ID = sessionID.String()

	registry := j.sessionRegistry

	sessionInJSON, err := json.Marshal(session)
	if err != nil {
		return "", errors.Wrap(err, "cannot marshal to json")
	}

	if err = registry.Set(sessionID.String(), sessionInJSON); err != nil {
		return "", errors.Wrap(err, "cannot add to the session registry")
	}
	return sessionID.String(), nil
}

// UpdateSessionRegistry updates the session present in the registry the where the session ID will be taken from the
// session object parameter
func (j *InMemSessionRegistry) UpdateSessionRegistry(ctx context.Context, session Session) error {
	registry := j.sessionRegistry

	sessionInJSON, err := json.Marshal(session)
	if err != nil {
		return errors.Wrap(err, "error while marshalling the Session object")
	}

	if err = registry.Set(session.ID, sessionInJSON); err != nil {
		return errors.Wrap(err, "error while updating to the Session registry")
	}
	return nil
}

// GetSessionRegistry fetches the session object for the claim
func (j *InMemSessionRegistry) GetSessionRegistry(ctx context.Context, claim UserClaims) (Session, error) {
	registry := j.sessionRegistry

	cacheValue, err := registry.Get(claim.SessionID)
	if err != nil {
		if err == bigcache.ErrEntryNotFound {
			return Session{}, ErrInvalidCredentialsError
		}
		return Session{}, errors.Wrap(err, "cannot get the session from registry")
	}

	var sessionObj Session
	err = json.Unmarshal(cacheValue, &sessionObj)
	if err != nil {
		return Session{}, errors.Wrap(err, "error while Session json unmarshal")
	}

	return sessionObj, err
}

// EvictFromSessionRegistry removes the session from the registry for the provided claim
func (j *InMemSessionRegistry) EvictFromSessionRegistry(ctx context.Context, claim UserClaims) error {
	registry := j.sessionRegistry
	if err := registry.Delete(claim.SessionID); err != nil {
		return errors.Wrap(err, "error while session evict")
	}
	return nil
}
