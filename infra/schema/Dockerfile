# Tiny image that ships the mysqldef binary. Used by the `schema` compose
# service (and locally via `make schema-apply`) to converge MySQL DDL to the
# desired state declared in schema.sql.
FROM alpine:3.20

ARG MYSQLDEF_VERSION=v0.17.20
ARG TARGETARCH=amd64

RUN apk add --no-cache curl tar ca-certificates \
 && case "${TARGETARCH}" in \
      amd64) ARCH="amd64" ;; \
      arm64) ARCH="arm64" ;; \
      *)     ARCH="amd64" ;; \
    esac \
 && curl -fsSL "https://github.com/sqldef/sqldef/releases/download/${MYSQLDEF_VERSION}/mysqldef_linux_${ARCH}.tar.gz" \
      | tar -xz -C /usr/local/bin \
 && mysqldef --version

WORKDIR /schema

ENTRYPOINT ["/usr/local/bin/mysqldef"]
