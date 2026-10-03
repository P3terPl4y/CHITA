FROM node:22-alpine AS frontend
WORKDIR /frontend
COPY react/package.json react/package-lock.json ./
RUN npm ci
COPY react/ ./
RUN npm run build

FROM golang:1.26.8-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/chita .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S chita \
    && adduser -S -G chita chita \
    && mkdir -p /www/storage/app /www/storage/framework/sessions /www/storage/logs \
    && chown -R chita:chita /www/storage
WORKDIR /www
COPY --from=builder --chown=chita:chita /out/chita /www/main
COPY --from=builder --chown=chita:chita /build/public/ /www/public/
COPY --from=builder --chown=chita:chita /build/resources/ /www/resources/
COPY --from=frontend --chown=chita:chita /frontend/dist/ /www/react/dist/
USER chita
EXPOSE 3330
ENTRYPOINT ["/www/main"]
