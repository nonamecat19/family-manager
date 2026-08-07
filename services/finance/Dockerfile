# Built from the repo root so the shared modules under libs/go and sdk/go are in context:
#   docker build -f services/finance/Dockerfile .
FROM golang:1.23-alpine AS build

WORKDIR /src

# Copy every module the go.work graph needs, manifests first so dependency downloads cache.
COPY go.work go.work.sum ./
COPY libs/go/ libs/go/
COPY sdk/go/ sdk/go/
COPY services/finance/go.mod services/finance/go.sum services/finance/
COPY services/auth/go.mod services/auth/go.sum services/auth/
COPY services/family/go.mod services/family/go.sum services/family/
RUN go mod download all

COPY services/finance/ services/finance/

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /out/server ./services/finance/cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
