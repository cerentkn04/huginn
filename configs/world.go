package configs

import "encoding/json"

type RegionBounds struct {
    MinX int `yaml:"min_x" json:"min_x"`
    MinY int `yaml:"min_y" json:"min_y"`
    MaxX int `yaml:"max_x" json:"max_x"`
    MaxY int `yaml:"max_y" json:"max_y"`
}

type Region struct {
    ID     string       `yaml:"id" json:"id"`
    Bounds RegionBounds `yaml:"bounds" json:"bounds"`
}
type WorldGroup struct {
    ID      string   `yaml:"id" json:"id"`
    Regions []Region `yaml:"regions" json:"regions"`
}
func (wg WorldGroup) Neighbors(regionID string) []string {
    var neighbors []string

    var current Region
    for _, r := range wg.Regions {
        if r.ID == regionID {
            current = r
            break
        }
    }

    for _, other := range wg.Regions {
        if other.ID == regionID {
            continue
        }

        if areNeighbors(current, other) {
            neighbors = append(neighbors, other.ID)
        }
    }

    return neighbors
}
func areNeighbors(a, b Region) bool {
    aB := a.Bounds
    bB := b.Bounds

    horizontal :=
        (aB.MaxX == bB.MinX || aB.MinX == bB.MaxX) &&
        aB.MinY < bB.MaxY &&
        aB.MaxY > bB.MinY

    vertical :=
        (aB.MaxY == bB.MinY || aB.MinY == bB.MaxY) &&
        aB.MinX < bB.MaxX &&
        aB.MaxX > bB.MinX

    return horizontal || vertical
}

