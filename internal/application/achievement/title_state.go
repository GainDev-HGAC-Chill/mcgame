// Open-sourced by BaoLT

// Title state reader proves distinct owned-title count milestones.
package achievement

import (
	"context"
	"fmt"

	"mcgame-server/internal/gamedata/models"
)

type TitleStateRepository interface {
	GetCharacterTitles(ctx context.Context, characterID int64) ([]int, error)
	GetCharacterSpecialTitles(ctx context.Context, characterID int64) ([]int, error)
}

type TitleStateCatalog interface {
	GetTitle(id int) *models.TitleTemplate
	GetBuff(id int) *models.BuffTemplate
}

type exactTitleRequirement struct {
	requirementID int64
	titleID       int
	buffID        int
	titleName     string
	buffName      string
	level         int
}

var exactTitleRequirements = []exactTitleRequirement{
	{requirementID: 128, titleID: 125, buffID: 2034, titleName: "Điểm Danh Đầy Đủ", buffName: "Điểm Danh Đầy Đủ", level: 1},
	{requirementID: 163, titleID: 128, buffID: 2037, titleName: "Bậc Thầy Dùng Gai", buffName: "Bậc Thầy Dùng Gai", level: 1},
	{requirementID: 170, titleID: 19, buffID: 665, titleName: "Bất Khuất Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Bất Khuất", level: 1},
	{requirementID: 171, titleID: 20, buffID: 667, titleName: "Bất Khuất Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Bất Khuất", level: 2},
	{requirementID: 172, titleID: 21, buffID: 668, titleName: "Bất Khuất Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Bất Khuất", level: 3},
	{requirementID: 173, titleID: 48, buffID: 1072, titleName: "Bất Khuất Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Bất Khuất", level: 4},
	{requirementID: 174, titleID: 22, buffID: 669, titleName: "Vô Uý Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Vô Úy", level: 1},
	{requirementID: 175, titleID: 23, buffID: 670, titleName: "Vô Uý Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Vô Úy", level: 2},
	{requirementID: 176, titleID: 24, buffID: 671, titleName: "Vô Uý Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Vô Úy", level: 3},
	{requirementID: 177, titleID: 25, buffID: 673, titleName: "Bá Thể Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Bá Thể", level: 1},
	{requirementID: 178, titleID: 26, buffID: 674, titleName: "Bá Thể Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Bá Thể", level: 2},
	{requirementID: 179, titleID: 27, buffID: 675, titleName: "Bá Thể Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Bá Thể", level: 3},
	{requirementID: 180, titleID: 28, buffID: 676, titleName: "Khí Phách Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Khí Phách", level: 1},
	{requirementID: 181, titleID: 29, buffID: 677, titleName: "Khí Phách Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Khí Phách", level: 2},
	{requirementID: 182, titleID: 30, buffID: 678, titleName: "Khí Phách Chi Hồn", buffName: "Dũng Sĩ Chi Hồn - Khí Phách", level: 3},
	{requirementID: 220, titleID: 130, buffID: 2039, titleName: "Chiếm Được Đỉnh Tháp", buffName: "Chiếm Được Đỉnh Tháp", level: 1},
	{requirementID: 327, titleID: 129, buffID: 2038, titleName: "Thần Ma Bất Dung", buffName: "Thần Ma Bất Dung", level: 1},
	{requirementID: 328, titleID: 129, buffID: 2038, titleName: "Thần Ma Bất Dung", buffName: "Thần Ma Bất Dung", level: 1},
	{requirementID: 329, titleID: 129, buffID: 2038, titleName: "Thần Ma Bất Dung", buffName: "Thần Ma Bất Dung", level: 1},
	{requirementID: 330, titleID: 129, buffID: 2038, titleName: "Thần Ma Bất Dung", buffName: "Thần Ma Bất Dung", level: 1},
	{requirementID: 331, titleID: 129, buffID: 2038, titleName: "Thần Ma Bất Dung", buffName: "Thần Ma Bất Dung", level: 1},
	{requirementID: 332, titleID: 129, buffID: 2038, titleName: "Thần Ma Bất Dung", buffName: "Thần Ma Bất Dung", level: 1},
	{requirementID: 335, titleID: 132, buffID: 2041, titleName: "Tấn Công Ảo Ma Tháp", buffName: "Tấn Công Ảo Ma Tháp", level: 1},
	{requirementID: 336, titleID: 107, buffID: 2016, titleName: "Giải Trừ Bạo Lực", buffName: "Giải Trừ Bạo Lực", level: 1},
	{requirementID: 337, titleID: 133, buffID: 2042, titleName: "Marathon Liên Tục", buffName: "Marathon Liên Tục", level: 1},
	{requirementID: 383, titleID: 144, buffID: 2095, titleName: "Cao Thủ Câu Cá", buffName: "Cao Thủ Câu Cá", level: 1},
	{requirementID: 384, titleID: 143, buffID: 2094, titleName: "Thần Thủ Câu Cá", buffName: "Thần Thủ Câu Cá", level: 2},
	{requirementID: 385, titleID: 142, buffID: 2093, titleName: "Thợ Câu Truyền Thuyết", buffName: "Thợ Câu Truyền Thuyết", level: 3},
	{requirementID: 582, titleID: 325, buffID: 3048, titleName: "Đại Thần Kỳ Thạch", buffName: "Đại Thần Kỳ Thạch", level: 1},
	{requirementID: 583, titleID: 326, buffID: 3047, titleName: "Bậc Thầy Kỳ Thạch", buffName: "Bậc Thầy Kỳ Thạch", level: 1},
	{requirementID: 584, titleID: 327, buffID: 3046, titleName: "Nhân Tài Kỳ Thạch", buffName: "Nhân Tài Kỳ Thạch", level: 1},
}

