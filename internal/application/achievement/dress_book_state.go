// Open-sourced by BaoLT

// Dress-book state reader proves activated fashion and flyer collection milestones.
package achievement

import (
	"context"
	"fmt"
	"strconv"

	domaindress "mcgame-server/internal/domain/dress"
	"mcgame-server/internal/gamedata/models"
)

type DressBookStateCatalog interface {
	GetAllDresses() []*models.DressTemplate
}

type dressBookIdentity struct {
	dressID     int64
	dressType   int
	equipmentID int
}

var dressBookIdentityRequirements = map[int64]dressBookIdentity{
	617: {1002, 1, 1962},
	618: {1018, 1, 1978},
	619: {1019, 1, 1979},
	620: {1020, 1, 1980},
	621: {1021, 1, 1981},
	622: {1022, 1, 1982},
	623: {1023, 1, 1983},
	624: {1007, 1, 1967},
	625: {1008, 1, 1968},
	626: {1009, 1, 1969},
	627: {1010, 1, 1970},
	628: {1011, 1, 1971},
	629: {1017, 1, 1977},
	630: {2011, 2, 1994},
	631: {2012, 2, 1995},
	632: {2013, 2, 1996},
	633: {2014, 2, 1997},
	634: {2009, 2, 1992},
	635: {2010, 2, 1993},
}

type DressBookStateReader struct {
	repo    CharacterStateRepository
	catalog DressBookStateCatalog
}

func NewDressBookStateReader(repo CharacterStateRepository, catalog DressBookStateCatalog) *DressBookStateReader {
	return &DressBookStateReader{repo: repo, catalog: catalog}
}

func (r *DressBookStateReader) Name() string {
	return "dress-book"
}

func (r *DressBookStateReader) RequirementIDs() []int64 {
	ids := requirementIDRange(611, 635)
	return append(ids, 712, 713, 714, 761, 762, 763, 764)
}

func (r *DressBookStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil {
		return nil, fmt.Errorf("dress-book state dependencies not configured")
	}
	character, err := r.repo.FindByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if character == nil || character.ID != characterID {
		return nil, fmt.Errorf("dress-book character %d not found", characterID)
	}
	info, err := domaindress.Decode(character.DressInfo)
	if err != nil {
		return nil, fmt.Errorf("decode dress-book state: %w", err)
	}
	catalogByID, err := validatedDressBookCatalog(r.catalog.GetAllDresses())
	if err != nil {
		return nil, err
	}
	active := make(map[int64]struct{})
	fashionCount := 0
	flyerCount := 0
	for rawID, state := range info.Book {
		if state < 1 {
			continue
		}
		dressID, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil || dressID <= 0 {
			return nil, fmt.Errorf("dress-book has invalid activated id %q", rawID)
		}
		template := catalogByID[dressID]
		if template == nil {
			return nil, fmt.Errorf("dress-book has unknown activated id %d", dressID)
		}
		dressType, _ := exactPetCatalogInt(template.Type)
		active[dressID] = struct{}{}
		if dressType == 1 {
			fashionCount++
		} else {
			flyerCount++
		}
	}
	proofs := []StateProof{
		stateProof(611, fashionCount, 1),
		stateProof(612, fashionCount, 10),
		stateProof(613, fashionCount, 20),
		stateProof(614, flyerCount, 1),
		stateProof(615, flyerCount, 5),
		stateProof(616, flyerCount, 10),
	}
	for requirementID := int64(617); requirementID <= 635; requirementID++ {
		identity := dressBookIdentityRequirements[requirementID]
		_, exists := active[identity.dressID]
		progress := 0
		if exists {
			progress = 1
		}
		proofs = append(proofs, stateProof(requirementID, progress, 1))
	}
	proofs = append(proofs,
		stateProof(712, fashionCount, 30),
		stateProof(713, fashionCount, 40),
		stateProof(714, flyerCount, 15),
		stateProof(761, fashionCount, 45),
		stateProof(762, fashionCount, 50),
		stateProof(763, fashionCount, 55),
		stateProof(764, flyerCount, 20),
	)
	return proofs, nil
}

func validatedDressBookCatalog(entries []*models.DressTemplate) (map[int64]*models.DressTemplate, error) {
	byID := make(map[int64]*models.DressTemplate, len(entries))
	fashionCount := 0
	flyerCount := 0
	for _, entry := range entries {
		if entry == nil || entry.ID <= 0 {
			return nil, fmt.Errorf("dress catalog contains invalid entry")
		}
		if _, duplicate := byID[entry.ID]; duplicate {
			return nil, fmt.Errorf("dress catalog repeats entry %d", entry.ID)
		}
		dressType, ok := exactPetCatalogInt(entry.Type)
		if !ok || (dressType != 1 && dressType != 2) {
			return nil, fmt.Errorf("dress catalog entry %d has invalid type", entry.ID)
		}
		byID[entry.ID] = entry
		if dressType == 1 {
			fashionCount++
		} else {
			flyerCount++
		}
	}
	if fashionCount < 55 || flyerCount < 20 {
		return nil, fmt.Errorf("dress catalog cannot prove achievement count targets")
	}
	for requirementID, identity := range dressBookIdentityRequirements {
		entry := byID[identity.dressID]
		if entry == nil {
			return nil, fmt.Errorf("dress requirement %d identity missing", requirementID)
		}
		dressType, typeOK := exactPetCatalogInt(entry.Type)
		equipmentID, equipmentOK := exactPetCatalogInt(entry.EquiptID)
		if !typeOK || !equipmentOK || dressType != identity.dressType || equipmentID != identity.equipmentID {
			return nil, fmt.Errorf("dress requirement %d identity drift", requirementID)
		}
	}
	return byID, nil
}
