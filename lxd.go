package main

import (
	"github.com/jmoiron/sqlx"
)

type lxd struct {
	ID          string `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Address     string `db:"address" json:"address"`
	Description string `db:"description" json:"description"`
}

func (l *lxd) getLxd(db *sqlx.DB) error {
	rows, err := db.Queryx("SELECT * FROM lxd WHERE id=$1 LIMIT 1", l.ID)
	if err != nil {
		return err
	}

	if rows.Next() {
		err = rows.StructScan(&l)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *lxd) insertLxd(db *sqlx.DB) error {
	_, err := db.NamedExec("INSERT INTO lxd (id, name, address, description) VALUES (:id, :name, :address, :description)", l)
	if err != nil {
		return err
	}

	return nil
}