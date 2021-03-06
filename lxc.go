package main

import (
	log "github.com/sirupsen/logrus"
)

type lxc struct {
	ID          string  `db:"id" json:"id"`
	LxdID       string  `db:"lxd_id" json:"lxd_id"`
	Name        string  `db:"name" json:"name"`
	Type        string  `db:"type" json:"type"`
	Alias       string  `db:"alias" json:"alias"`
	Protocol    string  `db:"protocol" json:"protocol"`
	Server      string  `db:"server" json:"server"`
	Address     string  `db:"address" json:"address"`
	Status      string  `db:"status" json:"status"`
	Description string  `db:"description" json:"description"`
	WeightScore string  `json:"weight"`
	WeightValue float64 `json:"weightValue"`
}

func (l *lxc) getLxc(db PostgresQL) error {
	rows, err := db.Queryx("SELECT id, lxd_id, name, type, alias, address, description, status FROM lxc WHERE id=$1 LIMIT 1", l.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.StructScan(&l)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *lxc) checkNeedUpdate(curLxc lxc) bool {
	if curLxc.Status == "started" {
		if l.Status == "starting" {
			return false
		} else {
			return true
		}
	} else if curLxc.Status == "stopped" {
		if l.Status == "stopping" {
			return false
		} else {
			return true
		}
	}
	return true
}

func (l *lxc) updateStatusByID(db PostgresQL) error {
	curLxc := lxc{ID: l.ID}
	if err := curLxc.getLxc(db); err != nil {
		return err
	}
	if l.checkNeedUpdate(curLxc) {
		log.Info("Lxc status update needed")
		_, err := db.Exec("UPDATE lxc SET status = $2 WHERE id = $1", l.ID, l.Status)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *lxc) insertLxc(db PostgresQL) error {
	_, err := db.NamedExec("INSERT INTO lxc (id, lxd_id, name, type, alias, protocol, server, address, description, status) VALUES (:id, :lxd_id, :name, :type, :alias, :protocol, :server, :address, :description, :status)", l)
	if err != nil {
		return err
	}

	return nil
}

func (l *lxc) deleteLxc(db PostgresQL) error {
	_, err := db.Queryx("DELETE FROM lxc WHERE id = $1", l.ID)
	if err != nil {
		return err
	}
	return nil
}

func (l *lxc) getLxcListByLxdID(db PostgresQL, lxdID string) ([]lxc, error) {
	rows, err := db.Queryx("SELECT id, lxd_id, name, type, alias, protocol, server, address, description, status FROM lxc WHERE lxd_id=$1", lxdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lxcList := []lxc{}

	for rows.Next() {
		l := lxc{}
		err = rows.StructScan(&l)
		if err != nil {
			return nil, err
		}
		lxcList = append(lxcList, l)
	}

	return lxcList, nil
}

func (l *lxc) checkIfLxcExist(db PostgresQL) bool {
	rows, err := db.Queryx("SELECT id, lxd_id, name, type, alias, protocol, server, address, description, status FROM lxc WHERE name=$1", l.Name)
	if err != nil {
		return false
	}
	defer rows.Close()

	if rows.Next() {
		return true
	}
	return false
}

func (l *lxc) getLxcNameByID(db PostgresQL) string {
	rows, err := db.Queryx("SELECT name FROM lxc WHERE id=$1", l.ID)
	if err != nil {
		log.Error(err.Error())
		return ""
	}
	defer rows.Close()
	lxcData := lxc{}
	if rows.Next() {
		if err = rows.StructScan(&lxcData); err != nil {
			log.Error(err.Error())
			return ""
		}
	}
	log.Infof("lxc name: %s", lxcData.Name)
	return lxcData.Name
}
