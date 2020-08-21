### Simple User Application with gRPC

This repo contains a simple go application that supports user login/signin, sign up, and the basic CRUD APIs related to a
user model. I'll be using the term `user.app` to refer to this repository.

Table of Contents:
[Transport](#transport)
[Database](#database)
[Project Layout](#project-layout)
[Start Up](#start-up)
[Client](#client)
[Setup](#setup)

##### Transport
The transport used in this project is Protocol Buffers. The gRPC service created here, UserApp, located [here](https://github.com/pallavJha/user.app/blob/master/message/message.proto).
```proto
syntax = "proto3";
package message;

message AuthRequest {
    string username = 1;
    string email = 2;
    string password = 3;
}

message AuthResponse {
    string username = 1;
    string token = 2;
}

message CreateUserRequest {
    string username = 1;
    string email = 2;
    string password = 3;
}

message CreateUserResponse {
    string user_id = 1;
}

message UpdateUserRequest {
    string username = 2;
    string email = 3;
}

message Empty {
}

service UserApp {
    rpc SignIn (AuthRequest) returns (AuthResponse);
    rpc SignOut (Empty) returns (Empty);
    rpc CreateUser (CreateUserRequest) returns (CreateUserResponse);
    rpc UpdateUser (UpdateUserRequest) returns (Empty);
    rpc DeleteUser (Empty) returns (Empty);
}
```
This simple project currently supports (^)these 4 APIs.

##### Database
This service is using CockroachDB as the RDBMS storage. The configuration for the connection parameters is located [here](https://github.com/pallavJha/user.app/blob/master/configs/config.toml).
```toml
[crdb]
user="root"
password=""
host="localhost"
port=26257
dbname="userapp"
max_open_conns=200
max_idle_conns=50
conn_max_lifetime=5 #in minutes
```

##### Project Layout
The code present here can be compiled in to a binary(static as well), and the binary supports the sub commands because
it is based on [cobra](https://github.com/spf13/cobra). So, the project layout incorporates the cobra layout as well, and
all the command and the subcommands are present in the `cmd` package.

Apart from that, it contains the following directories:
- `configs` - To store the configuration
- `message` - To store the proto, and the go output
- `migration` - It stores the DB migrations. The migrations can be applied using the `task migrate` command
- `models` - Contains the ORM entities created using the reverse ORM [sqlboiler](https://github.com/volatiletech/sqlboiler)
    - `pkg/api` - API implementations for the protocol buffers  
    - `pkg/auth` - Authentication and JWT related stuff  
    - `pkg/conn` - Database connections  
    - `pkg/constants` - contains the constants used in this project  
    - `pkg/query` - contains the queries made to the cockroach db
 
##### Tools and Libraries
1. [Task](https://github.com/go-task/task) because I'm not using Makefile  
2. [SQLBoiler](https://github.com/volatiletech/sqlboiler) ORM
3. [Cobra](https://github.com/spf13/cobra) 
4. [Viper](https://github.com/spf13/viper)

Other packages can be found [here](https://github.com/pallavJha/user.app/blob/master/go.mod).

##### Start Up
The application can be started using the following command:
```bash
go run main.go server
```

It will start listening on localhost:9688 if every thing goes well.
```bash
{"level":"info","time":"2020-08-21T17:36:53+05:30","message":"application server listening on localhost:9688"}
INSERT INTO "users" ("username","email","password","last_login","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id","is_superuser","deleted_at"
[Hello Hello $argon2i$v=19$m=32768,t=1,p=4$E6ZPxnp4CLh0H/LTpvOwbg$jC0pgYVKn8X8oPtYwXjaNeezzvF8ECurfpvLDKTSmwI {0001-01-01 00:00:00 +0000 UTC false} 2020-08-21 12:07:12.5730433 +0000 UTC 2020-08-21 12:07:12.57
30433 +0000 UTC]
{"level":"info","time":"2020-08-21T17:37:12+05:30","message":"inserted successfully"}
SELECT * FROM "users" WHERE ("users"."deleted_at" = $1) AND ("users"."username" = $2);
```

To get the help section execute with `--help`
```bash
$ go run main.go
Root command for the user.app application

Usage:
  user.app [command]

Available Commands:
  help        Help about any command
  server      Run the userapp RPC service
  server      Run the userapp RPC service

Flags:
      --config string   Config file location (default "C:\\Users\\TheUser\\code\\user.app\\configs")
  -h, --help            help for user.app

Use "user.app [command] --help" for more information about a command.

$ go run main.go server --help
Run the userapp RPC service

Usage:
  user.app server [flags]

Flags:
  -h, --help          help for server
  -n, --host string   Service Host (default "localhost")
  -p, --port string   Port (default "9688")

Global Flags:
      --config string   Config file location (default "C:\\Users\\TheUser\\code\\user.app\\configs")
```

#### Client

You can use [Bloom RPC](https://github.com/uw-labs/bloomrpc) as the client for this gRPC service.

![Usage](https://github.com/pallavJha/user.app/blob/master/usage.gif?raw=true)

#### Setup

To run this you'll need to install:
1. Go 1.13
2. [Migrate](https://github.com/golang-migrate/migrate)
3. [Task](https://github.com/go-task/task)
4. [CockroachDB](https://www.cockroachlabs.com/docs/releases/v19.2.0.html)

Execute `task migrate` to migrate the ddls to the database, and then execute `go run main.go server` to start the server. 




