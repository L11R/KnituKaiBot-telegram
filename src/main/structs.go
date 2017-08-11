package main

import "time"

type User struct {
	Id       int
	GroupNum string
	GroupID  int
}

type Group struct {
	Id       int
	Schedule [][]map[string]string
	Time     time.Time
}
