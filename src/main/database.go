package main

import (
	"errors"
	r "gopkg.in/gorethink/gorethink.v3"
	"log"
)

func InitConnectionPool() {
	var err error

	session, err = r.Connect(r.ConnectOpts{
		Address:    dbUrl,
		InitialCap: 10,
		MaxOpen:    10,
		Database:   "kai",
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetUser(ID int) (User, error) {
	res, err := r.Table("users").Get(ID).Run(session)
	if err != nil {
		return User{}, err
	}

	var user User
	err = res.One(&user)
	if err == r.ErrEmptyResult {
		return User{}, errors.New("DB: Row not found!")
	}
	if err != nil {
		return User{}, err
	}

	defer res.Close()
	return user, nil
}

func GetGroup(ID int) (Group, error) {
	res, err := r.Table("groups").Get(ID).Run(session)
	if err != nil {
		return Group{}, err
	}

	var group Group
	err = res.One(&group)
	if err == r.ErrEmptyResult {
		return Group{}, errors.New("DB: Row not found!")
	}
	if err != nil {
		return Group{}, err
	}

	defer res.Close()
	return group, nil
}
