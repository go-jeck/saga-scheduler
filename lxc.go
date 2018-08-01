package main

import (
	"github.com/jmoiron/sqlx"
)

type lxc struct {
	ID          string `db:"id" json:"id"`
	LxdID       string `db:"lxd_id" json:"lxd_id"`
	Name        string `db:"name" json:"name"`
	Type        string `db:"type" json:"type"`
	Alias       string `db:"alias" json:"alias"`
	Address     string `db:"address" json:"address"`
	Description string `db:"description" json:"description"`
	IsDeployed  int    `db:"is_deployed" json:"is_deployed"`
}

func (l *lxc) getLxc(db *sqlx.DB) error {
	rows, err := db.Queryx("SELECT id, lxd_id, name, type, alias, address, description, is_deployed FROM lxc WHERE id=$1 LIMIT 1", l.ID)
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

func getPendingLxcs(db *sqlx.DB) ([]lxc, error) {
	var result []lxc
	rows, err := db.Queryx("SELECT id, lxd_id, name, type, alias, is_deployed FROM lxc WHERE is_deployed = 0")
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		var temp lxc
		err = rows.StructScan(&temp)
		if err != nil {
			return nil, err
		}
		result = append(result, temp)
	}
	return result, nil
}

func (l *lxc) insertLxc(db *sqlx.DB) error {
	_, err := db.NamedExec("INSERT INTO lxc (id, lxd_id, name, type, alias, address, description, is_deployed) VALUES (:id, :lxd_id, :name, :type, :alias, :address, :description, :is_deployed)", l)
	if err != nil {
		return err
	}

	return nil
}
