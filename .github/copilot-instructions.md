# AI Coding Agent Instructions for `tdi_evid`

## Project Overview
This repository is a Go-based project with a backend and frontend structure. The backend is organized into several key directories:

- **`cmd/api`**: Contains the API server implementation, middleware, and request handlers.
- **`models`**: Defines data models used across the application.
- **`repository`**: Implements data access logic, interacting with the database.
- **`token`**: Handles token creation and validation (e.g., JWT, Paseto).
- **`util`**: Provides utility functions for configuration, password hashing, and random data generation.
- **`migrations`**: Contains SQL migration scripts for database schema management.

The backend communicates with a database and exposes APIs for the frontend. The frontend directory is currently empty or not implemented.

## Key Workflows

### Building the Project
To build the backend, navigate to the `backend` directory and run:
```powershell
go build
```
This will generate an executable for the backend service.

### Running the Project
To run the backend server locally:
```powershell
go run main.go
```
Ensure the `app.env` file is correctly configured with environment variables.

### Testing
Unit tests are located alongside the implementation files. Run all tests with:
```powershell
go test ./...
```

### Database Migrations
Manage database schema using the migration scripts in the `migrations` directory. Use the `migrate` tool to apply migrations:
```powershell
migrate -path migrations -database "<DB_URL>" up
```
Replace `<DB_URL>` with the appropriate database connection string.

## Project-Specific Conventions

1. **Error Handling**: Errors are returned and logged at the appropriate level. Use `util/helpers.go` for common error-handling patterns.
2. **Dependency Injection**: Repositories and services are passed as dependencies to handlers and middleware.
3. **Token Management**: Use the `token` package for creating and verifying tokens. Examples can be found in `cmd/api/token_handler.go`.
4. **Configuration**: Application configuration is loaded from `app.env` using the `util/config.go` file.

## Integration Points

- **Database**: The `repository` package interacts with the database. Each model has a corresponding repository file (e.g., `ugo_org.go` for `ugo_org` table operations).
- **Authentication**: Token-based authentication is implemented using JWT and Paseto in the `token` package.
- **API Documentation**: Swagger documentation is available in the `docs` directory (`swagger.yaml`).

## Examples

### Adding a New API Endpoint
1. Define the request/response types in `cmd/api/api_type.go`.
2. Implement the handler in `cmd/api/<entity>.go`.
3. Register the route in `cmd/api/server.go`.

### Writing a New Migration
1. Create an `.up.sql` and `.down.sql` file in the `migrations` directory.
2. Use the `migrate` tool to apply the migration.

### Adding a New Repository
1. Create a new file in the `repository` directory.
2. Define methods for interacting with the database.
3. Follow patterns from existing repositories like `repository/users.go`.

---

This document is a starting point. Update it as the project evolves to ensure AI agents remain productive.