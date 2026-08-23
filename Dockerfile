# Build stage.
#
# The go.mod has no requires, so there is no dependency download step and
# nothing to cache — the whole build is the standard library plus this source.
FROM golang:1.25-alpine AS build

WORKDIR /src
COPY . .

# CGO off makes the binary static, which is what lets the final image be
# distroless with no libc in it at all.
#
# -trimpath keeps build machine paths out of the binary, and -s -w drop the
# symbol table and DWARF data: nothing here is debugged from a production
# core dump, and it takes a few megabytes off the image.
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# The templates, stylesheet, fonts and the generated cv.pdf are all compiled
# into the binary by embed.FS, so verify it really is self-contained: a
# dynamically linked binary here would mean CGO crept back in and the
# distroless stage below would fail at runtime instead of now.
RUN ! ldd /out/server 2>/dev/null | grep -q "=>" || (echo "binary is not static" && exit 1)

# Runtime stage.
#
# distroless/static carries no shell, no package manager and no libc — there is
# nothing in the image to execute but the server itself, which is the smallest
# useful attack surface available. It does carry ca-certificates, which the
# contact form needs to reach the Resend and Postmark APIs over TLS.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/server /app/server

# Runs as uid 65532. Nothing is written to disk at runtime — there is no
# database and no upload path — so the filesystem needs no write access.
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/server"]
