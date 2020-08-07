package external

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// NullInt64 is an alias for sql.NullInt64 data type
type NullInt64 struct {
	sql.NullInt64
}

func NewNullInt64(i int64) *NullInt64 {
	return &NullInt64{
		sql.NullInt64{
			Int64: i,
			Valid: true,
		},
	}
}

// MarshalJSON for NullInt64
func (ni *NullInt64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return nil, nil
	}
	return json.Marshal(ni.Int64)
}

// UnmarshalJSON for NullInt64
func (ni *NullInt64) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &ni.Int64)
	ni.Valid = (err == nil)
	return err
}

// NullBool is an alias for sql.NullBool data type
type NullBool struct {
	sql.NullBool
}

func NewNullBool(b bool) *NullBool {
	return &NullBool{
		sql.NullBool{
			Bool:  b,
			Valid: true,
		},
	}
}

// MarshalJSON for NullBool
func (nb *NullBool) MarshalJSON() ([]byte, error) {
	if !nb.Valid {
		return nil, nil
	}
	return json.Marshal(nb.Bool)
}

// UnmarshalJSON for NullBool
func (nb *NullBool) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &nb.Bool)
	nb.Valid = (err == nil)
	return err
}

// NullString is an alias for sql.NullString data type
type NullString struct {
	sql.NullString
}

func NewNullString(s string) *NullString {
	return &NullString{
		sql.NullString{
			String: s,
			Valid:  true,
		},
	}
}

// MarshalJSON for NullString
func (ns *NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return nil, nil
	}
	return json.Marshal(ns.String)
}

// UnmarshalJSON for NullString
func (ns *NullString) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &ns.String)
	ns.Valid = (err == nil)
	return err
}

// NullTime is an alias for mysql.NullTime data type
type NullTime struct {
	pq.NullTime
}

// MarshalJSON for NullTime
func (nt *NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return nil, nil
	}
	val := fmt.Sprintf("\"%s\"", nt.Time.Format(time.RFC3339))
	return []byte(val), nil
}

// UnmarshalJSON for NullTime
func (nt *NullTime) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &nt.Time)
	nt.Valid = (err == nil)
	return err
}

type NullDecimal struct {
	decimal.NullDecimal
}

// MarshalJSON implements the json.Marshaler interface.
func (d NullDecimal) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return nil, nil
	}
	return d.Decimal.MarshalJSON()
}
