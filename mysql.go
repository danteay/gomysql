package gomysql

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	circuit "github.com/rubyist/circuitbreaker"
)

// Regenerate status for circuitbreaker
var Regenerate = "regenerate"

// Success status for circuitbreaker
var Success = "success"

// Fail status for circuitbreaker
var Fail = "fail"

// MysqlOptions conection options structre
type MysqlOptions struct {
	Url        string
	Host       string
	User       string
	Pass       string
	Dbas       string
	Poolsize   int64
	FailRate   float64
	Universe   int64
	TimeOut    time.Duration
	Regenerate time.Duration
}

// MysqlPool pool structure
type MysqlPool struct {
	cb         *circuit.Breaker
	conn       chan *sql.DB
	state      string
	trippedAt  int64
	failCount  int64
	regenTryes int64
	Configs    MysqlOptions
}

// InitPool sets all connections to pool
func InitPool(opts MysqlOptions) (*MysqlPool, error) {
	pool := new(MysqlPool)
	pool.Configs = opts

	configValidate(&opts)
	pool.cb = circuit.NewRateBreaker(pool.Configs.FailRate, pool.Configs.Universe)
	pool.subscribe()

	err := generatePool(pool, false)
	if err != nil {
		pool.state = Fail
	}

	return pool, err
}

// Execute wrapper for manage failures in circuit breaker
func (pool *MysqlPool) Execute(callback func(*sql.DB) error) error {
	if pool.State() == Fail {
		pool.regenerate()
		return errors.New("unavailable service")
	}

	if pool.State() == Regenerate {
		return errors.New("unavailable service")
	}

	conn := pool.popConx()
	if conn == nil {
		pool.cb.Fail()
		return errors.New("empty connection")
	}

	var err error

	pool.cb.Call(func() error {
		err = callback(conn)
		return nil
	}, pool.Configs.TimeOut)
	pool.pushConx(conn)

	return err
}

// Subscribe events for control reset
func (pool *MysqlPool) subscribe() {
	events := pool.cb.Subscribe()

	go func() {
		for {
			e := <-events
			switch e {
			case circuit.BreakerTripped:
				pool.state = Fail
			case circuit.BreakerReset:
				pool.state = Regenerate
			case circuit.BreakerFail:
				log.Println(":::::: breaker fail ::::::")
			case circuit.BreakerReady:
				pool.state = Success
			}
		}
	}()
}

// PopConx return a conection of the pool
func (pool *MysqlPool) popConx() *sql.DB {
	return <-pool.conn
}

// PushConx restore a conection into the pool
func (pool *MysqlPool) pushConx(conx *sql.DB) {
	pool.conn <- conx
}

func configValidate(options *MysqlOptions) {
	var urlConnect string
	if options.Host != "" {
		urlConnect = "user=" + options.User + " dbname=" + options.Dbas + " host=" + options.Host + " password=" + options.Pass
	}
	if options.Url != "" && urlConnect == "" {
		urlConnect = options.Url
	}
	if urlConnect == "" {
		urlConnect = os.Getenv("DATABASE_URL")
	}
	options.Url = urlConnect

	if options.FailRate < 0.0 {
		options.FailRate = 0.0
	}
	if options.FailRate > 1.0 {
		options.FailRate = 1.0
	}
	if options.Poolsize <= 0 {
		options.Poolsize = 5
	}
	if options.Universe <= options.Poolsize {
		options.Universe = options.Poolsize
	}
	if options.TimeOut < 0 {
		options.TimeOut = 0
	}
	if options.Regenerate <= 0 {
		options.Regenerate = time.Second * 3
	}
}

func (pool *MysqlPool) connect() (*sql.DB, error) {
	var err error
	var db *sql.DB

	if pool.Configs.Url == "" {
		return nil, errors.New("can't find url")
	}
	if pool.cb.Tripped() {
		return nil, errors.New("unavailable service")
	}

	err = pool.cb.Call(func() error {
		log.Println(pool.Configs.Url)

		conn, err := sql.Open("mysql", pool.Configs.Url)
		if err != nil {
			return err
		}

		errp := conn.Ping()
		if errp != nil {
			return errp
		}

		db = conn
		return nil
	}, pool.Configs.TimeOut)

	return db, err
}

func generatePool(pool *MysqlPool, failFirst bool) error {
	pool.conn = make(chan *sql.DB, pool.Configs.Poolsize)

	for x := int64(0); x < pool.Configs.Poolsize; x++ {
		var conn *sql.DB
		var err error

		if conn, err = pool.connect(); err != nil {
			if failFirst {
				pool.setTrippedTime()
				return err
			}

			pool.failCount++
		}
		pool.conn <- conn
	}

	if pool.cb.Tripped() {
		pool.setTrippedTime()
		return errors.New("failed to create connection pool")
	}

	pool.state = Success
	return nil
}

func (pool *MysqlPool) regenerate() {
	epoch := time.Now().Unix()
	diference := epoch - pool.trippedAt
	regentime := int64(pool.Configs.Regenerate / 1000000000)

	if diference >= regentime && pool.State() == Fail {
		pool.regenTryes++
		pool.reset()

		if err := generatePool(pool, true); err != nil {
			pool.cb.Trip()
			pool.setTrippedTime()
		} else {
			pool.regenTryes = 0
		}
	}
}

func (pool *MysqlPool) reset() {
	pool.clean()
	pool.cb.Reset()
	pool.failCount = 0
	pool.trippedAt = 0
}

func (pool *MysqlPool) clean() {
	if pool.regenTryes == 0 {
		for x := int64(0); x < pool.Configs.Poolsize; x++ {
			if aux := <-pool.conn; aux != nil {
				aux.Close()
			}
		}
	}
	close(pool.conn)
}

func (pool *MysqlPool) setTrippedTime() {
	if pool.trippedAt == 0 {
		trip := time.Now().Unix()
		pool.trippedAt = trip
	}
}

// GetUrl connection url
func (pool *MysqlPool) GetUrl() string {
	return pool.Configs.Url
}

// State returns actual state of the pool
func (pool *MysqlPool) State() string {
	return pool.state
}
