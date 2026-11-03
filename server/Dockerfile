FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN cd server && CGO_ENABLED=0 go build -o /wepub-server ./cmd/wepub-server

FROM gcr.io/distroless/static
COPY --from=build /wepub-server /wepub-server
USER nonroot
ENTRYPOINT ["/wepub-server"]
