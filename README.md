# mysqlcp

## Install

```bash
go get -u -v github.com/danteay/gomysql
```

And import in your files whit the next lines:

```go
import (
  "database/sql"
  "github.com/danteay/gomysql"
)
```

## Configure

Setup config for circut and recover strategies

```go
conf := gomysql.MysqlOptions{
  Url:        "user:password@/dbname",
  Poolsize:   10,
  FailRate:   0.25,
  Regenerate: time.Second * 5,
  TimeOut:    time.Second * 1,
}
```

Init connection pool

```go
pool, err := gomysql.InitPool(conf)

if err != nil {
  fmt.Println(err)
}
```

Execute querys inside of the circuit breaker

```go
var suma int

errQuery := pool.Execute(func(db *sql.DB) error {
  fmt.Println("Entra callback")
  return db.QueryRow("SELECT 1+1 AS suma").Scan(&suma)
})

if errQuery != nil {
  fmt.Println(errQuery)
}
```

Healt check of the pool connection

```go
fmt.Println("==>> State: ", pool.State())
```