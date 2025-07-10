# Wallet Application

A centralized wallet application built with Go, providing secure financial operations including deposits, withdrawals, transfers, balance checks, and transaction history.

## Features

- **User Management**: Create and manage user accounts
- **Wallet Operations**: 
  - Deposit funds to user wallets
  - Withdraw funds from user wallets
  - Transfer funds between users
  - Check wallet balance
  - View transaction history
- **Structured Logging**: Comprehensive logging with logrus
- **Database**: PostgreSQL for persistent storage
- **RESTful API**: Clean, RESTful endpoints
- **Testing**: Comprehensive unit and integration tests

## Tech Stack

- **Backend**: Go 1.24.4
- **Database**: PostgreSQL 15
- **Web Framework**: Gin
- **Database Driver**: pgx
- **Database Migration**: golang-migrate
- **Logging**: Logrus
- **Testing**: Testify
- **Mocking**: pgxmock

## Prerequisites

- Go 1.24.4 or higher
- PostgreSQL 15 or higher
- Make (optional, for using Makefile)

## Quick Start

### 1. Clone the Repository

```bash
git clone git@github.com:SauceFoong/walletapp.git
cd walletapp
```

### 2. Set Up Database

#### Using Docker

```bash
docker run --name wallet-postgres \
  -e POSTGRES_USER=walletuser \
  -e POSTGRES_PASSWORD=walletpass \
  -e POSTGRES_DB=walletdb \
  -p 5432:5432 \
  -d postgres:15
```

### 3. Configure Environment Variables

Create a `.env` file in the root directory:

```env
DATABASE_URL=postgres://walletuser:walletpass@localhost:5432/walletdb
SERVER_PORT=8080
```

### 4. Install Dependencies

```bash
go mod download
```

### 5. Generate API Documentation (Swagger)

```bash
# Install swag CLI if not already installed
go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger documentation, the command was included in the makefile
swag init --generalInfo cmd/app/main.go --output docs
```

### 6. Run Database migration

```bash
# Run migrations
migrate -database "postgres://walletuser:walletpass@localhost:5432/walletdb?sslmode=disable" -path ./migrations up


# Rollback all migrations ( If needed )
migrate -database "postgres://walletuser:walletpass@localhost:5432/walletdb?sslmode=disable" -path ./migrations down

```

Note: You may use a DB client (ex: DBeaver / Pgadmin) for better visibility on the database

### 7. Run the Application

```bash
# Using make
make run

# Or directly
go run cmd/app/main.go
```

The server will start on `http://localhost:8080` (Depends on the sever port you set)

## API Documentation

### Base URL
```
http://localhost:8080/api
```

### Interactive Swagger UI

Once the application is running, you can access the interactive API documentation at:
```
http://localhost:8080/swagger/index.html
```

### Authentication
Currently, the API uses user ID for identification. There is no JWT authentication yet.

### Endpoints

#### User Management

**Create User**
```http
POST v1/users
Content-Type: application/json

{
  "email": "johndoe@gmail.com",
  "first_name": "John", 
  "last_name": "Doe",
  "password": "password",
  "username": "johndoe"
}
```
1. Email and Username have to be unique.
2. User wallet will be created automatically during account creation.

**Get User**
```http
GET v1/users/{user_id}
```

Example Response: 
```json
{
  "code": 200,
  "message": "User retrieved successfully",
  "data": {
    "id": "652242c0-d72b-4f75-bacf-a72ade1bedda",
    "username": "johndoe",
    "first_name": "John",
    "last_name": "Doe",
    "email": "johndoe@gmail.com",
    "created_at": "2025-07-10T03:52:21.61777Z",
    "updated_at": "2025-07-10T03:52:21.61777Z",
    "wallet": {
      "id": "e0e92a6b-68cd-42f4-8bcc-8de015da71ad",
      "balance": 998.98,
      "created_at": "2025-07-10T03:52:21.623879Z",
      "updated_at": "2025-07-10T03:55:30.299644Z"
    }
  }
}
```

**Get All Users**
```http
GET v1/users
```

