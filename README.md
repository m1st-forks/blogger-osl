# blogger API

Run locally

- Requires Go 1.21+.
- From the `src` directory:

```sh
cd src
go build
./blogger
```

The server listens on `:$PORT` (default `8080`).

Persistence

- Set `POSTS_JSON` to change the file used for storage (default: `posts.json` in CWD).
- Data is written atomically using a temporary file and rename.
- On startup, data is loaded from the JSON file if it exists.

API

- GET `/api/posts`
    - Returns a list of posts.

- GET `/api/posts/:id`
    - Returns a single post by id.

- POST `/api/posts`
    - Create a new post.
    - Body:

```json
{ "author": "name", "title": "title", "description": "text" }
```

- PATCH `/api/posts/:id`
    - Update any subset of fields.
    - Body examples:

```json
{ "title": "new value" }
```

```json
{ "author": "bob", "description": "updated", "content": "# New\n\nBody" }
```

- DELETE `/api/posts/:id`
    Delete a post.

Quick curl examples

```sh
# list
curl -s http://localhost:$PORT/api/posts

# create
curl -s -X POST http://localhost:$PORT/api/posts \
    -H 'Content-Type: application/json' \
    -d '{"author":"alice","title":"hello","description":"first"}'

# get by id
curl -s http://localhost:$PORT/api/posts/1

# patch
curl -s -X PATCH http://localhost:$PORT/api/posts/1 \
    -H 'Content-Type: application/json' \
    -d '{"parts":{"title":"Hello","description":"updated"}}'

# delete
curl -i -X DELETE http://localhost:$PORT/api/posts/1
```

## src/env

```txt
BASE_URL=https://blog.warpdrive.team // what url this is being hosted on
DISCORD_WEBHOOK_URL= // used for forwarding to discord
```
