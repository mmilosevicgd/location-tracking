# Location tracking app

This app helps users monitor their movements and measure distances traveled. It collects location data over time and lets you calculate how far you've gone during any specific period.

## Prerequisites

- Docker and Docker Compose installed on your machine
- Internet connection to pull required images

## Services

### Location management service

This service handles user location data and provides search functionality to find nearby users.

URL | Request | Response
--- | --- | ---
POST /user/location | `{"username": "mmilosevic", "coordinates": "35.12314, 27.64532"}` | Stores the user's location. No response body.
POST /user/search | `{"coordinates": "35.12314, 27.64532", "distance": 5.6, "pageNumber": 1, "pageSize": 5}` | Returns a list of usernames within the specified distance, paginated.
GET /metrics | - | Returns Prometheus metrics for monitoring.

### Location history management service

This service calculates distances traveled by users over a specified time period.

URL | Request | Response
--- | --- | ---
POST /user/distance | `{"username": "mmilosevic", "start": "2025-01-01T00:00:00+00:00", "end": "2025-02-01T00:00:00+00:00"}` | Returns the total distance traveled by the user (in kilometers) during the specified time range.
GET /metrics | - | Returns Prometheus metrics for monitoring.

## Running the application

To start the application, ensure you are in the project root directory and run the following command:

```
$ docker compose up -d
```

This command will build and run both services. Once started, the services will be available at the following URLs:
- [`http://localhost:8080`](http://localhost:8080) - Location management service
- [`http://localhost:8081`](http://localhost:8081) - Location history management service

To stop the application, run:

```
$ docker compose down
```

For logs, use:

```
$ docker compose logs -f
```