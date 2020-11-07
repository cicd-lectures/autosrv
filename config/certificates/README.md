# Development Certificates

This folder hosts the certificate used for enabling TLS in a local development environment.

The following files are expected:

- `rootCA.pem` is the Certificate Authority file used to sign the certificate in PEM format.
- `cert.pem` is the certificate in PEM format. This certificate is expected to be valid for the following hostnames:
  - Principal hostname: `localhost`
  - Alternate hostnames= `*.localhost`, `127.0.0.1`
- `cert-key.pem` is the key of the certificate in PEM format.

The tool [mkcert](https://github.com/FiloSottile/mkcert) was used to generate this certificate but you can use any other tools if you want.
