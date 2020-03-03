package tammany

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"strings"

	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
)

const noWard = -1

func init() {
	var wardid wardID
	gob.RegisterName("*game.Nationality", new(nationality))
	gob.RegisterName("map[int]Chips", make(map[int]Chips))
	gob.RegisterName("game.BossesMap", BossesMap{})
	gob.RegisterName("game.WardID", wardid)
}

type wardID int

var wardIDValues = [...]wardID{1, 2, 4, 7, 6, 14, 9, 15, 8, 5, 3, 17, 11, 13, 10}
var wardIndices = map[wardID]int{1: 0, 2: 1, 4: 2, 7: 3, 6: 4, 14: 5, 9: 6, 15: 7, 8: 8, 5: 9, 3: 10, 17: 11, 11: 12, 13: 13, 10: 14}

var toWardID = map[string]wardID{
	"ward-1":  1,
	"ward-2":  2,
	"ward-3":  3,
	"ward-4":  4,
	"ward-5":  5,
	"ward-6":  6,
	"ward-7":  7,
	"ward-8":  8,
	"ward-9":  9,
	"ward-10": 10,
	"ward-11": 11,
	"ward-13": 13,
	"ward-14": 14,
	"ward-15": 15,
	"ward-17": 17,
}

// Ward represents a ward of the game.
type Ward struct {
	//game       *Game
	ID         wardID
	Immigrants Nationals
	Bosses     BossesMap
	LockedUp   bool
	Resolved   bool
}

var toOffice = map[string]office{
	"mayor":             mayor,
	"deputy-mayor":      deputyMayor,
	"council-president": councilPresident,
	"chief-of-police":   chiefOfPolice,
	"precinct-chairman": precinctChairman,
}

// BossesFor returns a count of the number of bosses the player has in the ward.
func (w *Ward) BossesFor(p *Player) int {
	return w.Bosses[p.ID()]
}

// OtherBosses returns a map indicating how many bosses each other player has in the ward.
func (w *Ward) OtherBosses(p *Player) (bm BossesMap) {
	bm = defaultBosses()
	for id, cnt := range w.Bosses {
		if cnt == 0 || id == p.ID() {
			delete(bm, id)
		} else {
			bm[id] = cnt
		}
	}
	return bm
}

// Equal returns true if the wards are equal, false otherwise.
func (w *Ward) Equal(w2 *Ward) bool {
	return w2 != nil && w.ID == w2.ID
}

func newWard(id wardID) (w *Ward) {
	w = new(Ward)
	w.ID = id
	w.Immigrants = defaultNationals()
	w.Bosses = defaultBosses()
	return
}

// Nationals provides a count of immigrants by nationality.
type Nationals map[nationality]int

// BossesMap maps player ids to a count of bosses.
type BossesMap map[int]int

func defaultNationals() Nationals {
	return Nationals{irish: 0, english: 0, german: 0, italian: 0}
}

func defaultBosses() BossesMap {
	return BossesMap{0: 0, 1: 0, 2: 0, 3: 0, 4: 0}
}

type nationality int

const (
	noNationality nationality = iota
	irish
	english
	german
	italian
)

var toNationality = map[string]nationality{
	"none":    noNationality,
	"irish":   irish,
	"english": english,
	"german":  german,
	"italian": italian,
}

// Nationalities provides slice of nationalities.
type Nationalities []nationality

// Nationalities returns the nationalities present in the game.
func (g *Game) Nationalities() Nationalities {
	return nationalities()
}

func nationalities() Nationalities {
	return Nationalities{irish, english, german, italian}
}

func (ns Nationals) draw() (n nationality) {
	i := sn.MyRand.Intn(ns.count())
	n = ns.at(i)
	ns[n]--
	return
}

func (ns Nationals) count() (cnt int) {
	for _, v := range ns {
		cnt += v
	}
	return
}

func (ns Nationals) any() bool {
	return ns.count() > 0
}

func (ns Nationals) empty() bool {
	return !ns.any()
}

func (ns Nationals) at(i int) (n nationality) {
	switch {
	case i < ns[irish]:
		n = irish
	case i < ns[irish]+ns[english]:
		n = english
	case i < ns[irish]+ns[english]+ns[german]:
		n = german
	default:
		n = italian
	}
	return
}

var nationalityStrings = map[nationality]string{irish: "Irish", english: "English", german: "German", italian: "Italian"}

func (n nationality) String() string {
	return nationalityStrings[n]
}

func (n nationality) LString() string {
	return strings.ToLower(n.String())
}

func (n nationality) Int() int {
	return int(n)
}

func (n nationality) Equal(n2 nationality) bool {
	return n == n2
}

func (n nationality) CubeImage() string {
	return fmt.Sprintf("/images/tammany/%s-cube.png", n.LString())
}

func (n nationality) ChipImage() string {
	return fmt.Sprintf("/images/tammany/%s-chip.png", n.LString())
}

// ActiveWards returns the active wards for the current term.
func (g *Game) ActiveWards() (ws Wards) {
	switch g.Term() {
	case 1:
		switch g.NumPlayers {
		case 5:
			ws = g.Wards
		case 4:
			ws = g.Wards[0:11]
		case 3:
			ws = g.Wards[0:6]
		}
	case 2:
		switch g.NumPlayers {
		case 4, 5:
			ws = g.Wards
		case 3:
			ws = g.Wards[0:11]
		}
	case 3, 4:
		ws = g.Wards
	}
	return
}

