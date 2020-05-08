FROM golang:1.13.4-alpine3.10 AS Builder

WORKDIR /hermes
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o hermes

FROM gcr.io/distroless/base-debian10
COPY --from=Builder /hermes/hermes /hermes
COPY --from=Builder /hermes/configs /configs
ENTRYPOINT ["./hermes"]
