# Preauthorization API

## Configuration

Put a secret into `server -> preauthorize-secret` (any random string of 32 characters or more)

## Create authorization token

Send a HTTP(S) POST request to the server at `/preauthorize` with either mTLS or an `Authorization` header or both (same as you would auth to WSVPN)

The server will return a JSON object like `{"success":true,"token":"abcdefg"}`.

## Send authorization token

Establish a connection to the server on `/preauthorize/TOKEN` (such as `ws://example.com/preauthorize/abcdefg`)