Example Response: 
```json
{
  "code": 200,
  "message": "Users retrieved successfully",
  "data": [
    {
      "id": "fb20cfcc-a065-403e-8654-a9c764138afc",
      "username": "johndoe",
      "first_name": "john",
      "last_name": "doe",
      "email": "johndoe@gmail.com",
      "created_at": "2025-07-09T16:04:48.83166Z",
      "updated_at": "2025-07-09T16:04:48.83166Z",
      "wallet": {
        "id": "1d15d947-8882-44be-881d-f950909b4c51",
        "balance": 998.98,
        "created_at": "2025-07-09T16:04:48.835457Z",
        "updated_at": "2025-07-10T03:19:18.740027Z"
      }
    },
    {
      "id": "66276621-6d2d-4bb1-9563-42fa8f8d1a4e",
      "username": "janedoe",
      "first_name": "Jane",
      "last_name": "Doe",
      "email": "janedoe@gmail.com",
      "created_at": "2025-07-09T16:05:40.466196Z",
      "updated_at": "2025-07-09T16:05:40.466196Z",
      "wallet": {
        "id": "2fc92397-1bb5-41ab-a25e-b1303046e502",
        "balance": 3.61,
        "created_at": "2025-07-09T16:05:40.467579Z",
        "updated_at": "2025-07-09T16:25:26.835714Z"
      }
    }
  ]
}
```

#### Wallet Operations

**Get Wallet Balance**
```http
GET /wallets/{user_id}/balance
```

Example Response:
```json
{
  "code": 200,
  "message": "Balance retrieved successfully",
  "data": {
    "user_id": "652242c0-d72b-4f75-bacf-a72ade1bedda",
    "balance": 999.99
  }
}
```

**Deposit to Wallet**
```http
POST /wallets/{user_id}/deposit
Content-Type: application/json

{
    "amount": 100.00 (Deposit amount)
}
```

**Withdraw from Wallet**
```http
POST /wallets/{user_id}/withdraw
Content-Type: application/json

{
    "amount": 50.00 (Withdraw amount) 
}
```

**Transfer Between Users**
```http
POST /transfers
Content-Type: application/json

{
    "from_user_id": "user123",
    "to_user_id": "user456",
    "amount": 25.00
}
```

#### Transaction History

**Get User Transactions**
```http
GET /users/{user_id}/transactions
```

Example Response:
```json
{
  "code": 200,
  "message": "Transaction history retrieved successfully",
  "data": [
    {
      "id": "33ed29c7-3ed2-4aa6-9365-e70ccde136f7",
      "wallet_id": "e0e92a6b-68cd-42f4-8bcc-8de015da71ad",
      "type": "TRANSFER_OUT",
      "amount": 1,
      "related_user_id": "66276621-6d2d-4bb1-9563-42fa8f8d1a4e",
      "created_at": "2025-07-10T03:55:30.299644Z",
      "updated_at": "2025-07-10T03:55:30.299644Z"
    },
    {
      "id": "2a2faf5c-5a5b-4578-adac-2bfdcebf3b0c",
      "wallet_id": "e0e92a6b-68cd-42f4-8bcc-8de015da71ad",
      "type": "WITHDRAW",
      "amount": 0.01,
      "created_at": "2025-07-10T03:55:02.971879Z",
      "updated_at": "2025-07-10T03:55:02.971879Z"
    },
    {
      "id": "f2e94afc-01d9-4200-a9bf-7f14a16ff6e7",
      "wallet_id": "e0e92a6b-68cd-42f4-8bcc-8de015da71ad",
      "type": "WITHDRAW",
      "amount": 0.01,
      "created_at": "2025-07-10T03:54:54.300797Z",
      "updated_at": "2025-07-10T03:54:54.300797Z"
    },
    {
      "id": "5c76195c-212a-48d9-8960-b277c47a952e",
      "wallet_id": "e0e92a6b-68cd-42f4-8bcc-8de015da71ad",
      "type": "DEPOSIT",
      "amount": 1000,
      "created_at": "2025-07-10T03:54:43.895092Z",
      "updated_at": "2025-07-10T03:54:43.895092Z"
    }
  ]
}

```

## Database Schema
![ERD Diagram](erd-diagram.png)

### Users Table
```sql
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Wallets Table
```sql
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance NUMERIC(20,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);
```

### Transactions Table
```sql
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- 'DEPOSIT', 'WITHDRAW', 'TRANSFER_IN', 'TRANSFER_OUT'
    amount NUMERIC(20,2) NOT NULL,
    related_user_id UUID, -- for transfers, the other user involved
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## Testing

### Run All Tests
```bash
make test
```

### Test Structure
- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test database operations and service layer

## Development

### Project Structure
```
walletapp/
├── cmd/app/           # Application entry point
├── internal/          # Private application code
│   ├── db/           # Database connection
│   ├── handlers/     # HTTP handlers
│   ├── logger/       # Logging configuration
│   ├── models/       # Data models
│   ├── repositories/ # Data access layer
│   └── services/     # Business logic
├── migrations/       # Database migration files
├── docs/            # Swagger Documentation
└── Makefile         # Commands for quick run & test
```

