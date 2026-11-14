# Users Service

**users-service** is a central Go microservice responsible for user management, authorization, and security within the application ecosystem, built with Gin, Ent, and a Postgres + Redis + Kafka backing stack. It was designed as a secure, event-driven identity foundation for the rest of the microservice landscape.

## Table of Contents
- [Key Features](#key-features)
- [Installation](#installation)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [Development and Testing](#development-and-testing)
- [Manually Building, Tagging, and Pushing Docker Images](#manually-building-tagging-and-pushing-docker-images)
- [Troubleshooting](#troubleshooting)
- [Useful Commands and Addresses](#useful-commands-and-addresses)
- [Updates](#updates)
- [License](#license)

## Key Features
- **User Lifecycle Management**: Full support for registration, account activation, verification, and account deletion processes.
- **Advanced Authorization**:
    - Generation and validation of secure JWTs (Ed25519).
    - Support for two-factor authentication (2FA).
    - Password reset mechanism.
- **Device Fingerprinting**: Unique identification of each device a user logs in from to enhance security and enable session management.
- **Access Control (Roles & Permissions)**: A system of roles and permissions that allows for fine-grained control over access to different parts of the system.
- **Event-Driven System**: Publication of key events (e.g., user registration, login) to a Kafka cluster, enabling asynchronous communication between microservices.
- **Caching**: Redis-backed caching for short-lived, security-sensitive data (e.g., activation and reset codes).
- **Self-Documenting API**: Swagger/OpenAPI docs and Redoc, generated from code annotations via `swag`.
- **Auditing and Activity Tracking**: Recording key actions performed by users.

## Installation

### Requirements
- **Linux, Debian 12 preferred**: Windows k8s installation might be problematic for promtail configuration.
- **Golang 1.24 or newer**: As dependencies in libraries required these versions.
- **Docker Engine 28.2.2 or newer** (or **Docker Desktop 28.1.1+**): Installed and running, for local infrastructure and integration/e2e tests.
- **`make`**: The project is driven by a `Makefile`.

### Build and Installation
The project includes a `Makefile` that orchestrates module installation, dev-tool setup, local infrastructure, and code generation:

```bash
git clone https://github.com/factory-chainline/users-service
cd users-service

make setup
```

`make setup` downloads Go modules, installs dev tools (`swag`, `mockery`, `ent`, `air`, `golangci-lint`), generates a local JWT keypair and Postgres TLS certificates, starts local infrastructure (Postgres with TLS, Redis, a 3-broker Kafka cluster), generates Ent models/mocks, and builds the binary.

Copy `.env.example` to `.env` and adjust it to your local setup before starting the service:

```bash
cp .env.example .env
```

If the installation was problematic, ensure the Ent client and its dependencies are available:
```bash
go get entgo.io/ent/cmd/ent
go get entgo.io/ent/cmd/internal/printer@v0.14.4
```

Start development afterwards with:

```bash
make dev
```

### Other useful targets
- `make infra-up` / `make infra-down` / `make infra-logs` — start/stop/follow local Postgres, Redis, and the Kafka cluster only
- `make up` / `make down` / `make logs` — start/stop/follow the full Docker stack (infra + the service itself, hot-reloaded via `air`)
- `make build` / `make run` — build the binary, or build and run it directly
- `make generate-keys` — generate a local Ed25519 JWT keypair
- `make generate-db-certs` — generate a local CA/server/client TLS cert chain for Postgres
- `make generate-ent` / `make generate-mocks` / `make generate-swagger` — regenerate Ent models, mocks, and Swagger docs
- `make lint` / `make lint-fix` — run `golangci-lint`
- `make clean` / `make nuke` — remove build artifacts, or also drop local infrastructure volumes

Run `make help` to list every available target.

## Configuration

### Environment Variables
The service reads configuration from a `.env` file at the repo root (see `.env.example` for the full list, consumed via Viper):

```bash
API_VERSION="v1"
API_PORT=":8000"
ENVIRONMENT="development"
LOG_LEVEL="debug"

ENCRYPTION_SECRET_KEY="<32-byte secret key>"
JWT_PRIVATE_KEY_PATH="private_key.pem"
JWT_PUBLIC_KEY_PATH="public_key.pem"

DB_USER="users_service"
DB_PASSWORD="users_service_password"
DB_HOST="localhost"
DB_PORT="5434"
DB_NAME="users_db"
DB_SSL_CA_PATH="docker/certs/ca.crt"
DB_SSL_CERT_PATH="docker/certs/client.crt"
DB_SSL_KEY_PATH="docker/certs/client.key"

REDIS_HOST="localhost"
REDIS_PORT="6381"

KAFKA_BOOTSTRAP_SERVERS="localhost:9093,localhost:9095,localhost:9097"
KAFKA_SECURITY_PROTOCOL="PLAINTEXT"
```

`make infra-up` (and `make up`) provision Postgres (TLS-only), Redis, and a local 3-broker Kafka cluster (KRaft mode, no Zookeeper) matching these defaults — see `docker/docker-compose.local.yml` and `docker/docker-compose.yml`. Postgres and Redis are exposed on non-default host ports (`5434`, `6381`) to avoid clashing with other local stacks.

### Generating JWT keys and TLS certificates
```bash
make generate-keys
make generate-db-certs
```
`generate-keys` writes `private_key.pem` and `public_key.pem` to the repo root (gitignored), matching `JWT_PRIVATE_KEY_PATH` / `JWT_PUBLIC_KEY_PATH`. `generate-db-certs` writes a local CA plus server/client certs to `docker/certs/` (gitignored) and fixes the server key ownership so Postgres accepts it; the app connects via `sslmode=verify-full`, so this must run before `make infra-up` brings up Postgres for the first time — `make setup` already does both in order.

## Project Structure
```bash
.
├── app
│   ├── cmd
│   ├── config
│   ├── controllers
│   ├── docs
│   ├── infrastructure
│   ├── middlewares
│   ├── models
│   │   ├── ent
│   │   ├── events
│   │   ├── handlers
│   │   ├── requests
│   │   ├── responses
│   │   └── utils
│   ├── repositories
│   ├── routes
│   ├── services
│   └── utils
├── docker                 # Dockerfile (prod), Dockerfile.dev (local, hot-reloaded), compose files
├── tests
│   ├── mocks
│   ├── registry
│   ├── resources
│   │   ├── e2e
│   │   ├── integration
│   │   └── unit
│   └── settings
├── .air.toml               # hot-reload config used by `make dev` / `make up`
├── .env.example             # local environment template
├── .golangci.yml             # lint configuration used by `make lint`
└── Makefile                   # setup, dev, build, docker, infra, generate, test, lint targets
```

## Development and Testing

### Running tests
- **Unit tests**: `make test-unit`
- **Integration tests** (requires Docker, uses Testcontainers): `make test-integration`
- **End-to-end tests** (requires Docker): `make test-e2e`
- **Everything**: `make test`

### Defining Tests in JSON

The test runner is data-driven, meaning test cases are defined in `.json` files located in the `tests/resources/` directory. Each JSON file represents a test suite.

#### Test Setup (`setup_data`)

The `setup_data` array is used to populate the database with a specific state before a test case runs.

**Example Structure:**
```json
"setup_data": [
  {
    "model": "UserRole",
    "data": {
      "id": 1,
      "name": "admin"
    }
  },
  {
    "model": "User",
    "data": {
      "id": 1,
      "mail": "admin@example.com",
      "password": "securepassword123",
      "user_role_ids": [1]
    }
  }
]
```

**Key Conventions:**
- **`model`**: The name of the `ent` model to create (e.g., "User", "UserRole").
- **`data`**: A map of fields and their values for the new entity.
- **`id`**: A **local, temporary ID** used only within this file to define relationships. It does not correspond to the actual database ID.
- **Relationships (to-one)**: To link to a single entity, use the `_id` suffix (e.g., `"owner_id": 1`). This links to the entity that has the local ID of `1`.
- **Relationships (to-many)**: To link to multiple entities, use the `_ids` suffix (e.g., `"user_role_ids": [1, 2]`). This links to entities with local IDs `1` and `2`.
- **Order Matters**: Entities must be defined before they are referenced. For example, `UserRole` must be defined before a `User` can be assigned to it.

#### Test Steps (`steps`)

The `steps` array defines the sequence of actions and assertions for a test case.

**Example Structure:**
```json
{
   "setup_data": [
         {
            "model": "UserRole",
            "data": {
            "id": 1,
            "name": "user",
            "description": "Standard user role"
            }
         }
      ],
   "steps": [
      {
         "type": "executeFunction",
         "description": "Should return true for a valid email",
         "params": {
            "function_name": "utils.CheckEmailFormat",
            "args": ["test@example.com"]
         },
         "expected": {
            "return_value": true
         }
      },
      {
         "type": "httpRequest",
         "description": "Attempt to register a new user",
         "params": {
            "method": "POST",
            "path": "/api/v1/auth/register",
            "body": {
            "mail": "new.user@example.com",
            "password": "Password123!"
            }
         },
         "expected": {
            "statusCode": 200
         }
      },
      {
         "type": "httpRequest",
         "description": "Update the authenticated user's basic info",
         "params": {
            "method": "PATCH",
            "path": "/api/v1/users/update",
            "headers": {
               "Authorization": "Bearer {{loginResult.data.token}}"
            },
            "body": {
               "name": "Updated Name"
            }
         },
         "expected": {
            "statusCode": 200
         }
      }
   ]
}
```
**Key Fields:**
- **`type`**: The type of action to perform. Supported types are `executeFunction`, `httpRequest`, `dbRowCount`, and `dbAttributeEquals`.
- **`description`**: A human-readable description of the step.
- **`params`**: The parameters for the action (e.g., function name and arguments, or HTTP method/path/body/headers).
- **`expected`**: The expected outcome (e.g., a specific return value or HTTP status code).
- **`store_result_as`**: (Optional) A key to store the result of this step, allowing it to be used as a dependency in subsequent steps (e.g., `{{myResult.data.id}}`).

`httpRequest` steps accept an optional `headers` map (e.g. for `Authorization: Bearer {{...}}`); both `body` and `headers` values support `{{dependency.path}}` placeholders resolved from prior steps' stored results.

## Manually Building, Tagging, and Pushing Docker Images
- **Building the production Docker image**
   ```bash
      make build-docker
   ```
   (equivalent to `docker build -t ghcr.io/factory-chainline/users-service:{current-version} -f ./docker/Dockerfile .`)

- **Tagging a Docker image**
   ```bash
      docker tag ghcr.io/factory-chainline/users-service:<source> ghcr.io/factory-chainline/users-service:<target>
   ```

- **Pushing a Docker image**
   ```bash
      make docker-login
      make docker-push
   ```

## Troubleshooting
- **Continuous Integration Actions Failed**:
  Ensure your code passed tests locally before commit. It will be checked anyway, but don't waste our common memory.
- **Docker not running**:
  Start Docker before running `make infra-up`, `make up`, or the integration/e2e tests.
- **Manifest Errors**:
  Ensure all `settings.yaml` in the `tests/settings` folder are properly configured for your local environment.

## Useful Commands and Addresses
- **Hosting documentation locally**
   ```bash
      godoc -http=:6060
   ```
- **Forwarding users-service port without ingress**
   ```bash
      kubectl port-forward svc/users-service -n users-service 8000:8000
   ```
- **Swagger address**: `localhost:8000/swagger`
- **Redoc address**: `localhost:8000/redoc`

## Updates
![Alt](https://repobeats.axiom.co/api/embed/0ccd32c70b973168f94e36384a64155cb9867f07.svg "Repobeats analytics image")

## License
© 2025 Robert Moń, integrate it whenever you want for non commercial purposes.
