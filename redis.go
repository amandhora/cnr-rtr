package main

import (
	"github.com/garyburd/redigo/redis"
	"log"
)

func setCacheEntry(r redis.Conn, app string, value []byte) error {

	_, err := r.Do("SET", app, value, "EX", 10)
	if err != nil {
		log.Print(err)
	}

	return err
}

func getCacheEntry(r redis.Conn, app string) ([]byte, error) {

	data, err := redis.Bytes(r.Do("GET", app))
	if err != nil {
		log.Print(err)
	}

	return data, err
}

func saveTransInCache(r redis.Conn, transId string) error {

	_, err := r.Do("SET", transId, 8, "EX", 8)
	if err != nil {
		log.Print(err)
	}

	return err
}

func getTransFromCache(r redis.Conn, transId string) (int, error) {

	data, err := redis.Int(r.Do("GET", transId))
	if err != nil {
		log.Print(err)
	}

	return data, err
}
