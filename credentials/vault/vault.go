package vault

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"
)

type ServiceAccount struct {
	Client *api.Client
}

type Config struct {
	Address   string
	AuthPath  string `default:"auth/kubernetes/login"`
	Role      string
	Autorenew bool
}

func NewVaultClient(logger log.Logger, config Config) (*api.Client, error) {
	logger = logger.With(
		"context", "appkit/credentials/vault",
		"address", config.Address,
		"auth_path", config.AuthPath,
		"role", config.Role,
	)

	logger.Debug().Log(
		"msg", "creating vault client",
	)
	////////////////////////////////////////
	// VAULT

	cfg := api.Config{
		Address: config.Address,
	}

	tokenFilename := "/var/run/secrets/kubernetes.io/serviceaccount/token"

	tokBytes, err := ioutil.ReadFile(tokenFilename)
	if os.IsNotExist(err) {
		logger.Info().Log("msg", "no token file, returning nil client")
		return nil, nil
	}

	client, err := api.NewClient(&cfg)

	if err != nil {
		return nil, errors.Wrap(err, "error in vault/api.NewClient")
	}

	if config.Autorenew {
		go autorenewAuthentication(client, string(tokBytes), config, logger)
	} else {
		_, err = fetchAuthToken(client, string(tokBytes), config, logger)
	}

	return client, err
}

func autorenewAuthentication(client *api.Client, token string, config Config, logger log.Logger) {
	logger = logger.With("autorenew", true)
	logger.Info().Log("msg", "starting automatic vault authentication renewal")

	sleeper := backoff.NewExponentialBackOff()
	sleeper.MaxInterval = 30 * time.Second

	for {
		var secret *api.Secret

		op := func() error {
			s, err := fetchAuthToken(client, token, config, logger)
			secret = s
			return err
		}

		notify := func(err error, next time.Duration) {
			logger.Error().Log(
				"msg", fmt.Sprintf("failed to authenticate with vault, will try again in %v: %v", next, err),
				"next_backoff", next,
				"err", err,
			)
		}

		err := backoff.RetryNotify(op, sleeper, notify)
		if err != nil {
			// There's no timeout on the sleeper, so if we get an error here, what can we do?
			logger.Crit().Log(
				"msg", fmt.Sprintf("error in backoff when authenticating with vault: %v", err),
				"err", err,
			)
		} else if secret == nil {
			logger.Crit().Log(
				"msg", "backoff returned no error, but we don't have a secret",
			)
		}

		l := logger.With("accessor", secret.Auth.Accessor)

		if secret.Auth.Renewable {

			// api.Client.NewRenewer only returns an error if
			// parameter is nil or parameter's secret is nil.
			renewer, _ := client.NewRenewer(&api.RenewerInput{
				Secret: secret,
			})

			go renewer.Renew()

			for {
				select {
				case err := <-renewer.DoneCh():
					if err != nil {
						l.WithError(errors.Wrap(err, "error renewing vault authentication")).Log()
					} else {
						l.Warn().Log(
							"msg", "halting vault authentication autorenewal",
						)
					}
					break

				case renewal := <-renewer.RenewCh():
					l.Info().Log(
						"msg", fmt.Sprintf("renewed vault authentication at %s", renewal.RenewedAt),
						"renewed_at", renewal.RenewedAt,
					)
					LogWarnings(renewal.Secret, logger)
				}
			}
		}
	}
}

func fetchAuthToken(client *api.Client, token string, config Config, logger log.Logger) (*api.Secret, error) {
	logger.Debug().Log("msg", "authenticating with vault")
	authReq := map[string]interface{}{
		"jwt":  token,
		"role": config.Role,
	}

	s, err := client.Logical().Write(config.AuthPath, authReq)

	if err != nil {
		return s, errors.Wrap(err, "error authenticating with vault")
	}

	logger = logger.With("request_id", s.RequestID)

	if s.Auth == nil {
		logger.Warn().Log("msg", "vault auth path didn't return auth data")
	} else {
		expiry := time.Now().Add(time.Duration(s.Auth.LeaseDuration) * time.Second)
		logger.Info().Log(
			"msg", fmt.Sprintf("authenticated with vault, lease expires at %s", expiry),
			"accessor", s.Auth.Accessor,
			"policies", strings.Join(s.Auth.Policies, ","),
			"lease_duration", s.Auth.LeaseDuration,
			"renewable", s.Auth.Renewable,
		)
	}

	LogWarnings(s, logger)

	client.SetToken(s.Auth.ClientToken)

	return s, nil
}

func LogWarnings(s *api.Secret, logger log.Logger) {
	if len(s.Warnings) > 0 {
		l := logger.Warn()
		for _, w := range s.Warnings {
			l.Log("msg", fmt.Sprintf("vault api warning: %s", w))
		}
	}
}