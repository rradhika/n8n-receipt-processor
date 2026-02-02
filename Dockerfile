# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc g++ musl-dev tesseract-ocr-dev leptonica-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install Tesseract OCR, Poppler tools, and dependencies
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    poppler-utils \
    libc6-compat

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Create uploads directory
RUN mkdir -p /app/uploads

# Expose port
EXPOSE 3000

# Run the application
CMD ["./main"]
