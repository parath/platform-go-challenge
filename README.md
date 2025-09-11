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

## Usage ecamples
Add a favourite  
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"assetId":"chart-42", "assetType":"chart", "description":"Top sales", "metadata":{"title":"Sales Q4","axes":["month","revenue"]}}' \
  http://localhost:8080/favourites/user123
```
`Response 201, with created entry including generated id`

```
{
  "id": "fav-1",
  "assetId": "chart-42",
  "assetType": "chart",
  "description": "Top sales",
  "metadata": { "title": "Sales Q4", "axes": ["month","revenue"] },
  "createdAt": "2025-09-11T17:18:53.9766385+03:00"
}
```

List of favourites per user  
```bash
curl -s http://localhost:8080/favourites/user123 | jq
```
`Response 200`

  ```
  [
    {
      "id": "fav-1",
      "assetId": "chart-42",
      "assetType": "chart",
      "description": "Top sales",
      "metadata": { "title": "Sales Q4", "axes": ["month","revenue"] },
      "createdAt": "2025-09-11T17:18:53.9766385+03:00"
    }
  ]
  ```

Update an asset in favourites  
```bash
curl -X PUT -H "Content-Type: application/json" \
  -d '{"assetId":"chart-21", "assetType":"chart", "description":"Sales frequency", "metadata":{"title":"Annual retention","axes":["month","invoice_count"]}}' \
  http://localhost:8080/favourites/user123/fav-1
```

Delete a favourite  
```bash
curl -X DELETE http://localhost:8080/favourites/USER123/fav-1
```

## Assumptions
- REST API endpoints to fetch, add, remove and update assets in favourites list
- JSON request/response
- In-memory store for the challenge purposes
- Unit tests for GET, POST, PUT, PATCH and DELETE verbs
- assetId is handled by other platform services. Favourites service does not validate asset existence.

## Next steps
- Ephemeral storage, currently in memory, to chaνge into platform-wide storage service(s)
- Scalability issues: pagination for large lists, persistent storage (Redis/Postgres JSONB), caching 
