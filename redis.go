package main

import (
	"github.com/garyburd/redigo/redis"
	"log"
)

func setCacheEntry(c redis.Conn, app string, value []byte) error {

	_, err := c.Do("SET", app, value, "EX", 10)
	if err != nil {
		log.Print(err)
	}

	return err
}

func getCacheEntry(c redis.Conn, app string) ([]byte, error) {

	//data, err := redis.String(c.Do("GET", app))
	data, err := redis.Bytes(c.Do("GET", app))

	//fmt.Println("REDIS: ", app)

	return data, err
}
