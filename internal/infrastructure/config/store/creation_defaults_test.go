package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"mcgame-server/internal/domain/character"
	"testing"
)

type defaultsTestReader struct {
	raw   []byte
	err   error
	calls int
}
type defaultsTestRow struct {
	raw []byte
	err error
}

func (r *defaultsTestReader) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	r.calls++
	if sql != "select value from public.server_settings where key=$1" || len(args) != 1 || args[0] != character.NewCharacterDefaultsKey {
		panic("unexpected defaults query")
	}
	return defaultsTestRow{r.raw, r.err}
}
func (r defaultsTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*[]byte) = append([]byte{}, r.raw...)
	return nil
}

func TestCreationDefaultsFreshReadAndFailClosed(t *testing.T) {
	r := &defaultsTestReader{err: pgx.ErrNoRows}
	d, err := loadNewCharacterDefaults(context.Background(), r)
	if err != nil || d.Level != 90 || d.Gold != 999999 || d.GoldBind != 999999 {
		t.Fatal(d, err)
	}
	r.err = errors.New("database unavailable")
	if _, err := loadNewCharacterDefaults(context.Background(), r); !errors.Is(err, r.err) {
		t.Fatal("database error replaced by fallback", err)
	}
	r.err = nil
	r.raw = []byte(`{"version":1,"level":2,"gold":3,"goldBind":4,"items":[],"pets":[]}`)
	d, err = loadNewCharacterDefaults(context.Background(), r)
	if err != nil || d.Level != 2 || d.Gold != 3 {
		t.Fatal(d, err)
	}
	r.raw = []byte(`{"version":1,"level":3,"gold":4,"goldBind":5,"items":[],"pets":[]}`)
	d, err = loadNewCharacterDefaults(context.Background(), r)
	if err != nil || d.Level != 3 || d.Gold != 4 || r.calls != 4 {
		t.Fatal("stale snapshot", d, err)
	}
	r.raw = []byte(`{"level":90}`)
	if _, err := loadNewCharacterDefaults(context.Background(), r); err == nil {
		t.Fatal("invalid durable config replaced by defaults")
	}
}
