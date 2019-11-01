FROM golang:1.13-alpine AS builder
RUN mkdir -p /k8s-proxy
WORKDIR /k8s-proxy

RUN apk add -u git curl

COPY go.* ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/k8s-proxy

FROM scratch
COPY --from=builder /k8s-proxy/bin/. .

ENTRYPOINT ["/k8s-proxy"]