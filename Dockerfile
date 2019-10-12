# Create image based on the official Node 10 image from dockerhub
FROM golang:latest

# Create a directory where our app will be placed
RUN mkdir -p /go/src/app

# Change directory so that our commands run inside this new directory
WORKDIR /go/src/app

# Get all the code needed to run the app
COPY . /go/src/app

# Expose the port the app runs in
EXPOSE 2112

# Download das dependencias
RUN go mod download

