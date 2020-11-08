FROM golang:1.15-alpine AS build
COPY ./ /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o ./deployer ./cmd/deployer

FROM scratch
COPY --from=build /app/deployer /deployer
VOLUME /var/run
ENTRYPOINT ["/deployer"]