func (g *Game) activeWard(wid wardID) (b bool) {
	for _, w := range g.ActiveWards() {
		if b = w.ID == wid; b {
			return
		}
	}
	return
}

type wardIDS []wardID

var adjacentWards = map[wardID]wardIDS{
	1:  {2, 3},
	2:  {1, 3, 4, 6},
	3:  {1, 2, 5, 6},
	4:  {2, 6, 7},
	5:  {3, 6, 8},
	6:  {2, 3, 4, 5, 10, 14},
	7:  {4, 10, 13},
	8:  {5, 9, 14, 15},
	9:  {8, 15},
	10: {6, 7, 13, 14, 17},
	11: {13, 17},
	13: {7, 10, 11, 17},
	14: {6, 8, 10, 15, 17},
	15: {8, 9, 14, 17},
	17: {10, 11, 13, 14, 15},
}

func (ids wardIDS) include(wid wardID) (b bool) {
	for _, id := range ids {
		if b = id == wid; b {
			return
		}
	}
	return
}

func (ids wardIDS) String() (s string) {
	ss := make([]string, len(ids))
	for i := range ids {
		ss[i] = fmt.Sprintf("%d", ids[i])
	}
	return restful.ToSentence(ss)
}

func (w *Ward) adjacent(ward *Ward) bool {
	return adjacentWards[w.ID].include(ward.ID)
}

// Wards provides a slice of game wards.
type Wards []*Ward

func newWards() Wards {
	wards := make(Wards, len(wardIDValues))
	for i, id := range wardIDValues {
		wards[i] = newWard(id)
	}
	return wards
}

// Zone1Wards returns the wards in zone 1.
func (g *Game) Zone1Wards() (ws Wards) {
	if len(g.Wards) > 5 {
		ws = g.Wards[:6]
	}
	return
}

// Zone2Wards returns the wards in zone 2.
func (g *Game) Zone2Wards() (ws Wards) {
	if len(g.Wards) > 10 {
		ws = g.Wards[6:11]
	}
	return
}

// Zone3Wards returns the wards in zone 3.
func (g *Game) Zone3Wards() (ws Wards) {
	if len(g.Wards) > 11 {
		ws = g.Wards[11:]
	}
	return
}

// Zone2ImmigrantDisplay returns the immigrants to be displaye for zone 2 setup box.
func (g *Game) Zone2ImmigrantDisplay() Nationals {
	if (g.NumPlayers == 2 || g.NumPlayers == 3) && g.Term() == 1 {
		return defaultZone2Immigrants()
	}
	return nil
}

// Zone3ImmigrantDisplay returns the immigrants to be displaye for zone 3 setup box.
func (g *Game) Zone3ImmigrantDisplay() Nationals {
	switch {
	case (g.NumPlayers == 2 || g.NumPlayers == 3) && (g.Term() == 1 || g.Term() == 2):
		return defaultZone3Immigrants()
	case g.NumPlayers == 4 && g.Term() == 1:
		return defaultZone3Immigrants()
	}
	return nil
}

var wardCoords = map[wardID]string{
	1:  "250,1856,309,1616,493,1725,649,1848,483,1989,355,2049,300,1999,269,1924,273,1862",
	2:  "765,1757,652,1847,490,1723,438,1689,491,1608,577,1574,643,1640,656,1634",
	3:  "437,1690,309,1616,347,1335,582,1462",
	4:  "826,1714,768,1758,655,1635,647,1641,575,1575,774,1516,826,1509,880,1471,927,1672,819,1694",
	5:  "581,1461,346,1333,397,995,713,1265",
	6:  "494,1604,712,1266,784,1317,778,1336,914,1386,877,1475,825,1510,765,1517",
	7:  "1287,1609,1141,1627,929,1672,881,1471,1324,1423,1557,1510,1566,1550,1541,1589,1518,1601,1294,1629",
	8:  "395,994,408,860,668,886,880,1000,711,1264",
	9:  "408,860,445,381,485,313,874,504,669,886",
	10: "878,1469,930,1355,990,1170,1216,1246,1138,1441",
	11: "1494,829,1515,840,1521,857,1651,920,1621,975,1625,1014,1614,1076,1630,1092,1616,1150,1632,1166,1560,1357,1271,1261,1305,1166,1312,1169",
	13: "1611,1377,1560,1511,1321,1422,1137,1441,1218,1245",
	14: "915,1384,775,1337,783,1317,713,1264,883,997,1021,1067,929,1358",
	15: "1021,1068,669,888,876,503,1114,621,1097,759",
	17: "1493,803,1495,827,1312,1171,1305,1165,1269,1260,992,1170,1099,752,1113,621",
}

// Key provides a key used by the image map interface to identify the ward.
func (w *Ward) Key() template.HTML {
	return restful.HTML("ward-%d", w.ID)
}

// Coords provides region coordinates used by the image map interface to identify the boundaries of a ward.
func (w *Ward) Coords() template.HTML {
	return restful.HTML(wardCoords[w.ID])
}
