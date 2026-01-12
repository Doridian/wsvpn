FROM ubuntu

ARG TARGETARCH
ARG TARGETVARIANT

ENV TARGETARCH ${TARGETARCH}
ENV TARGETVARIANT ${TARGETVARIANT}

USER root

RUN apt update && \
    apt -y --no-install-recommends install python3 python3-pip iproute2 openssl tcpdump && \
    pip3 install --break-system-packages pytest-rerunfailures getmac scapy requests pytest pyyaml

VOLUME "/mnt"
WORKDIR /mnt

ENTRYPOINT [ "pytest", "--reruns", "3", "--reruns-delay", "1", "-p", "no:cacheprovider", "./tests" ]
