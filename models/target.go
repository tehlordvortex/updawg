package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tehlordvortex/updawg/config"
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

func (t *Target) Pk() int64            { return t.pk }
func (t *Target) Id() string           { return t.id }
func (t *Target) CreatedAt() time.Time { return t.createdAt }
func (t *Target) UpdatedAt() time.Time { return t.updatedAt }
func (t *Target) DisplayName() string {
	if t.Name == "" {
		return t.Uri
	} else {
		return t.Name
	}
}

// Target impl PassiveRecord

func (t *Target) Load(Scan PassiveRecordScanFunc) error {
	return loadTarget(t, Scan)
}

func (t *Target) Reload(QueryRow PassiveRecordQueryRowFunc) error {
	if t.pk == -1 {
		return ErrRecordDeleted
	} else if t.pk == 0 && t.id == "" {
		return ErrRecordNotPersisted
	}

	row := QueryRow("SELECT * FROM targets WHERE pk = ?", t.pk)

	return t.Load(func(cols []interface{}) error {
		return row.Scan(cols...)
	})
}

func (t *Target) Save(Exec PassiveRecordExecFunc) error {
	unix := time.Now().UTC().Unix()

	if t.Uri == "" {
		return fmt.Errorf("target must have a uri")
	}

	if t.Period == 0 {
		t.Period = config.DefaultPeriod
	} else if t.Period < 0 {
		return fmt.Errorf("period cannot be negative")
	}

	if t.Method == "" {
		t.Method = config.DefaultMethod
	}

	if t.pk == 0 && t.id == "" {
		id := GenUlid("target")

		result, err := Exec("INSERT INTO targets (id, name, uri, period, created_at, updated_at, method) VALUES (?, ?, ?, ?, ?, ?, ?)", id, t.Name, t.Uri, t.Period, unix, unix, t.Method)
		if err != nil {
			return fmt.Errorf("target.Save: %v", err)
		}

		pk, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("target.Save: %v", err)
		}

		t.pk = pk
		t.id = id
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

func (t *Target) Delete(Exec PassiveRecordExecFunc) error {
	if t.pk == -1 {
		return ErrRecordDeleted
	}

	_, err := Exec("DELETE FROM targets WHERE pk = ?", t.pk)
	if err != nil {
		return err
	}

	t.pk = -1
	return nil
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

func loadTarget(t *Target, Scan PassiveRecordScanFunc) error {
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

func QueryAllActiveTargets(Query PassiveRecordQueryFunc) (*sql.Rows, error) {
	return Query("SELECT * FROM targets")
}

func LoadAllActiveTargets(Query PassiveRecordQueryFunc) ([]Target, error) {
	rows, err := QueryAllActiveTargets(Query)
	if err != nil {
		return nil, err
	}

	return LoadTargets(rows)
}
