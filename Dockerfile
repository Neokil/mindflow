FROM node:26.13.1-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

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
COPY LICENSE /usr/share/licenses/mindflow/LICENSE
COPY THIRD_PARTY_NOTICES.md /usr/share/licenses/mindflow/THIRD_PARTY_NOTICES.md
ENV MINDFLOW_DATA=/data MINDFLOW_ADDR=0.0.0.0:8080
USER 10001:10001
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/usr/local/bin/mindflow", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/mindflow"]
