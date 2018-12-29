FROM golang:1.11.4-alpine@sha256:198cb8c94b9ee6941ce6d58f29aadb855f64600918ce602cdeacb018ad77d647 AS build

RUN apk add --no-cache \
    make \
    git \
    upx=3.94-r0

RUN mkdir -p /go/src/github.com/hairyhenderson/go-revendor-bot
WORKDIR /go/src/github.com/hairyhenderson/go-revendor-bot
COPY . /go/src/github.com/hairyhenderson/go-revendor-bot

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

RUN make build-x compress-all

FROM scratch AS artifacts

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/hairyhenderson/go-revendor-bot/bin/* /bin/

CMD [ "/bin/go-revendor-bot_linux-amd64" ]

FROM scratch AS latest

ARG OS=linux
ARG ARCH=amd64

COPY --from=artifacts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=artifacts /bin/go-revendor-bot_${OS}-${ARCH} /go-revendor-bot

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

LABEL org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.title=go-revendor-bot \
      org.opencontainers.image.authors=$CODEOWNERS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/hairyhenderson/go-revendor-bot"

ENTRYPOINT [ "/go-revendor-bot" ]

FROM alpine:3.8@sha256:46e71df1e5191ab8b8034c5189e325258ec44ea739bba1e5645cff83c9048ff1 AS alpine

ARG OS=linux
ARG ARCH=amd64

RUN apk add --no-cache ca-certificates
COPY --from=artifacts /bin/go-revendor-bot_${OS}-${ARCH}-slim /bin/go-revendor-bot

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

LABEL org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.title=go-revendor-bot \
      org.opencontainers.image.authors=$CODEOWNERS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/hairyhenderson/go-revendor-bot"

ENTRYPOINT [ "/bin/go-revendor-bot" ]

FROM scratch AS slim

ARG OS=linux
ARG ARCH=amd64

COPY --from=artifacts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=artifacts /bin/go-revendor-bot_${OS}-${ARCH}-slim /go-revendor-bot

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

LABEL org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.title=go-revendor-bot \
      org.opencontainers.image.authors=$CODEOWNERS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/hairyhenderson/go-revendor-bot"

ENTRYPOINT [ "/go-revendor-bot" ]