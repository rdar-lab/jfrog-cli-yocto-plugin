# build stage
FROM golang:1.15.5-alpine AS build-stage
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o jfrog-yocto

# production stage
FROM gmacario/build-yocto
COPY --from=build-stage /src/jfrog-yocto /home/build
RUN mkdir /home/build/workspace
WORKDIR /home/build/workspace
ENTRYPOINT ["/home/build/jfrog-yocto"]