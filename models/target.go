package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tehlordvortex/updawg/config"
)

const (
	TargetModelTableName = "targets"
	TargetCreatedTopic   = "target.created"
	TargetDeletedTopic   = "target.deleted"
	TargetUpdatedTopic   = "target.updated"
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

func (t *Target) Reload(ctx context.Context, qe QueryExecutor) error {
	if t.pk == -1 {
		return ErrRecordDeleted
	} else if t.pk == 0 && t.id == "" {
		return ErrRecordNotPersisted
	}

	row := qe.QueryRowContext(ctx, "SELECT * FROM targets WHERE pk = ?", t.pk)

	return t.Load(func(cols []interface{}) error {
		return row.Scan(cols...)
	})
}

func (t *Target) Save(ctx context.Context, qe QueryExecutor) error {
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

		result, err := qe.ExecContext(ctx, "INSERT INTO targets (id, name, uri, period, created_at, updated_at, method) VALUES (?, ?, ?, ?, ?, ?, ?)", id, t.Name, t.Uri, t.Period, unix, unix, t.Method)
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

	_, err := qe.ExecContext(ctx, "UPDATE targets SET (name, uri, period, updated_at, method) = (?, ?, ?, ?, ?) WHERE pk = ?", t.Name, t.Uri, t.Period, unix, t.Method, t.pk)
	if err != nil {
		return fmt.Errorf("target.Save(%s): %v", t.id, err)
	}

	t.updatedAt = time.Unix(unix, 0)

	return nil
}

func (t *Target) Delete(ctx context.Context, qe QueryExecutor) error {
	if t.pk == -1 {
		return ErrRecordDeleted
	}

	_, err := qe.ExecContext(ctx, "DELETE FROM targets WHERE pk = ?", t.pk)
	if err != nil {
		return err
	}

	t.pk = -1

	return nil
}

func FindAllTargets(ctx context.Context, qe QueryExecutor) ([]*Target, error) {
	rows, err := qe.QueryContext(ctx, "SELECT * FROM targets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return LoadRecords[*Target](rows)
}

func FindAllActiveTargets(ctx context.Context, qe QueryExecutor) ([]*Target, error) {
	// TODO: Disable/enable targets
	return FindAllTargets(ctx, qe)
}

func FindTargetById(ctx context.Context, qe QueryExecutor, id string) (*Target, error) {
	return LoadRecord[*Target](qe.QueryRowContext(ctx, "SELECT * FROM targets WHERE id = ?", id))
}

func FindTargetsByIdPrefix(ctx context.Context, qe QueryExecutor, prefix string) ([]*Target, error) {
	rows, err := qe.QueryContext(ctx, "SELECT * FROM targets WHERE id LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return LoadRecords[*Target](rows)
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
