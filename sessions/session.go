package sessions

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/gorilla/sessions"
)

type session struct {
	w      http.ResponseWriter
	r      *http.Request
	config *Config
	store  *sessions.CookieStore
}

// Config the necessary configs for session
type Config struct {
	Name   string
	Key    string
	MaxAge int
	Secure bool
}

type sessionContextKey int

const sessionCtxKey sessionContextKey = iota

// WithSession is middleware to generate a session store for the whole request lifetime.
// later session operations should call `Get`/`Put`/`Del` to work with the session
func WithSession(conf *Config) func(http.Handler) http.Handler {
	store := setupSessionStore(conf)
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			se := newSession(w, r, conf, store)

			// Record session storer in context
			ctx := context.WithValue(r.Context(), sessionCtxKey, se)

			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		})
	}
}

// Get retrieves the value of the given key from the session.
func Get(ctx context.Context, key string) (string, error) {
	s, err := getSession(ctx)
	if err != nil {
		return "", err
	}

	session, err := s.store.Get(s.r, s.config.Name)
	if err != nil {
		return "", errors.Wrapf(err, "cannot get key %v from session", key)
	}

	strInf, ok := session.Values[key]
	if !ok {
		return "", fmt.Errorf("Cannot find value for: %s", key)
	}

	str, ok := strInf.(string)
	if !ok {
		return "", fmt.Errorf("The value is not a string: %+v", strInf)
	}

	return str, nil
}

// Put adds value for key into the session
func Put(ctx context.Context, key, value string) error {
	s, err := getSession(ctx)
	if err != nil {
		return err
	}

	session, err := s.store.Get(s.r, s.config.Name)
	if err != nil {
		return errors.Wrapf(err, "cannot put key %v in to session", key)
	}

	session.Values[key] = value
	session.Save(s.r, s.w)
	return nil
}

// Del deletes value from the session by given key
func Del(ctx context.Context, key string) error {
	s, err := getSession(ctx)
	if err != nil {
		return err
	}

	session, err := s.store.Get(s.r, s.config.Name)
	if err != nil {
		return errors.Wrapf(err, "cannot delete key %v from session", key)
	}

	delete(session.Values, key)
	session.Save(s.r, s.w)

	return nil
}

func setupSessionStore(config *Config) *sessions.CookieStore {
	if config.Name == "" {
		panic("session name must be present")
	}

	if _, err := base64.StdEncoding.DecodeString(config.Key); err != nil {
		panic(err)
	}

	csConfig := CookieStoreConfig{
		Name:       config.Name,
		Key:        config.Key,
		NoHTTPOnly: false,
		NoSecure:   !config.Secure,
		MaxAge:     config.MaxAge,
	}

	return NewCookieStore(csConfig)
}

func newSession(w http.ResponseWriter, r *http.Request, config *Config, sessionStore *sessions.CookieStore) *session {
	return &session{w, r, config, sessionStore}
}

func getSession(ctx context.Context) (*session, error) {
	if se, ok := ctx.Value(sessionCtxKey).(*session); ok {
		return se, nil
	}

	return nil, errors.New("cannot get session from the context, please make sure the WithSession middleware has been executed before calling this function")
}
