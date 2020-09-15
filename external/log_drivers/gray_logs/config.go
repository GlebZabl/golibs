package gray_logs

import "crypto/tls"

type Config struct {
	Addr         string
	Port         uint
	TimeOut      int
	Version      string
	HostName     string
	Certificates []tls.Certificate
}
