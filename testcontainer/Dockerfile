FROM ubuntu

ARG TARGETARCH
ARG TARGETVARIANT

ENV TARGETARCH ${TARGETARCH}
ENV TARGETVARIANT ${TARGETVARIANT}

USER root

RUN apt update && \
    apt -y --no-install-recommends install python3 python3-yaml python3-pytest iproute2 openssl python3-scapy python3-pip tcpdump && \
    pip3 install pytest-github-actions-annotate-failures

VOLUME "/mnt"
WORKDIR /mnt

ENTRYPOINT [ "python3", "-m", "pytest", "-p", "no:cacheprovider", "./tests" ]
