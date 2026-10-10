FROM golang:1.24-alpine AS build

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION
ARG GIT_COMMIT
ARG GIT_COMMIT_TIME

ENV GOPATH=/go
ENV CGO_ENABLED=0
ENV GO111MODULE=on

WORKDIR /build

RUN apk add -U --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN LDFLAGS="$(MINIO_RELEASE=RELEASE \
    MINIO_COMMIT_ID="${GIT_COMMIT}" \
    MINIO_COMMIT_TIME="${GIT_COMMIT_TIME}" \
    go run buildscripts/gen-ldflags.go "${VERSION}")" && \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -tags kqueue -trimpath --ldflags "${LDFLAGS}" -o /output/minio

FROM alpine:3.20

# tzdata for timezone correct time and date.
RUN apk add -U --no-cache ca-certificates \
    && apk add -U --no-cache libc6-compat \
    && apk add -U --no-cache tzdata \
    && apk add -U --no-cache curl \
    && rm -rf /var/cache/apk/* \
    && mkdir -p /licenses \
    && chmod -R 777 /usr/bin

COPY --from=build /output/minio /usr/bin/minio
COPY dockerscripts/docker-entrypoint.sh /usr/bin/docker-entrypoint.sh
COPY CREDITS /licenses/CREDITS
COPY LICENSE /licenses/LICENSE

ENV MINIO_ACCESS_KEY_FILE=access_key \
    MINIO_SECRET_KEY_FILE=secret_key \
    MINIO_ROOT_USER_FILE=access_key \
    MINIO_ROOT_PASSWORD_FILE=secret_key \
    MINIO_KMS_SECRET_KEY_FILE=kms_master_key \
    MINIO_CONFIG_ENV_FILE=config.env \
    MINIO_PROMETHEUS_AUTH_TYPE=public

EXPOSE 9000 9001

VOLUME ["/data"]

ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]
CMD ["minio"]