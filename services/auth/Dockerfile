# syntax=docker/dockerfile:1
# Built from the repo root so the shared modules under libs/go and sdk/go are in context:
#   docker build -f services/auth/Dockerfile .
FROM golang:1.23-alpine AS build

WORKDIR /src

# Every module go.work lists must exist before `go mod download all`: it resolves the whole
# `use` set and hard-errors on a missing directory rather than skipping it. Enumerating the
# service manifests by hand rotted exactly that way once — adding recipes left the other three
# images failing here with "cannot load module services/recipes listed in go.work file" — so a
# glob does it instead, and a new service needs no edit in any of these four files. The
# `# syntax` pin above is what makes `COPY --parents` available; it preserves the
# services/<name>/ prefix that a plain wildcard would flatten into a single collided path.
# Still manifests only, never `COPY services/`: keeping sources out of this layer is what lets
# a code change reuse the dependency download below.
# go.work.sum is gitignored (.gitignore), so it exists on a developer's disk but never in a
# clean checkout — which is what CI builds from. The trailing glob makes it optional:
# Docker treats a wildcard matching nothing as a no-op, where the bare name is a hard
# error. The per-module go.sum files cover every dependency `go mod download all` needs,
# verified by building from a fresh clone with the file absent.
COPY go.work go.work.sum* ./
COPY libs/go/ libs/go/
COPY sdk/go/ sdk/go/
COPY --parents services/*/go.mod services/*/go.sum ./
RUN go mod download all

COPY services/auth/ services/auth/

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /out/server ./services/auth/cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
