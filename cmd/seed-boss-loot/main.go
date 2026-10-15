// Open-sourced by BaoLT

// Generates the deterministic map-boss and Beast Handbook loot seed.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	appactivity "mcgame-server/internal/domain/activity"
)

const (
	rewardItem      = 29
	rewardEquipment = 19
)

type bossRow struct {
	NID   int
	Level int
	Kind  string
	Tier  string
}

type lootRow struct {
	BossNID int
	ItemID  int
	Quality int
	Rate    int
	QtyMin  int
	QtyMax  int
	Type    int
	Notes   string
}

var materialIDs = []int{7, 21, 24, 25, 26, 28}
var rareIDs = []int{5, 274, 408, 1660, 1661, 1662}
var subWeaponIDs = []int{1418, 1421, 1422, 1423, 1424, 1425, 1426, 1427, 1428, 1429, 1430, 1431, 1661, 1662, 1663}
var gemIDs = [][]int{
	{9, 10, 11, 12, 13},
	{14, 15, 47, 48, 49},
	{50, 51, 52, 53, 54},
	{55, 56, 57, 58, 59},
	{60, 61, 62, 63, 64},
	{97, 98, 99, 100, 101, 102, 103, 104},
}

func main() {
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "local compose Postgres DSN")
	flag.Parse()
	if strings.TrimSpace(*dsn) == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL or -dsn is required")
		os.Exit(2)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, *dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	bosses, err := loadBosses(ctx, conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load bosses: %v\n", err)
		os.Exit(1)
	}
	loot := buildLoot(bosses)
	if err := validateCatalog(ctx, conn, loot); err != nil {
		fmt.Fprintf(os.Stderr, "validate rewards: %v\n", err)
		os.Exit(1)
	}
	emitSeed(loot)
}

