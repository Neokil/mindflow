FROM node:26.13.1-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Native OneNote codec: compile ahead of time and statically link musl.
# The SDK archive hashes are from Microsoft's .NET 10 release metadata.
FROM alpine:3.23 AS onenote
ARG TARGETARCH
WORKDIR /codec
RUN apk add --no-cache clang cmake build-base zlib-dev zlib-static openssl-dev openssl-libs-static icu-libs curl ca-certificates
RUN case "${TARGETARCH:-$(uname -m)}" in \
      amd64|x86_64) rid=x64; checksum=68ada5fd8fa2b65f55c3db1c2e773dfc9468c9828cbb56fc600151415cc2b116a32804f9aeeb82ca1a3229e3821af7b5b074d5e34487b2807630b5c00f51d801 ;; \
      arm64|aarch64) rid=arm64; checksum=b39da19b6134190ed0b9107c05c98c58586f4f77782c4b0e3e36c6274a2d59528f67412c2b0cbb927a914378177f763cb662b435c5bef62ad5c2cd86a422c877 ;; \
      *) exit 1 ;; \
    esac && curl -fsSL "https://builds.dotnet.microsoft.com/dotnet/Sdk/10.0.404/dotnet-sdk-10.0.404-linux-musl-$rid.tar.gz" -o /tmp/sdk.tar.gz \
    && echo "$checksum  /tmp/sdk.tar.gz" | sha512sum -c - \
    && mkdir /dotnet && tar -xzf /tmp/sdk.tar.gz -C /dotnet && rm /tmp/sdk.tar.gz
ENV DOTNET_ROOT=/dotnet PATH=/dotnet:$PATH DOTNET_CLI_TELEMETRY_OPTOUT=1
COPY internal/workspace/onenote/codec/ ./
RUN case "${TARGETARCH:-$(uname -m)}" in amd64|x86_64) rid=x64 ;; arm64|aarch64) rid=arm64 ;; *) exit 1 ;; esac \
    && dotnet publish -c Release -r "linux-musl-$rid" -o /out -p:RestoreLockedMode=true -p:StaticExecutable=true -p:StaticOpenSslLinking=true

FROM golang:1.26.11-alpine AS backend
ARG MINDFLOW_RELEASE=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY web/embed.go ./web/embed.go
COPY --from=frontend /app/web/dist/ ./web/dist/
RUN go run ./cmd/genmigrations -check && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.releaseVersion=${MINDFLOW_RELEASE}" -o /mindflow ./cmd/mindflow

FROM alpine:3.23 AS certificates
RUN apk add --no-cache ca-certificates && mkdir -p /runtime/data /runtime/tmp && chown 10001:10001 /runtime/data && chmod 1777 /runtime/tmp

FROM scratch
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=certificates /runtime/ /
COPY --from=backend /mindflow /usr/local/bin/mindflow
COPY --from=onenote /out/mindflow-onenote /usr/local/bin/mindflow-onenote
COPY LICENSE /usr/share/licenses/mindflow/LICENSE
COPY THIRD_PARTY_NOTICES.md /usr/share/licenses/mindflow/THIRD_PARTY_NOTICES.md
ENV MINDFLOW_DATA=/data MINDFLOW_ADDR=0.0.0.0:8080
USER 10001:10001
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/usr/local/bin/mindflow", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/mindflow"]
