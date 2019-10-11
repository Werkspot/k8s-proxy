FROM golang:1.13-alpine AS builder
RUN mkdir -p /varnish-purger
WORKDIR /varnish-purger

RUN apk add -u git curl

COPY go.* ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/varnish-purger

FROM scratch
COPY --from=builder /varnish-purger/bin/. .

ENTRYPOINT ["/varnish-purger"]