### Make Commands For Quick Run & Test

```bash
make run            # Run the application
make test           # Run all tests
```

### Adding New Features

1. **Models**: Add new data structures in `internal/models/`
2. **Repositories**: Add data access logic in `internal/repositories/`
3. **Services**: Add business logic in `internal/services/`
4. **Handlers**: Add HTTP endpoints in `internal/handlers/`
5. **Migrations**: Add database schema changes in `migrations/`

## Logging

The application uses structured logging with logrus. Log levels:

- **DEBUG**: Detailed debugging information
- **INFO**: General application flow
- **WARN**: Warning conditions
- **ERROR**: Error conditions

## Error Handling

The application implements comprehensive error handling:

- **HTTP Status Codes**: Proper status codes for different scenarios
- **Readable Error Messages**: Human-readable error messages
- **Logging**: All errors are logged with context

## Security Considerations

- **Input Validation**: All inputs are validated
- **Error Sanitization**: Sensitive information is not exposed in errors
- **Race Condition Prevention**: Using `SELECT ... FOR UPDATE` to lock wallet rows during transactions
- **Ensure transaction atomicity**: Deposits, withdrawals, and transfers are atomic - they either complete entirely or rollback completely
- **Min & Max Cap for fund transfer**: Each transaction must be between $0.01 and $1,000,000 to prevent misuse.

##  Project Overview

###  1. Requirements Checklist

#### Functional Requirements
1. **Deposit to User Wallet**: Allow users to deposit funds into their wallet. [✅]
2. **Withdraw from User Wallet**: Allow users to withdraw funds from their wallet. [✅]
3. **Transfer Between Users**: Enable fund transfers between different users. [✅]
4. **Check User Balance**: Provide balance inquiry functionality. [✅]
5. **View Transaction History**: Display user's transaction history. [✅]

#### Non Functional Requirements
1. **Data Integrity**: All wallet operations (deposit, withdraw, transfer) must be ACID-compliant and atomic. [✅]
2. **Race Condition Prevention**: Concurrent transactions must not cause inconsistent balances. [✅]
3. **Error Handling**: Sensitive information must not be exposed in error messages. [✅]
4. **Input Validation**: User input was validated. [✅]

###  2. Technical Decisions

#### Architecture Choices
- **Gin Framework**: Chose Gin for its lightweight and simplicity for developemnt
- **Clean Architecture**: Implemented layered architecture (handlers → services → repositories) for maintainability
- **Dependency Injection**: Used interfaces and DI for better testability
- **Structured Logging**: Implemented logrus for production-ready logging to improve observability

#### Database Design
- **PostgreSQL**: Chose for ACID compliance and financial transaction safety
- **UUID Primary Keys**: Used UUIDs as entity primary keys for security purpose
- **1:1 relationship between User & Wallet**: For simplicity, a user can only have one wallet. The wallet will created during account creation.

#### Testing Strategy
- **Unit Tests**: Mocked dependencies for isolated testing
- **Integration Tests**: Real database tests for end-to-end validation
- **pgxmock**: Used for database mocking in unit test

### 3. Code Review Guide

#### Where to Start
1. **Entry Point**: `cmd/app/main.go` - Application setup and routing
2. **API Layer**: `internal/handlers/` - HTTP request handling
3. **Business Logic**: `internal/services/` - Core business logics
4. **Data Layer**: `internal/repositories/` - Database interactions

#### Key Design Patterns
- **Dependency Injection**: See `internal/services/` for interface implementations
- **Repository Pattern**: Data access abstraction in `internal/repositories/`
- **Service Layer**: Business logic encapsulation
- **Response Handling**: Consistent sucess & error responses in `internal/models/`

#### Testing Approach
- **Unit Tests**: `*_test.go` files alongside source code
- **Integration Tests**: `*_integration_test.go` files

###  4. Areas for improvement

#### Features / Changes I chose not to do in the submission but is good to have:
A. **JWT Token Authentication**: Implement JWT-based authentication for all API endpoints
  - Secure user sessions with token expiration and refresh mechanisms
  - Role-based access control for different user types

B. **Rate Limiting & Abuse Prevention**: Implement Rate Limiter with Redis
  - To rate limit critical endpoints that involve monetary movement.
  - For examaple only allow 10 requests per minute per user to prevent misuse.
  
C. **Performance & Scalability**
  - **Caching Layer**: Redis caching for frequently accessed data
    - User profile data
    - Balance caching with TTL
    - Transaction history
  - **Database Optimization**: Query optimization and indexing strategies

###  5. Time spent on this project
Dedicated 4 hours each day across 3 days, resulting in a total of 12 hours.



