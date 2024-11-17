package database

import (
	ulid "github.com/oklog/ulid/v2"
)

func GenUlid(prefix string) string {
	return prefix + "_" + ulid.Make().String()
}
