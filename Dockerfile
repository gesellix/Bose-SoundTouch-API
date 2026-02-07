# Stage 1: Build Go binary
FROM golang:1.25.6-alpine AS go-builder
WORKDIR /src
COPY soundcork-go/go.mod soundcork-go/go.sum ./
RUN go mod download
COPY soundcork-go/ ./
RUN CGO_ENABLED=0 go build -o /app/bose-soundtouch-api-bin ./soundcork-go

# Stage 2: Build final image
FROM python:3.12-slim

# Set environment variables
ENV BASE_URL=""
ENV DATA_DIR="/data"
ENV PORT=8000
ENV BIND_ADDR="0.0.0.0"
ENV PYTHON_BACKEND_URL="http://localhost:8001"

# Expose the primary port
EXPOSE ${PORT}

# Create a directory for data persistence
RUN mkdir -p /data

# Set the working directory in the container
WORKDIR /app

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy Go binary from builder
COPY --from=go-builder /app/bose-soundtouch-api-bin .

# Copy the rest of the application code
COPY . .

# Ensure entrypoint is executable
RUN chmod +x entrypoint.sh

# Run both Go and Python services
CMD ["./entrypoint.sh"]
