# String Analyzer API

This project implements a RESTful API service in Go using the Gin framework. It analyzes strings and stores their computed properties, providing various endpoints for managing and querying these analyzed strings.

## Features

* **String Analysis:** Computes length, palindrome status, unique character count, word count, SHA-256 hash, and character frequency map for any given string.
* **CRUD Endpoints:**
  * `POST /strings`: Create/analyze a new string.
  * `GET /strings/{string_value}`: Retrieve properties of a specific string.
  * `GET /strings`: Retrieve all strings with robust filtering capabilities.
  * `GET /strings/filter-by-natural-language`: Filter strings using natural language queries.
  * `DELETE /strings/{string_value}`: Delete a specific string.
* **In-memory Storage:** Uses a simple in-memory map for storing analyzed strings. (Note: Data will be lost upon server restart).
* **Error Handling:** Provides meaningful HTTP status codes and JSON error responses.
* **Logging:** Basic request logging for monitoring.

## String Properties Computed

For each analyzed string, the following properties are computed and stored:

* `length`: Number of characters in the string.
* `is_palindrome`: Boolean indicating if the string reads the same forwards and backwards (case-insensitive).
* `unique_characters`: Count of distinct characters in the string.
* `word_count`: Number of words separated by whitespace.
* `sha256_hash`: SHA-256 hash of the string for unique identification (also used as the `id`).
* `character_frequency_map`: Object/dictionary mapping each character to its occurrence count.

## Setup Instructions

Follow these steps to get the API running locally.

### Prerequisites

* Go (version 1.16 or higher)

### 1. Navigate to the Project Directory

```bash
cd /Users/mide/Documents/Projects/Hng-13/Backend/stage-1/string-analyzer-api
```

### 2. Install Dependencies

The project uses Go Modules for dependency management. Install the required packages by running:

```bash
go mod tidy
```

This command will download `github.com/gin-gonic/gin` and its dependencies.

### 3. Run the Application

You can run the application directly using `go run`:

```bash
go run main.go
```

The server will start on port `8080` by default.

### 4. Environment Variables

* **`PORT`**: Specifies the port on which the server will listen. If not set, it defaults to `8080`.

#### Example using `PORT`

```bash
PORT=5000 go run main.go
```

## API Endpoints

Once the server is running, you can interact with the API using `curl` or any API client.

### 1. Create/Analyze String

Analyzes a given string and stores its properties.

* **Endpoint:** `POST /strings`
* **Content-Type:** `application/json`
* **Request Body:**
  ```json
  {
    "value": "string to analyze"
  }
  ```
* **Example Request:**
  ```bash
  curl -X POST -H "Content-Type: application/json" -d '{"value": "hello world"}' http://localhost:8080/strings
  ```
* **Success Response (201 Created):**
  ```json
  {
    "id": "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
    "value": "hello world",
    "properties": {
      "length": 11,
      "is_palindrome": false,
      "unique_characters": 8,
      "word_count": 2,
      "sha256_hash": "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
      "character_frequency_map": {
        " ": 1,
        "d": 1,
        "e": 1,
        "h": 1,
        "l": 3,
        "o": 2,
        "r": 1,
        "w": 1
      }
    },
    "created_at": "2025-08-27T10:00:00Z"
  }
  ```
* **Error Responses:**
  * `409 Conflict`: String already exists in the system.
  * `400 Bad Request`: Invalid request body or missing "value" field.
  * `422 Unprocessable Entity`: Invalid data type for "value" (must be string).

### 2. Get Specific String

Retrieves the properties of a previously analyzed string using its SHA-256 hash or the string value itself.

* **Endpoint:** `GET /strings/{string_value_or_hash}`
* **Example Request (using value):**
  ```bash
  curl http://localhost:8080/strings/hello%20world
  ```
* **Example Request (using hash):**
  ```bash
  curl http://localhost:8080/strings/b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
  ```
