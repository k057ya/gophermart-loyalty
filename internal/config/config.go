package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type httpServer struct {
	Address string
	Port    int
}

func (s *httpServer) String() string {
	return net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
}

func (s *httpServer) Set(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return err
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid HTTP port %q: %w", port, err)
	}
	s.Address = host
	s.Port = int(number)
	return nil
}

type database struct {
	Dsn string
}

type client struct {
	BaseUrl string
}

func (d *database) String() string {
	return d.Dsn
}

func (d *database) Set(value string) error {
	d.Dsn = value
	return nil
}

type ServiceConfig struct {
	Http     *httpServer
	Database *database
	Client   *client
}

func (c *ServiceConfig) Validate() error {
	var errs []error
	if c.Database == nil || strings.TrimSpace(c.Database.Dsn) == "" {
		errs = append(errs, errors.New("database URI is empty (set DATABASE_URI or pass -d)"))
	}
	if c.Client == nil || strings.TrimSpace(c.Client.BaseUrl) == "" {
		errs = append(errs, errors.New("accrual system address is empty (set ACCRUAL_SYSTEM_ADDRESS or pass -r)"))
	}
	return errors.Join(errs...)
}

var ServiceCfg = &ServiceConfig{
	Http:     &httpServer{Address: "localhost", Port: 8080},
	Database: &database{},
	Client:   &client{BaseUrl: "http://localhost:8081"},
}
