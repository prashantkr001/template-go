[![golangci-lint](https://github.com/prashantkr001/template-go
/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/prashantkr001/template-go
/actions/workflows/golangci-lint.yml)

# Go service template

This is the template repository which sets a standard project structure for any Go application. The
intention of this repository is to help understand the structure, and serve as guidelines.
e.g. it has a MongoDB driver initialization, that doesn't mean you cannot use another database
while using this structure.

The template is _heavily inspired_ by [Goapp](https://github.com/naughtygopher/goapp).
It is recommended to read Goapp to better understand the structure implemented. Any deviations
from Goapp will & are documented here as and when we discover.

## Important

The [Goapp](https://github.com/naughtygopher/goapp) README helps you understand the folder structure,
control flow of the application, dependency flow of the application etc. The sole purpose of this
template is to help us _easily_ identify how to create different **layers** of the application.
Which would inturn help us maintain a well structured, maintainable, readable repository.

## How to run the sample application?

Ensure the pre-requisites are met. There's a docker-compose.yml file provided in `docker/development`.
So you can run the following from the repository directory to start this application

```bash
$ cd ./docker/development
$ docker compose up
```

Make some API calls to the app

For HTTP, you can try the below calls, but should have [curl](https://curl.se/) installed.

```bash
$ curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"id":1,"name":"Bottle"}' \
  http://localhost:5001/items

# if the above command was successful, the below one should return some result
$ curl -v "http://localhost:5001/items?limit=2"

# try different errors
$ curl -v "http://localhost:5001/items?limit=haha"
$ curl -v --header "Content-Type: application/json" \
  --request POST \
  --data '{"id":0,"name":"Bottle"}' \
  http://localhost:5001/items
$ curl -v --header "Content-Type: application/json" \
  --request POST \
  --data '{"id":-1,"name":"Bottle"}' \
  http://localhost:5001/items
```

For gRPC, you can try the below calls, but should have [grpcurl](https://github.com/fullstorydev/grpcurl) installed.

```bash
# Item create gRPC call using grpcurl
$ grpcurl -plaintext -d '{"id":1, "name": "Fullsnack developer"}' localhost:5002 items.v1.ItemsService/CreateItem

# Item list gRPC call using grpcurl
$ grpcurl -plaintext -d '{"limit":10}' localhost:5002 items.v1.ItemsService/ListItems
```

### Pre-requisites

1. Docker (on Linux) / [Colima](https://github.com/abiosoft/colima) (on MacOS & Windows)
2. [Docker compose](https://docs.docker.com/compose/gettingstarted/)

If you do not want to use Docker, please refer the docker-compose.yml file for all the configurations required.

**Note**: [Colima file-change notification](https://github.com/lima-vm/lima/issues/615) doesn't seem to
be working. So hot-reload wouldn't work. It'd work fine if you're using Docker Desktop.

**For MacOS**

Install the latest [stable version of Brew](https://brew.sh/).

```bash
$ brew install docker
$ brew install colima
```

Install docker compose plugin; [ref](https://docs.docker.com/compose/install/linux/#install-the-plugin-manually)

```bash
$ DOCKER_CONFIG=${DOCKER_CONFIG:-$HOME/.docker}
$ mkdir -p $DOCKER_CONFIG/cli-plugins
$ export DC_VERSION="v2.15.1" && curl -SL "https://github.com/docker/compose/releases/download/${DC_VERSION}/docker-compose-darwin-$(uname -m)" -o $DOCKER_CONFIG/cli-plugins/docker-compose-testing
$ chmod +x $DOCKER_CONFIG/cli-plugins/docker-compose

# the below command should successfully run and produce an output similar to what's shown below
$ docker compose version
Docker Compose version v2.15.1
# finally start the runtime
$ colima start
```

## Notes

For using grpc + protobuf, you need the following tools installed on your computer.

1. [Go](https://go.dev/)
2. [Protoc](https://grpc.io/docs/protoc-installation/)

After you make sure you have installed Go on your computer, run the below commands to
install plugins required for protobuff generation for Go.

```bash
$ go install google.golang.org/protobuf/cmd/protoc-gen-go@v1
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1
```

You can try generating the code for the items.proto file used in this repository as follows

```bash
$ cd cmd/server/grpc/proto
$ protoc --go_out=${PWD} \
    --go-grpc_out=${PWD} items.proto
```

This will generate the serialization/deserialization code as well as code required for
gRPC client & server.
