FROM ubuntu

ARG TARGETARCH
ARG TARGETVARIANT

ENV TARGETARCH ${TARGETARCH}
ENV TARGETVARIANT ${TARGETVARIANT}

USER root

RUN apt update && \
    apt -y --no-install-recommends install python3 python3-pip iproute2 openssl tcpdump && \
    pip3 install pytest-github-actions-annotate-failures pytest-rerunfailures getmac scapy requests pytest pyyaml

VOLUME "/mnt"
WORKDIR /mnt

ENTRYPOINT [ "python3", "--reruns", "3", "--reruns-delay", "1", "-m", "pytest", "-p", "no:cacheprovider", "./tests" ]
