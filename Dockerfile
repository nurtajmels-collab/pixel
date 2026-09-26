# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -o /server ./cmd/server

# Frontend build stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Final stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /server ./server
COPY --from=frontend-builder /app/frontend/dist ./public

EXPOSE 8080

CMD ["./server"]