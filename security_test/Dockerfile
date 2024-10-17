# Stage 1: Use the Terraform image to copy Terraform binary
FROM gcr.io/dlp-demo-2-193019/terraform:latest as terraform

# Stage 2: Use a Go image to build your Go application
FROM golang:1.22 as builder

# Install gcloud SDK and apt-utils
RUN apt-get update && apt-get install -y \
    apt-utils \
    curl \
    apt-transport-https \
    ca-certificates \
    gnupg && \
    # Add the Google Cloud SDK distribution URI as a package source
    echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    # Import the Google Cloud public key
    curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && \
    # Update and install the Google Cloud SDK and GKE gcloud auth plugin
    apt-get update && apt-get install -y \
    google-cloud-sdk \
    google-cloud-sdk-gke-gcloud-auth-plugin

# Copy Terraform binary from the terraform stage
COPY --from=terraform /usr/bin/terraform /usr/local/bin/terraform

# Set the working directory inside the container
WORKDIR /workspace

# Copy the Go module files and the vendor folder
COPY go.mod go.sum vendor/ ./

# Copy the rest of the application code
COPY . .

# Run `go fmt` and capture the output, using a script for better error handling
RUN gofmt_output=$(go fmt ./... 2>&1) || (echo "$gofmt_output" && exit 1)

# Build the Go binary
RUN go build -mod=vendor -o check_violations .

# Ensure the binary is executable
RUN chmod +x /workspace/check_violations

# Specify the entrypoint for the container
ENTRYPOINT ["/workspace/check_violations"]
