package common

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func init() {
	sql.Register("fakepingfail", fakePingFailDriver{})
}

type fakePingFailDriver struct{}

func (fakePingFailDriver) Open(string) (driver.Conn, error) {
	return fakePingFailConn{}, nil
}

type fakePingFailConn struct{}

func (fakePingFailConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (fakePingFailConn) Close() error                         { return nil }
func (fakePingFailConn) Begin() (driver.Tx, error)           { return nil, errors.New("no tx") }

func (fakePingFailConn) Ping(_ context.Context) error {
	return errors.New("connection refused")
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body: got %q, want %q", rec.Body.String(), "ok")
	}
}

func TestReadinessHandlerNilDB(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/readyz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	ReadinessHandler(nil)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body: got %q, want %q", rec.Body.String(), "ok")
	}
}

func TestReadinessHandlerDBPingFailure(t *testing.T) {
	db, err := sql.Open("fakepingfail", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	req, err := http.NewRequest(http.MethodGet, "/readyz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	ReadinessHandler(db)(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !strings.HasPrefix(rec.Body.String(), "db: ") {
		t.Errorf("body: got %q, want prefix %q", rec.Body.String(), "db: ")
	}
}
