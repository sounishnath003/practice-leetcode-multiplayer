# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod tidy && go mod download && go mod verify
RUN CGO_ENABLED=0 GOARCH=amd64 go build -o /app/bin/practice_leetcode_multiplayer main.go

# Final stage
FROM scratch
WORKDIR /app

COPY --from=builder /app/bin/practice_leetcode_multiplayer /app/bin/practice_leetcode_multiplayer
COPY --from=builder /app/templates/ ./templates/

EXPOSE 3000
ENTRYPOINT [ "/app/bin/practice_leetcode_multiplayer" ]