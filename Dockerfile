FROM golang:alpine3.12 AS builder
MAINTAINER lucmichalski <michalski.luc@gmail.com>

RUN apk add --no-cache make gcc g++ ca-certificates musl-dev make git

COPY . /go/src/github.com/paper2code/golang-nng-surveyor-pattern-demo
WORKDIR /go/src/github.com/paper2code/golang-nng-surveyor-pattern-demo

RUN go install

FROM alpine:3.12 AS runtime
# FROM alpine:3.11 AS runtime
MAINTAINER lucmichalski <michalski.luc@gmail.com>

ARG TINI_VERSION=${TINI_VERSION:-"v0.18.0"}

# Install tini to /usr/local/sbin
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-muslc-amd64 /usr/local/sbin/tini

# Install runtime dependencies & create runtime user
RUN apk --no-cache --no-progress add ca-certificates go bash nano \
    && chmod +x /usr/local/sbin/tini && mkdir -p /opt \
    && adduser -D nng -h /opt/nng -s /bin/sh \
    && su nng -c 'cd /opt/nng; mkdir -p bin config data ui'

# Switch to user context
# USER nng
WORKDIR /opt/nng/bin

# copy executable
COPY --from=builder /go/bin/golang-nng-surveyor-pattern-demo /opt/nng/bin/golang-nng-surveyor-pattern-demo

ENV PATH $PATH:/opt/nng/bin

# Container configuration
# EXPOSE 9000
# VOLUME ["/opt/nng/bin/public"]

ENTRYPOINT ["tini", "-g", "--", "/opt/nng/bin/golang-nng-surveyor-pattern-demo"]
