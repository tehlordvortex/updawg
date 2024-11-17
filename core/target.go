package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tehlordvortex/updawg/config"
	"github.com/tehlordvortex/updawg/database"
)

const (
	TargetTableName = "targets"
)

type Target struct {
	pk        int64
	id        string
	Name      string
	Uri       string
	Method    string
	Period    int64
	config    map[string]interface{}
	createdAt time.Time
	updatedAt time.Time
}

func NewTarget(name, uri string, period int64) Target {
	return Target{
		pk:     -1,
		id:     "",
		Name:   name,
		Uri:    uri,
		Period: period,
		Method: config.DefaultMethod,
	}
}

func LoadTarget(row *sql.Row) (Target, error) {
	var t Target

	err := t.Load(func(cols []interface{}) error {
		return row.Scan(cols...)
	})
	if err != nil {
		return Target{}, fmt.Errorf("LoadTarget: %v", err)
	}

	return t, nil
}

func LoadTargets(rows *sql.Rows) ([]Target, error) {
	var targets []Target

	for rows.Next() {
		var t Target

		err := t.Load(func(cols []interface{}) error {
			return rows.Scan(cols...)
		})
		if err != nil {
			return nil, fmt.Errorf("LoadTargets: %v", err)
		}

		targets = append(targets, t)
	}

	return targets, nil
}

func (t *Target) Pk() int64            { return t.pk }
func (t *Target) Id() string           { return t.id }
func (t *Target) CreatedAt() time.Time { return t.createdAt }
func (t *Target) UpdatedAt() time.Time { return t.updatedAt }

// Target impl PassiveRecord

func (t *Target) Load(Scan database.PassiveRecordScanFunc) error {
	return loadTarget(t, Scan)
}

func (t *Target) Reload(QueryRow database.PassiveRecordQueryRowFunc) error {
	if t.pk == -1 {
		return database.ErrRecordNotPersisted
	}

	row := QueryRow("SELECT * FROM targets WHERE pk = ?", t.pk)

	return t.Load(func(cols []interface{}) error {
		return row.Scan(cols...)
	})
}

func (t *Target) Save(Exec database.PassiveRecordExecFunc) error {
	ts := time.Now().UTC()
	unix := ts.Unix()

	if t.pk == -1 {
		t.id = database.GenUlid("target")

		result, err := Exec("INSERT INTO targets (id, name, uri, period, created_at, updated_at, method) VALUES (?, ?, ?, ?, ?, ?, ?)", t.id, t.Name, t.Uri, t.Period, unix, unix, t.Method)
		if err != nil {
			return fmt.Errorf("target.Save(%s): %v", t.id, err)
		}

		pk, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("target.Save(%s): %v", t.id, err)
		}

		t.pk = pk
		t.createdAt = time.Unix(unix, 0)
		t.updatedAt = time.Unix(unix, 0)
		return nil
	}

	_, err := Exec("UPDATE targets SET (name, uri, period, updated_at, method) = (?, ?, ?, ?, ?) WHERE pk = ?", t.Name, t.Uri, t.Period, unix, t.Method, t.pk)
	if err != nil {
		return fmt.Errorf("target.Save(%s): %v", t.id, err)
	}

	t.updatedAt = time.Unix(unix, 0)
	return nil
}

func loadTarget(t *Target, Scan database.PassiveRecordScanFunc) error {
	var nameNullable, methodNullable sql.NullString
	var configJson string
	var createdAtUnix, updatedAtUnix int64

	cols := []interface{}{&t.pk, &t.id, &nameNullable, &t.Uri, &t.Period, &configJson, &createdAtUnix, &updatedAtUnix, &methodNullable}
	err := Scan(cols)
	if err != nil {
		return err
	}

	if nameNullable.Valid {
		t.Name = nameNullable.String
	}

	if methodNullable.Valid {
		t.Method = methodNullable.String
	} else {
		t.Method = config.DefaultMethod
	}

	err = json.Unmarshal([]byte(configJson), &t.config)
	if err != nil {
		return err
	}

	t.createdAt = time.Unix(createdAtUnix, 0)
	t.updatedAt = time.Unix(updatedAtUnix, 0)

	return nil
}
