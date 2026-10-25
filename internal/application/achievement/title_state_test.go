// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"
)

type titleStateRepositoryFixture struct {
	titles        []int
	specialTitles []int
}

func (r titleStateRepositoryFixture) GetCharacterTitles(ctx context.Context, characterID int64) ([]int, error) {
	return r.titles, nil
}

func (r titleStateRepositoryFixture) GetCharacterSpecialTitles(ctx context.Context, characterID int64) ([]int, error) {
	return r.specialTitles, nil
}

func TestTitleStateReaderCountsDistinctPositiveTitles(t *testing.T) {
	repo := titleStateRepositoryFixture{
		titles:        []int{1, 2, 3, 4, 5, 0},
		specialTitles: []int{5, 6, 7, 8, 9, 10},
	}
	proofs, err := NewTitleStateReader(repo).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 3 || !proofs[0].Done || !proofs[1].Done || proofs[2].Done || proofs[2].Progress != 10 {
		t.Fatalf("proofs = %#v", proofs)
	}
}

var _ TitleStateRepository = titleStateRepositoryFixture{}
