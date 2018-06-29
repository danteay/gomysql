package test

import (
	"os"
	"testing"
	"time"

	"github.com/danteay/gomysql"
	"github.com/stretchr/testify/assert"
)

func TestConfigStruct(t *testing.T) {
	conf := gomysql.MysqlOptions{
		Url:      os.Getenv("MYSQL_URL"),
		Poolsize: 10,
		FailRate: 0.25,
		Universe: 4,
		TimeOut:  time.Second * 1,
	}

	assert.True(t, conf.Url == os.Getenv("MYSQL_URL"))
	assert.True(t, conf.Poolsize == 10)
	assert.True(t, conf.FailRate == 0.25)
	assert.True(t, conf.Universe == 4)
	assert.True(t, conf.TimeOut == time.Second*1)
}

func TestEnvUrlConnetc(t *testing.T) {
	_, err := gomysql.InitPool(gomysql.MysqlOptions{Url: os.Getenv("MYSQL_URL")})
	assert.Nil(t, err)
}

func TestPoolConection(t *testing.T) {
	conf := gomysql.MysqlOptions{
		Url:      os.Getenv("MYSQL_URL"),
		Poolsize: 5,
		FailRate: 0.25,
		Universe: 4,
		TimeOut:  time.Second * 1,
	}
	_, err := gomysql.InitPool(conf)

	assert.Nil(t, err)
}
