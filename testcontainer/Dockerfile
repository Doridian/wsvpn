FROM ubuntu

ARG TARGETARCH
ARG TARGETVARIANT

ENV TARGETARCH ${TARGETARCH}
ENV TARGETVARIANT ${TARGETVARIANT}

USER root

RUN apt update && \
    apt -y --no-install-recommends install python3 python3-pip iproute2 openssl tcpdump && \
    pip3 install wheel pytest-github-actions-annotate-failures getmac scapy requests pytest pyyaml

VOLUME "/mnt"
WORKDIR /mnt

ENTRYPOINT [ "python3", "-m", "pytest", "-p", "no:cacheprovider", "./tests" ]
