# User Favourites Service

## Scope
A web server that lets users manage their favourite assets including charts, insights and audiences.  
Supports listing, adding, removing and editing favourites.

## How to Run

### Run locally
To build and run the program:
```bash
go run main.go
```
Or, if you want to build first:
```bash
go build -o favourites
./favourites
```

### Run with Docker
```bash
docker build -t favourites .
docker run --rm -p 8080:8080 favourites
```

## How to Test
Tests cover basic functionality and a few error scenarios.

Run them with:
```bash
go test ./...
```

## Usage examples
- Add a favourite  
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"assetId":"chart-42", "assetType":"chart", "description":"Top sales", "assetData":{"title":"Sales Q4","axes":["month","revenue"]}}' \
  http://localhost:8080/favourites/user123
```
`Response 201, with created entry including generated id`

```
{
  "id": "fav-1",
  "userId": "user123",
  "assetId": "chart-42",
  "assetType": "chart",
  "description": "Top sales",
  "assetData": { "title": "Sales Q4", "axes": ["month","revenue"] },
  "createdAt": "2025-09-11T17:18:53.9766385+03:00",
  "updatedAt": "2025-09-11T17:18:53.9766385+03:00"
}
```

- List favourites for a user  
```bash
curl -s http://localhost:8080/favourites/user123 | jq
```
`Response 200`

  ```
  [
    {
      "id": "fav-1",
      "userId": "user123",
      "assetId": "chart-42",
      "assetType": "chart",
      "description": "Top sales",
      "assetData": { "title": "Sales Q4", "axes": ["month","revenue"] },
      "createdAt": "2025-09-11T17:18:53.9766385+03:00",
      "updatedAt": "2025-09-11T17:18:53.9766385+03:00"
    }
  ]
  ```

- Update a favourite  
```bash
curl -X PUT -H "Content-Type: application/json" \
  -d '{"assetId":"chart-21", "assetType":"chart", "description":"Sales frequency", "assetData":{"title":"Annual retention","axes":["month","invoice_count"]}}' \
  http://localhost:8080/favourites/user123/fav-1
```

- Delete a favourite  
```bash
curl -X DELETE http://localhost:8080/favourites/user123/fav-1
```

## Assumptions
- REST API endpoints to fetch, add, remove and update assets in favourites list.
- JSON request/response with lower-camel JSON keys (id, userId, assetId, assetType, description, assetData, createdAt, updatedAt).
- In-memory store for the challenge purposes. No data store persistence present. Stored data are lost on restart.
- Tests cover store methods, HTTP handlers, validation, conflicts and error paths (including with a mock store).

## Data model
- This service stores references to existing platform assets. The source of truth for assets lives upstream.
- `description` is a user-provided label for the favourite and may be empty.
- `assetData` stores the upstream asset JSON as-is (flexible); on updates, clients send the full updated asset JSON. The service does not validate against upstream schemas.
- On reads, the service returns stored favourites. If upstream assets are missing, clients are expected to avoid selecting them; future versions may exclude or flag missing assets when an upstream lookup is enabled.

## CI/CD (suggested checks)
In a CI pipeline, you can run basic quality gates before building and deploying:

```bash
# Format code
go fmt ./...

# Static analysis
go vet ./...
staticcheck ./...

# Run tests
go test ./...
```

## Next steps
- Solution is based on an in-memory store which makes storage ephemeral and works for a single instance only. Persistent storage should be used, for instance Postgres JSONB, or adopt a platform-wide storage solution.
- Make use of request context for better efficiency, especially when a persistent store is integrated.
- Coordinate upstream contracts for asset verification and potential reconciliation (exclusion/flags for missing assets) once teams align.
- Performance: caching per user with Redis since it is shown on the front page.
- Performance: pagination for very large lists.
