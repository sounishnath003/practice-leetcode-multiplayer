# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .
# https://stackoverflow.com/questions/76883458/go-error-in-docker-x509-certificate-signed-by-unknown-authority
RUN apk --no-cache add ca-certificates

RUN go mod tidy && go mod download && go mod verify
RUN CGO_ENABLED=0 GOARCH=amd64 go build -o /app/bin/practice_leetcode_multiplayer main.go

# Final stage
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Otherwise directly I could use -
#Using google distroless as it includes static and ca-certificates bundle
# FROM gcr.io/distroless/static
WORKDIR /app

COPY --from=builder /app/bin/practice_leetcode_multiplayer /app/bin/practice_leetcode_multiplayer
COPY --from=builder /app/templates/ ./templates/

EXPOSE 3000
ENTRYPOINT [ "/app/bin/practice_leetcode_multiplayer" ]