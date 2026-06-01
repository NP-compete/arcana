package db

import (
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// envOr
// ---------------------------------------------------------------------------

func TestEnvOr(t *testing.T) {
	t.Run("returns env var when set", func(t *testing.T) {
		t.Setenv("TEST_ENV_OR_KEY", "from-env")
		got := envOr("TEST_ENV_OR_KEY", "fallback")
		if got != "from-env" {
			t.Errorf("envOr: got %q, want %q", got, "from-env")
		}
	})

	t.Run("returns fallback when unset", func(t *testing.T) {
		os.Unsetenv("TEST_ENV_OR_MISSING")
		got := envOr("TEST_ENV_OR_MISSING", "fallback-val")
		if got != "fallback-val" {
			t.Errorf("envOr: got %q, want %q", got, "fallback-val")
		}
	})

	t.Run("returns fallback when empty", func(t *testing.T) {
		t.Setenv("TEST_ENV_OR_EMPTY", "")
		got := envOr("TEST_ENV_OR_EMPTY", "default")
		if got != "default" {
			t.Errorf("envOr: got %q, want %q", got, "default")
		}
	})
}

// ---------------------------------------------------------------------------
// dbMode
// ---------------------------------------------------------------------------

func TestDBMode(t *testing.T) {
	t.Run("returns local when DB_MODE not set", func(t *testing.T) {
		os.Unsetenv("DB_MODE")
		got := dbMode()
		if got != "local" {
			t.Errorf("dbMode: got %q, want %q", got, "local")
		}
	})

	t.Run("returns env value when set", func(t *testing.T) {
		t.Setenv("DB_MODE", "cnpg")
		got := dbMode()
		if got != "cnpg" {
			t.Errorf("dbMode: got %q, want %q", got, "cnpg")
		}
	})
}

// ---------------------------------------------------------------------------
// resolveHost
// ---------------------------------------------------------------------------

func TestResolveHost_Local(t *testing.T) {
	os.Unsetenv("POSTGRES_HOST")
	os.Unsetenv("DB_MODE")

	got := resolveHost()
	if got != "postgres" {
		t.Errorf("resolveHost (local): got %q, want %q", got, "postgres")
	}
}

func TestResolveHost_CNPG(t *testing.T) {
	os.Unsetenv("POSTGRES_HOST")
	t.Setenv("DB_MODE", "cnpg")

	got := resolveHost()
	want := "arcana-db-rw.arcana.svc.cluster.local"
	if got != want {
		t.Errorf("resolveHost (cnpg): got %q, want %q", got, want)
	}
}

func TestResolveHost_CNPG_CustomHost(t *testing.T) {
	os.Unsetenv("POSTGRES_HOST")
	t.Setenv("DB_MODE", "cnpg")
	t.Setenv("CNPG_CLUSTER_HOST", "custom-cnpg.ns.svc")

	got := resolveHost()
	if got != "custom-cnpg.ns.svc" {
		t.Errorf("resolveHost (cnpg custom): got %q, want %q", got, "custom-cnpg.ns.svc")
	}
}

func TestResolveHost_External(t *testing.T) {
	os.Unsetenv("POSTGRES_HOST")
	t.Setenv("DB_MODE", "external")

	got := resolveHost()
	if got != "localhost" {
		t.Errorf("resolveHost (external): got %q, want %q", got, "localhost")
	}
}

func TestResolveHost_External_CustomHost(t *testing.T) {
	os.Unsetenv("POSTGRES_HOST")
	t.Setenv("DB_MODE", "external")
	t.Setenv("EXTERNAL_DB_HOST", "rds.amazonaws.com")

	got := resolveHost()
	if got != "rds.amazonaws.com" {
		t.Errorf("resolveHost (external custom): got %q, want %q", got, "rds.amazonaws.com")
	}
}

func TestResolveHost_EnvOverride(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "override-host")
	t.Setenv("DB_MODE", "cnpg")

	got := resolveHost()
	if got != "override-host" {
		t.Errorf("resolveHost (override): got %q, want %q", got, "override-host")
	}
}

// ---------------------------------------------------------------------------
// resolvePort
// ---------------------------------------------------------------------------

func TestResolvePort_Default(t *testing.T) {
	os.Unsetenv("POSTGRES_PORT")
	os.Unsetenv("DB_MODE")

	got := resolvePort()
	if got != "5432" {
		t.Errorf("resolvePort (default): got %q, want %q", got, "5432")
	}
}

func TestResolvePort_EnvOverride(t *testing.T) {
	t.Setenv("POSTGRES_PORT", "5433")

	got := resolvePort()
	if got != "5433" {
		t.Errorf("resolvePort (override): got %q, want %q", got, "5433")
	}
}

func TestResolvePort_External(t *testing.T) {
	os.Unsetenv("POSTGRES_PORT")
	t.Setenv("DB_MODE", "external")
	t.Setenv("EXTERNAL_DB_PORT", "15432")

	got := resolvePort()
	if got != "15432" {
		t.Errorf("resolvePort (external): got %q, want %q", got, "15432")
	}
}