func loadBosses(ctx context.Context, conn *pgx.Conn) ([]bossRow, error) {
	rows, err := conn.Query(ctx, `
		select b.nid, case when b.level > 0 then b.level else coalesce(r.level, 1) end,
		       b.kind, b.tier
		from data.data_tbl_schedule_boss b
		left join (
			select nid, max(level)::int as level
			from data.data_tbl_npc_creature
			group by nid
		) r on r.nid = b.nid
		where b.is_active
		order by b.nid
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bosses []bossRow
	for rows.Next() {
		var boss bossRow
		if err := rows.Scan(&boss.NID, &boss.Level, &boss.Kind, &boss.Tier); err != nil {
			return nil, err
		}
		bosses = append(bosses, boss)
	}
	return bosses, rows.Err()
}

func buildLoot(bosses []bossRow) []lootRow {
	loot := make([]lootRow, 0, len(bosses)*7)
	for _, boss := range bosses {
		profile := rewardProfile(boss)
		band := levelBand(boss.Level)
		quantity := 1 + band
		loot = append(loot, lootRow{
			BossNID: boss.NID,
			ItemID:  materialIDs[boss.NID%len(materialIDs)],
			Rate:    profile.materialRate,
			QtyMin:  quantity,
			QtyMax:  quantity + 1,
			Type:    rewardItem,
			Notes:   profile.name + " material band " + fmt.Sprint(band+1),
		})
		if profile.gemRate > 0 {
			pool := gemIDs[band]
			loot = append(loot, lootRow{
				BossNID: boss.NID,
				ItemID:  pool[boss.NID%len(pool)],
				Rate:    profile.gemRate,
				QtyMin:  1,
				QtyMax:  max(1, 1+band/2),
				Type:    rewardItem,
				Notes:   profile.name + " gem band " + fmt.Sprint(band+1),
			})
		}
		if profile.featherRate > 0 {
			grade := min(4, max(0, band-1))
			poolStart := 3036 + grade*22
			loot = append(loot, lootRow{
				BossNID: boss.NID,
				ItemID:  poolStart + boss.NID%22,
				Rate:    profile.featherRate,
				QtyMin:  1,
				QtyMax:  1,
				Type:    rewardItem,
				Notes:   profile.name + " feather grade " + fmt.Sprint(grade+1),
			})
		}
		loot = append(loot, lootRow{
			BossNID: boss.NID,
			ItemID:  rareIDs[band],
			Rate:    profile.rareRate,
			QtyMin:  1,
			QtyMax:  max(1, 1+band/3),
			Type:    rewardItem,
			Notes:   profile.name + " rare progression band " + fmt.Sprint(band+1),
		})
		if boss.Level >= 30 || profile.name != "ground" {
			loot = append(loot, lootRow{
				BossNID: boss.NID,
				ItemID:  subWeaponIDs[boss.NID%len(subWeaponIDs)],
				Quality: min(5, max(1, band)),
				Rate:    profile.subWeaponRate,
				QtyMin:  1,
				QtyMax:  1,
				Type:    rewardEquipment,
				Notes:   profile.name + " sub magic weapon",
			})
		}
		if boss.Level >= 55 || profile.name == "mythic" {
			loot = append(loot,
				lootRow{BossNID: boss.NID, ItemID: 4841, Rate: profile.infiniteDustRate, QtyMin: quantity, QtyMax: quantity + 2, Type: rewardItem, Notes: profile.name + " Infinite Dust"},
				lootRow{BossNID: boss.NID, ItemID: 4842, Rate: profile.magicDustRate, QtyMin: quantity, QtyMax: quantity + 1, Type: rewardItem, Notes: profile.name + " Magic Dust"},
			)
		}
		if profile.wingRate > 0 {
			loot = append(loot, lootRow{
				BossNID: boss.NID,
				ItemID:  1675,
				Quality: min(5, max(2, band)),
				Rate:    profile.wingRate,
				QtyMin:  1,
				QtyMax:  1,
				Type:    rewardEquipment,
				Notes:   "mythic Wings of Light",
			})
		}
	}
	return loot
}

type profile struct {
	name             string
	materialRate     int
	gemRate          int
	featherRate      int
	rareRate         int
	subWeaponRate    int
	infiniteDustRate int
	magicDustRate    int
	wingRate         int
}

func rewardProfile(boss bossRow) profile {
	if boss.Tier == "mythic" {
		return profile{name: "mythic", materialRate: 22500, gemRate: 12500, rareRate: 5000, subWeaponRate: 3000, infiniteDustRate: 12500, magicDustRate: 10000, wingRate: 1000}
	}
	if boss.Kind == "flying" {
		return profile{name: "flying", materialRate: 17500, featherRate: 10000, rareRate: 2500, subWeaponRate: 1500, infiniteDustRate: 7500, magicDustRate: 5000}
	}
	if boss.Kind == "daily_only" {
		if config := appactivity.GetBossConfig(boss.NID); config != nil && config.Num == 4 {
			return profile{name: "flying", materialRate: 17500, featherRate: 10000, rareRate: 2500, subWeaponRate: 1500, infiniteDustRate: 7500, magicDustRate: 5000}
		}
	}
	return profile{name: "ground", materialRate: 17500, gemRate: 10000, rareRate: 2500, subWeaponRate: 1500, infiniteDustRate: 7500, magicDustRate: 5000}
}

func levelBand(level int) int {
	switch {
	case level < 30:
		return 0
	case level < 55:
		return 1
	case level < 90:
		return 2
	case level < 120:
		return 3
	case level < 150:
		return 4
	default:
		return 5
	}
}

func validateCatalog(ctx context.Context, conn *pgx.Conn, loot []lootRow) error {
	items := make(map[int]struct{})
	equipment := make(map[int]struct{})
	for _, row := range loot {
		if row.Type == rewardEquipment {
			equipment[row.ItemID] = struct{}{}
		} else {
			items[row.ItemID] = struct{}{}
		}
	}
	if err := validateIDs(ctx, conn, "data.data_tbl_item_template", items); err != nil {
		return err
	}
	return validateIDs(ctx, conn, "data.data_tbl_equipt_template", equipment)
}

func validateIDs(ctx context.Context, conn *pgx.Conn, table string, expected map[int]struct{}) error {
	ids := make([]int, 0, len(expected))
	for id := range expected {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	rows, err := conn.Query(ctx, "select id from "+table+" where id = any($1)", ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := make(map[int]struct{}, len(ids))
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		found[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return fmt.Errorf("missing reward %s id %d", table, id)
		}
	}
	return rows.Err()
}

func emitSeed(loot []lootRow) {
	fmt.Println("-- Generated by cmd/seed-boss-loot")
	fmt.Println("truncate data.data_tbl_boss_loot restart identity;")
	for _, row := range loot {
		fmt.Printf("insert into data.data_tbl_boss_loot (boss_nid, item_id, quality, rate, qty_min, qty_max, bound, type, qid, notes) values (%d, %d, %d, %d, %d, %d, true, %d, -1, '%s');\n",
			row.BossNID, row.ItemID, row.Quality, row.Rate, row.QtyMin, row.QtyMax, row.Type, strings.ReplaceAll(row.Notes, "'", "''"))
	}
}
