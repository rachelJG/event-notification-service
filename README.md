# Event Notification Service


A high-performance, scalable backend service built in Go for processing domain events and delivering notifications asynchronously across multiple channels.

## Features

- **Event-Driven Architecture**: Built with event sourcing in mind for reliable message delivery
- **Multiple Notification Channels**: Supports various notification methods
- **Scalable**: Designed to handle high throughput with horizontal scaling
- **Reliable**: Implements retry mechanisms and dead-letter queues
- **Easy to Integrate**: Simple HTTP API for event submission

## Getting Started

## Tech Stack

- **Go** - Core programming language
- **PostgreSQL** - Primary database
- **Docker** - Containerization
- **HTTP APIs** - RESTful interface for communication

## Running Locally with Docker

### Prerequisites
- Docker and Docker Compose installed

### Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/rachelJG/event-notification-service.git
   cd event-notification-service
   ```

2. Copy the example environment file and update it with your configuration:
   ```bash
   cp .env.example .env
   # Edit .env file with your configuration
   ```

3. Start the services using Docker Compose:
   ```bash
   docker-compose up -d
   ```

4. The service should now be running at `http://localhost:8080`

### Development

For development, you can run the application directly with Go:

```bash
# Install dependencies
go mod download

# Run the application
go run main.go
```

## Supported Events

The service currently supports the following domain events:

- **UserRegistered**: Triggered when a new user registers
- **PasswordResetRequested**: Triggered when a user requests a password reset
- **OrderPaid**: Triggered when an order payment is completed
- **OrderShipped**: Triggered when an order is shipped

## Supported Channels

Notifications can be delivered through the following channels:

- **Email**: Send notifications via SMTP
- **Webhook**: Deliver notifications to configured webhook endpoints

The system is designed to be easily extensible, allowing new events and channels to be added without changes to the core processing logic.

## API Documentation

### Submit Event

```http
POST /api/events
Content-Type: application/json

{
  "event_type": "UserRegistered",
  "payload": {
    "user_id": "12345",
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

## Configuration

Configuration is done through environment variables:

```env
REDIS_ADDR=localhost:6379
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user@example.com
SMTP_PASSWORD=yourpassword
API_PORT=8080
```

## Acknowledgments

- Built with using Go
- Inspired by event-driven architecture patterns
- Thanks to all contributors who have helped shape this project