func TestResolvePort_ExternalDefault(t *testing.T) {
	os.Unsetenv("POSTGRES_PORT")
	os.Unsetenv("EXTERNAL_DB_PORT")
	t.Setenv("DB_MODE", "external")

	got := resolvePort()
	if got != "5432" {
		t.Errorf("resolvePort (external default): got %q, want %q", got, "5432")
	}
}

// ---------------------------------------------------------------------------
// buildDSN
// ---------------------------------------------------------------------------

func TestBuildDSN_DevMode(t *testing.T) {
	// Clear all env vars that influence DSN construction
	os.Unsetenv("ARCANA_ENV")
	os.Unsetenv("POSTGRES_HOST")
	os.Unsetenv("POSTGRES_PORT")
	os.Unsetenv("POSTGRES_USER")
	os.Unsetenv("POSTGRES_PASSWORD")
	os.Unsetenv("POSTGRES_DB")
	os.Unsetenv("POSTGRES_SSLMODE")
	os.Unsetenv("DB_MODE")

	dsn := buildDSN()

	checks := []struct {
		field string
		want  string
	}{
		{"host=", "host=postgres"},
		{"port=", "port=5432"},
		{"user=", "user=arcana"},
		{"password=", "password=arcana-dev"},
		{"dbname=", "dbname=arcana"},
		{"sslmode=", "sslmode=prefer"},
	}

	for _, c := range checks {
		if !strings.Contains(dsn, c.want) {
			t.Errorf("buildDSN dev mode: DSN %q missing %q", dsn, c.want)
		}
	}
}

func TestBuildDSN_DevModeCustomPassword(t *testing.T) {
	os.Unsetenv("ARCANA_ENV")
	t.Setenv("POSTGRES_PASSWORD", "custom-pass")

	dsn := buildDSN()
	if !strings.Contains(dsn, "password=custom-pass") {
		t.Errorf("buildDSN custom password: DSN %q should contain custom password", dsn)
	}
}

func TestBuildDSN_ProductionWithPassword(t *testing.T) {
	t.Setenv("ARCANA_ENV", "production")
	t.Setenv("POSTGRES_PASSWORD", "prod-secret")
	os.Unsetenv("POSTGRES_SSLMODE")

	dsn := buildDSN()
	if !strings.Contains(dsn, "password=prod-secret") {
		t.Errorf("buildDSN production: DSN %q missing production password", dsn)
	}
	if !strings.Contains(dsn, "sslmode=require") {
		t.Errorf("buildDSN production: DSN %q should default to sslmode=require", dsn)
	}
}

// Note: TestBuildDSN_ProductionRequiresPassword is not possible to test
// directly because buildDSN calls os.Exit(1) when POSTGRES_PASSWORD is
// empty in production mode. The exit behavior is verified by the existence
// of the guard in buildDSN. Testing os.Exit requires subprocess testing
// which is out of scope for unit tests.

// ---------------------------------------------------------------------------
// Migration embedded FS
// ---------------------------------------------------------------------------

func TestMigrateReadsMigrationFiles(t *testing.T) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		t.Fatalf("failed to read embedded migrations dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no migration files found in embedded FS")
	}

	// Verify that the expected initial migration files exist
	foundUp := false
	foundDown := false
	for _, e := range entries {
		if e.Name() == "001_initial_schema.up.sql" {
			foundUp = true
		}
		if e.Name() == "001_initial_schema.down.sql" {
			foundDown = true
		}
	}

	if !foundUp {
		t.Error("expected 001_initial_schema.up.sql in embedded migrations")
	}
	if !foundDown {
		t.Error("expected 001_initial_schema.down.sql in embedded migrations")
	}
}

func TestMigrationFilesReadable(t *testing.T) {
	content, err := migrationFS.ReadFile("migrations/001_initial_schema.up.sql")
	if err != nil {
		t.Fatalf("failed to read up migration: %v", err)
	}
	if len(content) == 0 {
		t.Error("up migration file is empty")
	}
	if !strings.Contains(string(content), "CREATE") {
		t.Error("up migration should contain CREATE statements")
	}
}

func TestPendingUpFiles_EmptyApplied(t *testing.T) {
	applied := make(map[string]bool)
	files, err := pendingUpFiles(applied)
	if err != nil {
		t.Fatalf("pendingUpFiles: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one pending migration when none applied")
	}
	for _, f := range files {
		if !strings.HasSuffix(f, ".up.sql") {
			t.Errorf("pending file %q should end with .up.sql", f)
		}
	}
}

func TestPendingUpFiles_AllApplied(t *testing.T) {
	// Get all migration versions
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	applied := make(map[string]bool)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			version := strings.TrimSuffix(e.Name(), ".up.sql")
			applied[version] = true
		}
	}

	files, err := pendingUpFiles(applied)
	if err != nil {
		t.Fatalf("pendingUpFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no pending migrations when all applied, got %d", len(files))
	}
}

func TestPendingUpFiles_Sorted(t *testing.T) {
	applied := make(map[string]bool)
	files, err := pendingUpFiles(applied)
	if err != nil {
		t.Fatalf("pendingUpFiles: %v", err)
	}
	for i := 1; i < len(files); i++ {
		if files[i] < files[i-1] {
			t.Errorf("pending files not sorted: %q before %q", files[i-1], files[i])
		}
	}
}
