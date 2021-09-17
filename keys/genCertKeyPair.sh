#!/bin/sh
# Generates a self-signed cert/key pair
# This is not signed by another agency like LetsEncrypt
# Warning will be shown on a browser but you can pass through
# This command only works on UNIX, replace the stuff with $(...) with
# your location of GOROOT is on Plan9/Windows
go run $(go env GOROOT)/src/crypto/tls/generate_cert.go --host=localhost
