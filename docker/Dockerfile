# Fork build of PG Back Web.
#
# - Build stage: Go + Node build the binaries and the frontend assets.
# - Runtime stage: Debian slim with the PostgreSQL clients only. No Go, no
#   Node, no build tools, no source code. Runs as an unprivileged user.
#
# Local backups: the /backups volume must be writable by UID 10001, for
# example `chown -R 10001:10001 ./backups` on the host.

FROM --platform=$BUILDPLATFORM node:22-trixie-slim AS node

FROM --platform=$BUILDPLATFORM golang:1.26-trixie AS build
ARG TARGETOS
ARG TARGETARCH
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
      go install -tags='no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb' \
      github.com/pressly/goose/v3/cmd/goose@v3.25.0 && \
    mkdir -p /out && \
    (cp /go/bin/${TARGETOS}_${TARGETARCH}/goose /out/goose 2>/dev/null || cp /go/bin/goose /out/goose)

WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN node ./scripts/sqlc-prebuild.ts && sqlc generate && \
    npm run tailwindcss -- --minify --config ./tailwind.config.ts \
      --input ./internal/view/static/css/style.css \
      --output ./internal/view/static/build/style.min.css && \
    node ./scripts/build-js.ts && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
      go build -trimpath -ldflags='-s -w' -o /out/app ./cmd/app/. && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
      go build -trimpath -ldflags='-s -w' -o /out/change-password ./cmd/changepw/.

FROM debian:trixie-slim
LABEL org.opencontainers.image.source="https://github.com/alehano/pgbackweb"
ENV DEBIAN_FRONTEND=noninteractive
RUN set -e && \
    apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates postgresql-common && \
    /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y && \
    apt-get install -y --no-install-recommends \
      tzdata wget unzip \
      postgresql-client-13 postgresql-client-14 postgresql-client-15 \
      postgresql-client-16 postgresql-client-17 postgresql-client-18 && \
    rm -rf /var/lib/apt/lists/* && \
    ! command -v node && \
    groupadd --gid 10001 pbw && \
    useradd --uid 10001 --gid 10001 --no-create-home --shell /usr/sbin/nologin pbw && \
    mkdir -p /backups && chown pbw:pbw /backups

COPY --from=build /out/app /app/app
COPY --from=build /out/goose /usr/local/bin/goose
COPY --from=build /out/change-password /usr/local/bin/change-password
COPY --from=build /src/internal/database/migrations /app/migrations
COPY docker/entrypoint.sh /app/entrypoint.sh

WORKDIR /app
USER pbw
EXPOSE 8085
ENTRYPOINT ["/app/entrypoint.sh"]
