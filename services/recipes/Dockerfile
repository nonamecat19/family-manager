# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.work go.work.sum* ./
COPY libs/go/ libs/go/
COPY sdk/go/ sdk/go/
COPY --parents services/*/go.mod services/*/go.sum ./
RUN go mod download all

COPY services/recipes/ services/recipes/

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /out/server ./services/recipes/cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
