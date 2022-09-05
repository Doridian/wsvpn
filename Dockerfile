FROM alpine:latest

RUN apk add --no-cache ca-certificates curl bash

ARG TARGETARCH
ARG TARGETVARIANT
ARG PROJECT

COPY --chown=0:0 --chmod=755 dist/$PROJECT-linux-$TARGETARCH$TARGETVARIANT /wsvpn
VOLUME /config

WORKDIR /config
ENTRYPOINT [ "/wsvpn" ]
