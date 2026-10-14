package database

import (
	"testing"
)

func TestGormDialector_ByDriver(t *testing.T) {
	for _, test := range []struct {
		driver string
		want   string
	}{{"sqlserver", "sqlserver"}, {"mysql", "mysql"}} {
		t.Run(test.driver, func(t *testing.T) {
			dialector, err := gormDialector(test.driver, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := dialector.Name(); got != test.want {
				t.Fatalf("expected %q dialector, got %q", test.want, got)
			}
		})
	}
}

func TestGormDialector_UnsupportedDriver_ReturnsError(t *testing.T) {
	if _, err := gormDialector("postgres", nil); err == nil {
		t.Fatal("expected unsupported driver error")
	}
}
