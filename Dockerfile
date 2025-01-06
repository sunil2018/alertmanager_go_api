# Use the official Alpine Linux image
FROM alpine:latest

# Create a directory for the app (optional but good practice)
RUN mkdir /app

# Set the working directory
WORKDIR /app

# Copy the pre-compiled Go binary into the container
COPY alertapi /app/

# Make the binary executable
RUN chmod +x /app/alertapi

# Set the entrypoint to your Go binary
ENTRYPOINT ["/app/alertapi"]