type TitleStateReader struct {
	repo    TitleStateRepository
	catalog TitleStateCatalog
}

func NewTitleStateReader(repo TitleStateRepository, catalog TitleStateCatalog) *TitleStateReader {
	return &TitleStateReader{repo: repo, catalog: catalog}
}

func (r *TitleStateReader) Name() string {
	return "title"
}

func (r *TitleStateReader) RequirementIDs() []int64 {
	ids := []int64{50, 51, 52}
	for _, definition := range exactTitleRequirements {
		ids = append(ids, definition.requirementID)
	}
	return ids
}

func (r *TitleStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil {
		return nil, fmt.Errorf("title state repository not configured")
	}
	titles, err := r.repo.GetCharacterTitles(ctx, characterID)
	if err != nil {
		return nil, err
	}
	specialTitles, err := r.repo.GetCharacterSpecialTitles(ctx, characterID)
	if err != nil {
		return nil, err
	}
	owned := make(map[int]struct{}, len(titles)+len(specialTitles))
	for _, titleID := range titles {
		if titleID > 0 {
			owned[titleID] = struct{}{}
		}
	}
	for _, titleID := range specialTitles {
		if titleID > 0 {
			owned[titleID] = struct{}{}
		}
	}
	count := len(owned)
	proofs := []StateProof{
		stateProof(50, count, 5),
		stateProof(51, count, 10),
		stateProof(52, count, 15),
	}
	for _, definition := range exactTitleRequirements {
		title := r.catalog.GetTitle(definition.titleID)
		buff := r.catalog.GetBuff(definition.buffID)
		if title == nil || int(title.B) != definition.buffID || title.N != definition.titleName || int(title.L) != definition.level || buff == nil || buff.Name != definition.buffName {
			return nil, fmt.Errorf("title authority for requirement %d is invalid", definition.requirementID)
		}
		_, hasTitle := owned[definition.titleID]
		proofs = append(proofs, stateProof(definition.requirementID, boolStateProgress(hasTitle), 1))
	}
	return proofs, nil
}
