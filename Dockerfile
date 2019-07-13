FROM golang:1.11.5-alpine@sha256:4b8a4130c0d96bc9d75ed0e9d606b16a122a589cdc1d10491fbae8a12828b136 AS build

RUN apk add --no-cache \
    make \
    git \
    upx=3.94-r0

RUN mkdir -p /go/src/github.com/hairyhenderson/go-revendorbot
WORKDIR /go/src/github.com/hairyhenderson/go-revendorbot
COPY . /go/src/github.com/hairyhenderson/go-revendorbot

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

RUN make build-x compress-all

FROM scratch AS artifacts

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/hairyhenderson/go-revendorbot/bin/* /bin/

CMD [ "/bin/go-revendorbot_linux-amd64" ]

FROM scratch AS latest

ARG OS=linux
ARG ARCH=amd64

COPY --from=artifacts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=artifacts /bin/go-revendorbot_${OS}-${ARCH} /go-revendorbot

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

LABEL org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.title=go-revendorbot \
      org.opencontainers.image.authors=$CODEOWNERS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/hairyhenderson/go-revendorbot"

ENTRYPOINT [ "/go-revendorbot" ]

FROM alpine:3.10@sha256:6a92cd1fcdc8d8cdec60f33dda4db2cb1fcdcacf3410a8e05b3741f44a9b5998 AS alpine

ARG OS=linux
ARG ARCH=amd64

RUN apk add --no-cache ca-certificates
COPY --from=artifacts /bin/go-revendorbot_${OS}-${ARCH}-slim /bin/go-revendorbot

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

LABEL org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.title=go-revendorbot \
      org.opencontainers.image.authors=$CODEOWNERS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/hairyhenderson/go-revendorbot"

ENTRYPOINT [ "/bin/go-revendorbot" ]

FROM scratch AS slim

ARG OS=linux
ARG ARCH=amd64

COPY --from=artifacts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=artifacts /bin/go-revendorbot_${OS}-${ARCH}-slim /go-revendorbot

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

LABEL org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.title=go-revendorbot \
      org.opencontainers.image.authors=$CODEOWNERS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/hairyhenderson/go-revendorbot"

ENTRYPOINT [ "/go-revendorbot" ]
