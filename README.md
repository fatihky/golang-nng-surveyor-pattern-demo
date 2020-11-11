# Nanomsg New Generation surveyor pattern demo

Nanomsg New Generation surveyor pattern demo with docker and docker-compose.

## Local
```sh
make deps
goreman start
open http://127.0.0.1:3200/surveyor/[YOUR QUERY]
```

## With docker

### Build
```sh
docker-compose build
```

### Start the surveyor
```sh
docker-compose up -d search
```

### Start the respondents
```sh
docker-compose up websearch image video
```

