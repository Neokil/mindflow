FROM node:26.13.1-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.11-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY web/embed.go ./web/embed.go
COPY --from=frontend /app/web/dist/ ./web/dist/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /mindflow ./cmd/mindflow

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && addgroup -g 10001 mindflow && adduser -D -H -u 10001 -G mindflow mindflow && mkdir /data && chown 10001:10001 /data
COPY --from=backend /mindflow /usr/local/bin/mindflow
COPY THIRD_PARTY_NOTICES.md /usr/share/licenses/mindflow/THIRD_PARTY_NOTICES.md
ENV MINDFLOW_DATA=/data MINDFLOW_ADDR=0.0.0.0:8080
USER 10001:10001
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["mindflow", "healthcheck"]
ENTRYPOINT ["mindflow"]
