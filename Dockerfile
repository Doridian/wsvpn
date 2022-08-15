FROM alpine:latest

RUN apk add --no-cache ca-certificates curl bash

ARG TARGETARCH
ARG TARGETVARIANT
ARG SIDE

COPY --chown=0:0 --chmod=755 dist/$SIDE-linux-$TARGETARCH$TARGETVARIANT /wsvpn
VOLUME /config

WORKDIR /config
ENTRYPOINT [ "/wsvpn" ]
