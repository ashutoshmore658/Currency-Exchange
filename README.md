

# Exchange Rate Service

A backend service to fetch real-time and historical currency exchange rates, perform currency conversions, and support efficient caching and validation.  
Supports USD, INR, EUR, JPY, and GBP.

---

## How to Run the Project

### **1. Prerequisites**

- [Go 1.24+](https://golang.org/dl/)
- [Docker](https://www.docker.com/) and [docker-compose](https://docs.docker.com/compose/)
- (Optional) [Postman](https://www.postman.com/) or `curl` for testing

### **2. Clone the Repository**

```sh
git clone https://github.com/ashutoshmore658/Currency-Exchange
cd Currency-Exchange
```

### **3. Start the Service**

The project uses Redis for caching and requires network access to the exchange rate API.

```sh
sudo docker-compose up --build

```

```markdown
### Troubleshooting: Rebuilding the Docker Environment

If you encounter issues or want to start fresh, especially when using an older Docker version, follow these steps to clean up and rebuild your Docker environment:

```
# Stop and remove containers, volumes, images, and orphaned containers
```sh
sudo docker-compose down -v --rmi all --remove-orphans

```

# Remove unused Docker data to free up space

```sh
sudo docker system prune -f

```

# Rebuild and start the containers

```sh
sudo docker-compose up --build
```
```

This will ensure a clean slate by removing all related containers, volumes, images, and dangling resources before rebuilding your project.



This will:
- Start a Redis instance
- Build and run the Go service

The service will be available at:  
`http://localhost:8080` (`8080` or `any other port`, which you set in your env's)

### **4. Environment Variables**

The service can be configured using the following environment variables. You can set these in a `.env` file or directly in your environment.
---------------------------------------------------------------------------------------------------------------
| Variable               | Description                                       | Example                         |
|------------------------|---------------------------------------------------|---------------------------------|
| `SERVER_PORT`          | Port on which the service runs                    | `8080`                          |
| `EXTERNAL_API_URL`     | URL of the external exchange rate API             | `https://api.frankfurter.app`   |
| `LATEST_RATE_CACHE_TTL`| Time-to-live for caching latest exchange rates    | `1h` (1 hour)                   |
| `HISTORICAL_CACHE_TTL` | Time-to-live for caching historical rates         | `24h` (24 hours)                |
| `REFRESH_INTERVAL`     | Interval for background refresh of latest rates   | `1h`                            |
| `HISTORY_DAYS_LIMIT`   | Maximum number of days allowed for historical data| `90`                            |
| `REDIS_ADDR`           | Redis server address                              | `localhost:6379`                |
| `REDIS_PASSWORD`       | Password for Redis (if any)                       | `yourpassword`                  |
| `REDIS_DB`             | Redis database number                             | `0`                             |
| `DATE_FMT`             | Date format used throughout the service           | `2006-01-02`                    |
----------------------------------------------------------------------------------------------------------------
---
**Note:**  
If you are too lazy to change environment variables, no problem!  
We have set sensible default values, so you can directly use the service by building and running it with Docker Compose without any additional configuration.

Make sure to reload your environment or restart your Docker containers after changing these variables if you do customize them.
---

## How to Test the Endpoints

### **1. Fetch Latest Exchange Rate**

**cURL Request:**
```sh
curl --location 'http://localhost:8080/v1/latest?base=EUR&symbol=JPY'
```
**Response:**
```json
{
    "base": "EUR",
    "rates": {
        "EUR": 1,
        "JPY": 162.89
    },
    "timestamp": 1746576000
}
```

---

### **2. Convert Currency**

**cURL Request with historical date value as parameter:**
```sh
curl --location 'http://localhost:8080/v1/convert?from=USD&to=INR&amount=100&date=2025-04-14'
```
**Response:**
```json
{
    "from": "USD",
    "to": "INR",
    "amount": 100,
    "convertedAmount": 8599,
    "rate": 85.99,
    "onDate": "2025-04-14T00:00:00Z"
}
```

**cURL Request without historical date value as parameter (API will work in latest exchange rate mode):**
```sh
curl --location 'http://localhost:8080/v1/convert?from=USD&to=INR&amount=100'
```
**Response:**
```json
{
    "from": "USD",
    "to": "INR",
    "amount": 100,
    "convertedAmount": 8476,
    "rate": 84.76
}
```

---

### **3. Get Historical Rates**

**Request:**
```sh
curl --location 'http://localhost:8080/v1/historical?base=USD&symbol=INR&startDate=2025-04-05&endDate=2025-04-10'
```
**Response:**
```json
{
    "base": "USD",
    "rates": {
        "2025-04-04T00:00:00Z": 85.4,
        "2025-04-07T00:00:00Z": 85.77,
        "2025-04-08T00:00:00Z": 86.22,
        "2025-04-09T00:00:00Z": 86.67,
        "2025-04-10T00:00:00Z": 86.16
    },
    "amount": 1,
    "target": "INR"
}
```

---

### **4. Error Handling Example**

If you request a date older than 90 days:

```sh
curl --location 'http://localhost:8080/v1/historical?base=USD&symbol=INR&startDate=2025-01-05&endDate=2025-01-10'
```
**Response:**
```json
{
    "error": {
        "code": "Bad Request",
        "message": "requested date is older than 90 days"
    }
}
```
---
**Note:**  There are plenty of other toxic combinations like invalid format of date, unsupported currencies, same currency in base and target currency, 
more than one parameters of same type, missing parameters etc. User can try them out using and making changes to the above cURL's.

---

### **5. Using Postman**

- Import the above URLs as GET requests.
- Set query parameters as shown in the curl examples.

---

## Assumptions

- **Supported Currencies:** Only USD, INR, EUR, JPY, GBP are supported. Requests for other currencies will return a 400 error.
- **Historical Data Limit:** Only the last 90 days of historical data are available. Older dates return an error.
- **Date Format:** All dates must be in `YYYY-MM-DD` format.
- **Caching:** Latest and historical rates are cached in Redis for efficiency.
- **Rate Refresh:** The service refreshes the latest rates every hour in the background.
- **Error Responses:** All validation errors return a JSON error object with a code and message.
- **API Source:** The service uses a public exchange rate API (e.g., exchangerate.host) or a mock for testing.
- **Single Target Currency:** Only one target currency per request is supported for `/latest`, `/historical` and `/convert`.
- **Conversion Using Same Currency in Base and Target:** Will throw a 400 bad request, as noone can convert a currency to itself.

---

## Running Tests

To run all unit tests and see coverage:

```sh
go test ./... -cover
```

---

## Contact

For any issues or questions, please contact [ashutoshmoreofficial@gmail.com | +91 7420008138].

---

**Enjoy using the Exchange Rate Service!**

---