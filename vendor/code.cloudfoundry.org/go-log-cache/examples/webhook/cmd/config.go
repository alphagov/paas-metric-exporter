package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	envstruct "code.cloudfoundry.org/go-envstruct"
	"google.golang.org/grpc/credentials"
)

// Config is the configuration for a LogCache Gateway.
type Config struct {
	LogCacheAddr string `env:"LOG_CACHE_ADDR, required"`
	HealthPort   int    `env:"HEALTH_PORT"`
	TLS          TLS

	GroupPrefix string `env:"GROUP_PREFIX"`

	// Encoded as SourceID=TemplatePath
	TemplatePaths []templateInfo `env:"TEMPLATE_PATHS"`

	// Encoded as SourceID=TemplatePath
	FollowTemplatePaths []templateInfo `env:"FOLLOW_TEMPLATE_PATHS"`
}

type TLS struct {
	CAPath   string `env:"CA_PATH,   required"`
	CertPath string `env:"CERT_PATH, required"`
	KeyPath  string `env:"KEY_PATH,  required"`
}

func (t TLS) Credentials(cn string) credentials.TransportCredentials {
	creds, err := NewTLSCredentials(t.CAPath, t.CertPath, t.KeyPath, cn)
	if err != nil {
		log.Fatalf("failed to load TLS config: %s", err)
	}

	return creds
}

func NewTLSCredentials(
	caPath string,
	certPath string,
	keyPath string,
	cn string,
) (credentials.TransportCredentials, error) {
	cfg, err := NewTLSConfig(caPath, certPath, keyPath, cn)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(cfg), nil
}

func NewTLSConfig(caPath, certPath, keyPath, cn string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		ServerName:         cn,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
	}

	caCertBytes, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertBytes); !ok {
		return nil, errors.New("cannot parse ca cert")
	}

	tlsConfig.RootCAs = caCertPool

	return tlsConfig, nil
}

// LoadConfig creates Config object from environment variables
func LoadConfig() (*Config, error) {
	c := Config{
		HealthPort: 6063,
	}

	if err := envstruct.Load(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

type templateInfo struct {
	SourceIDs    []string
	TemplatePath string
}

// UnmarshalEnv implements envstruct.Unmarshaller. It expects the data to be
// of the form: SourceID=TemplatePath
func (i *templateInfo) UnmarshalEnv(s string) error {
	if s == "" {
		return nil
	}

	r := strings.Split(s, "=")
	if len(r) != 2 {
		return fmt.Errorf("%s is not of valid form. (SourceID-1;SourceID-2=TemplatePath)", s)
	}

	sourceIDs := r[0]

	i.SourceIDs = strings.Split(sourceIDs, ";")
	i.TemplatePath = r[1]
	return nil
}
