# The two build stages run on the builder's own architecture and cross-compile,
# so multi-arch images do not need emulation.
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG VERSION=dev
ARG COMMIT=unknown
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -o /chaptarr ./cmd/chaptarr

FROM alpine:3.24
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata && adduser -D -u 1000 chaptarr \
	&& mkdir -p /config && chown chaptarr /config
COPY --from=build /chaptarr /usr/local/bin/chaptarr
USER chaptarr
ENV CHAPTARR_DATA=/config CHAPTARR_ADDR=:8789
EXPOSE 8789
VOLUME /config
HEALTHCHECK --interval=30s --timeout=5s CMD wget -qO- http://127.0.0.1:8789/healthz || exit 1
ENTRYPOINT ["chaptarr"]
