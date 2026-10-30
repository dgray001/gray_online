package risq

type wallEntryKind uint8

const (
	wallEntryUnresolved wallEntryKind = iota
	wallEntryOpen
	wallEntryBlocked
	wallEntryBorderCombat
	wallEntryClimb
)

type wallEntry struct {
	kind wallEntryKind
	cost uint
}

func (e wallEntry) stopsMove() bool {
	if e.kind == wallEntryUnresolved {
		panic("movement wall entry was not resolved")
	}
	return e.kind == wallEntryBlocked || e.kind == wallEntryBorderCombat
}

// What the deciding side knows about the wall in the zone being entered
type wallState struct {
	blocks   bool
	defended bool
}

func (p pathProfile) decideWallEntry(wall wallState, attacking_wall bool) wallEntry {
	switch {
	case !wall.blocks:
		return wallEntry{kind: wallEntryOpen}
	case attacking_wall:
		return wallEntry{kind: wallEntryBorderCombat}
	case p.scalesWall(wall.defended):
		return wallEntry{kind: wallEntryClimb, cost: wallScaleCost}
	default:
		return wallEntry{kind: wallEntryBlocked}
	}
}

func (u *RisqUnit) plannedWallEntry(from *RisqZone, to *RisqZone, attacking_wall bool) wallEntry {
	cache, blocks := to.knownBlockingWall(from, u.player_id)
	return u.path_profile.decideWallEntry(wallState{blocks: blocks, defended: cache.has_garrisoned_units}, attacking_wall)
}

func (u *RisqUnit) resolvedWallEntry(move *MoveIntent, walls, defended map[*RisqBuilding]bool) wallEntry {
	to := move.next_step
	attacking_wall := move.attacking && to == move.goal
	blocks := to.wallBlocksEntry(u.zone, u.player_id, walls)
	return u.path_profile.decideWallEntry(wallState{blocks: blocks, defended: defended[to.building]}, attacking_wall)
}
