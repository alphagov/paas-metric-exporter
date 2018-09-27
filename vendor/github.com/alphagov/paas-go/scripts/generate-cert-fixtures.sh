#!/bin/bash

set -eu

# Create state for OpenSSL database
touch index.txt
echo 1000 > serial

cleanup() {
  rm -f fixtures/1000.pem
  rm -f fixtures/1001.pem
  rm -f fixtures/1002.pem
  rm -f fixtures/client.csr.pem
  rm -f fixtures/locket-server.csr.pem
  rm -f fixtures/loggregator-server.csr.pem
  rm -f index.txt index.txt.attr index.txt.old index.txt.attr.old
  rm -f serial serial.old
}
trap 'cleanup' EXIT

# Create CA certificate
openssl genrsa -out fixtures/ca.key.pem 2048
openssl req -config scripts/openssl.cnf \
  -batch \
  -subj "/C=GB/ST=London/L=London/O=Global Security/OU=IT Department/CN=ca" \
  -key fixtures/ca.key.pem \
  -new -x509 -days 7300 -sha256 -extensions v3_ca \
  -out fixtures/ca.cert.pem

# Create client certificate
openssl genrsa -out fixtures/client.key.pem 2048
openssl req -config scripts/openssl.cnf -new -sha256 \
  -subj "/C=GB/ST=London/L=London/O=Global Security/OU=IT Department/CN=client" \
  -key fixtures/client.key.pem \
  -out fixtures/client.csr.pem
openssl ca -config scripts/openssl.cnf -extensions usr_cert \
  -batch \
  -days 3650 -notext -md sha256 \
  -in fixtures/client.csr.pem \
  -out fixtures/client.cert.pem

# Create Locket server certificate
openssl genrsa -out fixtures/locket-server.key.pem 2048
openssl req -config scripts/openssl.cnf -new -sha256 \
  -subj "/C=GB/ST=London/L=London/O=Global Security/OU=IT Department/CN=locket" \
  -key fixtures/locket-server.key.pem \
  -out fixtures/locket-server.csr.pem
# Note: we have to set the SAN to the loopback address for the Locket client to accept it.
openssl ca -config scripts/openssl.cnf -extensions server_cert_with_san \
  -batch \
  -days 3650 -notext -md sha256 \
  -in fixtures/locket-server.csr.pem \
  -out fixtures/locket-server.cert.pem

# Create Loggregator server certificate
# Note: the common name MUST be set to `metron`.
openssl genrsa -out fixtures/loggregator-server.key.pem 2048
openssl req -config scripts/openssl.cnf -new -sha256 \
  -subj "/C=GB/ST=London/L=London/O=Global Security/OU=IT Department/CN=metron" \
  -key fixtures/loggregator-server.key.pem \
  -out fixtures/loggregator-server.csr.pem
# Note: we should NOT add a SAN here
openssl ca -config scripts/openssl.cnf -extensions server_cert \
  -batch \
  -days 3650 -notext -md sha256 \
  -in fixtures/loggregator-server.csr.pem \
  -out fixtures/loggregator-server.cert.pem

# Verify
openssl verify -CAfile fixtures/ca.cert.pem fixtures/client.cert.pem
openssl verify -CAfile fixtures/ca.cert.pem fixtures/locket-server.cert.pem
openssl verify -CAfile fixtures/ca.cert.pem fixtures/loggregator-server.cert.pem