* **Success Response (200 OK):** (Same as `POST /strings` success response)
* **Error Response:**
  * `404 Not Found`: String does not exist in the system.

### 3. Get All Strings with Filtering

Retrieves all analyzed strings, with optional filtering based on various properties.

* **Endpoint:** `GET /strings`
* **Query Parameters:**
  * `is_palindrome`: `true`/`false`
  * `min_length`: integer (minimum string length)
  * `max_length`: integer (maximum string length)
  * `word_count`: integer (exact word count)
  * `contains_character`: string (single character to search for)
* **Example Request:**
  ```bash
  curl "http://localhost:8080/strings?is_palindrome=true&min_length=3&word_count=1&contains_character=a"
  ```
* **Success Response (200 OK):**
  ```json
  {
    "data": [
      {
        "id": "hash1",
        "value": "madam",
        "properties": {
          "length": 5,
          "is_palindrome": true,
          "unique_characters": 3,
          "word_count": 1,
          "sha256_hash": "hash1",
          "character_frequency_map": {
            "a": 2,
            "d": 1,
            "m": 2
          }
        },
        "created_at": "2025-08-27T10:05:00Z"
      }
    ],
    "count": 1,
    "filters_applied": {
      "contains_character": "a",
      "is_palindrome": true,
      "min_length": 3,
      "word_count": 1
    }
  }
  ```
* **Error Response:**
  * `400 Bad Request`: Invalid query parameter values or types.

### 4. Natural Language Filtering

Filters strings based on a natural language query.

* **Endpoint:** `GET /strings/filter-by-natural-language`
* **Query Parameters:**
  * `query`: string (natural language query)
* **Example Queries Supported:**
  * `"all single word palindromic strings"`
  * `"strings longer than 10 characters"`
  * `"palindromic strings that contain the first vowel"`
  * `"strings containing the letter z"`
  * `"strings exactly 5 characters long"`
  * `"strings with 3 words"`
* **Example Request:**
  ```bash
  curl "http://localhost:8080/strings/filter-by-natural-language?query=all%20single%20word%20palindromic%20strings"
  ```
* **Success Response (200 OK):**
  ```json
  {
    "data": [
      {
        "id": "hash1",
        "value": "madam",
        "properties": { /* ... */ },
        "created_at": "2025-08-27T10:05:00Z"
      },
      {
        "id": "hash2",
        "value": "level",
        "properties": { /* ... */ },
        "created_at": "2025-08-27T10:06:00Z"
      }
    ],
    "count": 2,
    "interpreted_query": {
      "original": "all single word palindromic strings",
      "parsed_filters": {
        "is_palindrome": true,
        "word_count": 1
      }
    }
  }
  ```
* **Error Responses:**
  * `400 Bad Request`: Unable to parse natural language query or missing `query` parameter.
  * `422 Unprocessable Entity`: Query parsed but resulted in conflicting filters (e.g., `min_length > max_length`).

### 5. Delete String

Deletes a previously analyzed string using its SHA-256 hash or the string value itself.

* **Endpoint:** `DELETE /strings/{string_value_or_hash}`
* **Example Request (using value):**
  ```bash
  curl -X DELETE http://localhost:8080/strings/hello%20world
  ```
* **Example Request (using hash):**
  ```bash
  curl -X DELETE http://localhost:8080/strings/b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
  ```
* **Success Response (204 No Content):** (Empty response body)
* **Error Response:**
  * `404 Not Found`: String does not exist in the system.

## Notes

* The current implementation uses an in-memory store (`stringStore`). For production use cases, a persistent database (like PostgreSQL, MongoDB, Redis) would be required.
* Natural language parsing is basic and relies on regular expressions. It can be extended for more complex queries and better NLP capabilities.
* Error messages could be more detailed for certain scenarios.
* Consider adding unit tests for the analysis functions and integration tests for the API endpoints.
