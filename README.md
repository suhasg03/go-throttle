# Go rate limiter

This package provides a Golang implementation of the leaky-bucket, sliding window and fixed window rate limit algorithm.
Create GoThrottle object by passing your redis client and rate limit strategy and call Allow() by passing the key and the limit and the window size.

### Installation

```bash
go get github.com/suhasg03/go-throttle
```

```go
import (
    "github.com/suhasg03/go-throttle"
)

func main() {
    throttle := NewGoThrottle(rdb, FixedWindow)
    allowed, err := throttle.Allow(ctx, key, limit, window)
}
```
