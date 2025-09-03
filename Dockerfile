FROM golang:1.24.3-bookworm AS build

WORKDIR /src

COPY . .

RUN set -xe; \
    go build \
      -buildmode=pie \
      -ldflags "-linkmode external -extldflags -static-pie" \
      -tags netgo \
      -o /cli \
      cli/cli.go \
    ;

# This stage is optional but reduces overall image size
FROM scratch
COPY --from=build /cli /cli
COPY --from=build /lib/x86_64-linux-gnu/libc.so.6 /lib/x86_64-linux-gnu/
COPY --from=build /lib64/ld-linux-x86-64.so.2 /lib64/