package tammany

import (
	"encoding/gob"
	"html/template"
	"sort"
	"strings"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("*game.placedLockUpMarkerEntry", new(placedLockUpMarkerEntry))
	gob.RegisterName("*game.movedImmigrantEntry", new(movedImmigrantEntry))
}

type office int

const (
	noOffice office = iota
	mayor
	deputyMayor
	councilPresident
	chiefOfPolice
	precinctChairman
)

// Value provides an integer representation of an office.
func (o office) Value() int {
	return int(o)
}

// String provides a string representation of an office.
func (o office) String() string {
	return officeStrings[o]
}

// IDString provides an id string representatino of an office.
func (o office) IDString() string {
	return strings.Replace(strings.ToLower(o.String()), " ", "-", -1)
}

// Offices provides a slice of offices.
type Offices []office

func (os Offices) include(o2 office) (b bool) {
	for _, o := range os {
		if b = o == o2; b {
			return
		}
	}
	return
}

var assignableOfficeValues = Offices{mayor, deputyMayor, councilPresident, chiefOfPolice, precinctChairman}
var officeValues = Offices{noOffice, mayor, deputyMayor, councilPresident, chiefOfPolice, precinctChairman}
var officeStrings = [...]string{
	noOffice:         "None",
	mayor:            "Mayor",
	deputyMayor:      "Deputy Mayor",
	councilPresident: "Council President",
	chiefOfPolice:    "Chief Of Police",
	precinctChairman: "Precinct Chairman",
}

// Offices provides a slice of the all offices.
func (g *Game) Offices() Offices {
	return officeValues
}

// AssignableOffices provides a list of the assignable offices.
func (g *Game) AssignableOffices() Offices {
	return assignableOfficeValues
}

// PlayerByOffice returns the player have the office o.
func (g *Game) PlayerByOffice(o office) (p *Player) {
	for _, p2 := range g.Players() {
		if p2.hasOffice(o) {
			p = p2
			return
		}
	}
	return
}

func (p *Player) hasOffice(o office) bool {
	return p.Office == o
}

func (g *Game) mayor() *Player {
	return g.PlayerByOffice(mayor)
}

func (g *Game) deputyMayor() *Player {
	return g.PlayerByOffice(deputyMayor)
}

func (g *Game) councilPresident() *Player {
	return g.PlayerByOffice(councilPresident)
}

func (g *Game) chiefOfPolice() *Player {
	return g.PlayerByOffice(chiefOfPolice)
}

func (g *Game) precinctChairman() *Player {
	return g.PlayerByOffice(precinctChairman)
}

func (g *Game) officeAssigned(o2 office) bool {
	return g.PlayerByOffice(o2) != nil
}

func (p *Player) isMayor() bool {
	return p.Office == mayor
}

func (p *Player) isDeputyMayor() bool {
	return p.Office == deputyMayor
}

func (p *Player) isCouncilPresident() bool {
	return p.Office == councilPresident
}

func (p *Player) isChiefOfPolice() bool {
	return p.Office == chiefOfPolice
}

func (p *Player) isPrecinctChairman() bool {
	return p.Office == precinctChairman
}

func (g *Game) newMayor(results *electionResults) *Player {
	currentMayor := g.mayor()
	mayors := make(Players, 0)
	wonWards := 0
	for _, player := range g.Players() {
		pWards := len(results.PlayerResults[player.ID()].WardIDS)
		switch {
		case pWards == wonWards:
			mayors = append(mayors, player)
		case pWards > wonWards:
			wonWards = pWards
			mayors = Players{player}
		}
	}

	if len(mayors) == 1 {
		return mayors[0]
	}

	sort.Sort(Reverse{ByChipsAndMayor{mayors}})
	if mayors[0].compareWithoutScore(mayors[1]) == game.GreaterThan {
		return mayors[0]
	}

	// Unable to break tie.  Current Mayor stays in power.
	return currentMayor
}

func (g *Game) tieBreaker(players Players) Players {
	// Number of Chips tiebreaker
	ps := make(Players, 0)
	var chips int
	for _, p := range players {
		pchips := p.Chips.Count()
		switch {
		case pchips == chips:
			ps = append(ps, p)
		case pchips > chips:
			chips = pchips
			ps = Players{p}
		}
	}

	if len(ps) == 1 {
		return ps
	}

	// Nationality Tiebreakers
	for _, nationality := range g.Nationalities() {
		ps = g.nationalityTieBreaker(nationality, ps)
		if len(ps) == 1 {
			return ps
		}
	}
	return ps
}

func (g *Game) nationalityTieBreaker(n nationality, players Players) Players {
	ps := make(Players, 0)
	chips := 0
	for _, p := range players {
		pchips := p.ChipsFor(n)
		switch {
		case pchips == chips:
			ps = append(ps, p)
		case pchips > chips:
			chips = pchips
			ps = Players{p}
		}
	}
	return ps
}

func (g *Game) placeLockupMarker(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	var w *Ward
	if w, err = g.validatePlaceLockupMarker(c, cu); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	// Log Placement
	cp := g.CurrentPlayer()
	e := g.newPlacedLockUpMarkerEntryFor(cp)
	e.WardID = w.ID

	// LockUp Ward
	w.LockedUp = true
	cp.UsedOffice = true
	cp.LockedUp++

	tmpl, act = "tammany/place_pieces", game.Cache
	return
}

type placedLockUpMarkerEntry struct {
	*Entry
	WardID wardID
}

func (g *Game) newPlacedLockUpMarkerEntryFor(p *Player) (e *placedLockUpMarkerEntry) {
	e = new(placedLockUpMarkerEntry)
	e.Entry = g.newEntryFor(p)
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *placedLockUpMarkerEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s locked-up ward %d.", g.NameByPID(e.PlayerID), e.WardID)
}

func (g *Game) validatePlaceLockupMarker(c *gin.Context, cu *user.User) (w *Ward, err error) {
	var (
		cp, prez *Player
	)

	switch w, cp, prez = g.getWard(c), g.CurrentPlayer(), g.councilPresident(); {
	case !g.IsCurrentPlayer(cu):
		err = sn.NewVError("Only the current player can lockup a ward.")
	case w == nil:
		err = sn.NewVError("You must first select a ward.")
	case w.LockedUp:
		err = sn.NewVError("You can't place lockup an already locked ward.")
	case cp.UsedOffice:
		err = sn.NewVError("You have already lockedup a ward this year.")
	case cp.hasPlacedOnePiece():
		err = sn.NewVError("You are in the process of placing pieces (immigrants and/or bosses).  You must use office before or after placing pieces, but not during.")
	case !g.inActionPhase():
		err = sn.NewVError("Wrong phase for performing this action.")
	case cp.LockedUp >= 2:
		err = sn.NewVError("You have already lockedup two wards this term.")
	case cp.NotEqual(prez):
		err = sn.NewVError("You are the %s.  Only the Council President may lockup a ward.", cp.Office)
	}
	return
}

func (g *Game) moveFrom(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	var (
		w *Ward
		n nationality
	)

	if w, n, err = g.validateMoveFrom(c, cu); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	// Move Immigrant From Ward
	g.endSlander(cu)
	w.Immigrants[n]--
	g.Bag[n]++
	g.setMoveFromWard(w)
	g.ImmigrantInTransit = n

	tmpl, act = "tammany/place_pieces", game.Cache
	return
}

func (g *Game) validateMoveFrom(c *gin.Context, cu *user.User) (w *Ward, n nationality, err error) {
	var cp, chairman *Player

	switch n, w, cp, chairman = getNationality(c), g.getWard(c), g.CurrentPlayer(), g.precinctChairman(); {
	case !g.IsCurrentPlayer(cu):
		err = sn.NewVError("Only the current player can move an immigrant between wards.")
	case w == nil:
		err = sn.NewVError("You must first select a ward.")
	case w.LockedUp:
		err = sn.NewVError("You can't move an immigrant from a locked ward.")
	case cp.placedPieces() == 1:
		err = sn.NewVError("You must move an immigrant before or after the placing pieces action, not during.")
	case !g.inActionPhase():
		err = sn.NewVError("You can't move an immigrant during the %s phase.", g.Phase)
	case w.hasOneImmigrant():
		err = sn.NewVError("You can't move the last immigrant from the ward.")
	case cp.NotEqual(chairman):
		err = sn.NewVError("You are the %s.  Only the Precinct Chairman can move an immigrant between wards.", cp.Office)
	}
	return
}

func (w *Ward) hasOneImmigrant() bool {
	return w.Immigrants.count() == 1
}

func (g *Game) moveTo(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	var (
		w *Ward
		n nationality
	)

	if w, n, err = g.validateMoveTo(c, cu); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	// Log Placement
	cp := g.CurrentPlayer()
	e := g.newMovedImmigrantEntryFor(cp, g.MoveFromWardID, w.ID, n)
	restful.AddNoticef(c, string(e.HTML(c)))

	// Move Immigrant To Ward
	g.endSlander(cu)
	w.Immigrants[n]++
	g.Bag[n]--
	cp.UsedOffice = true
	g.ImmigrantInTransit = noNationality

	tmpl, act = "tammany/place_pieces", game.Cache
	return
}

type movedImmigrantEntry struct {
	*Entry
	FromWardID wardID
	ToWardID   wardID
	Immigrant  nationality
}

func (g *Game) newMovedImmigrantEntryFor(p *Player, fromID, toID wardID, n nationality) (e *movedImmigrantEntry) {
	e = new(movedImmigrantEntry)
	e.Entry = g.newEntryFor(p)
	e.FromWardID = fromID
	e.ToWardID = toID
	e.Immigrant = n
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *movedImmigrantEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	p := g.PlayerByID(e.PlayerID)
	return restful.HTML("%s moved a %s immigrant from ward %d to ward %d.", g.NameFor(p), e.Immigrant, e.FromWardID, e.ToWardID)
}

func (g *Game) validateMoveTo(c *gin.Context, cu *user.User) (w *Ward, n nationality, err error) {
	var cp, chairman *Player

	switch n, w, cp, chairman = getNationality(c), g.getWard(c), g.CurrentPlayer(), g.precinctChairman(); {
	case !g.IsCurrentPlayer(cu):
		err = sn.NewVError("Only the current player can place a boss.")
	case w == nil:
		err = sn.NewVError("You must first select a ward.")
	case w.LockedUp:
		err = sn.NewVError("You can't move an immigrant to a locked ward.")
	case cp.UsedOffice:
		err = sn.NewVError("You have already used your office power.")
	case cp.UsedOffice:
		err = sn.NewVError("You have already used your office power.")
	case g.Bag[n] <= 0:
		err = sn.NewVError("The Immigrant Bag does not have a %s cube to place.", n)
	case g.ImmigrantInTransit != n:
		err = sn.NewVError("Expected placement of %s immigrant, but received placement of %s immigrant.", g.ImmigrantInTransit, n)
	case chairman != nil && !cp.Equal(chairman):
		err = sn.NewVError("You are the %s.  Only the Precinct Chairman can move an immigrant between wards.", cp.Office)
	case !w.adjacent(g.moveFromWard()):
		err = sn.NewVError("Ward %d is not adjacent to ward %d.", w.ID, g.MoveFromWardID)
	}
	return
}

var officeCoords = map[string]string{
	"deputy-mayor":      "1675,358,1938,360,1937,511,1673,511",
	"council-president": "1673,680,1673,523,1938,525,1937,681",
	"chief-of-police":   "1674,915,1673,688,1937,688,1937,918",
	"precinct-chairman": "1673,1111,1675,928,1937,928,1939,1110",
}

func (o office) AdminKey() template.HTML {
	return restful.HTML("admin-%s", o.IDString())
}

func (o office) Key() template.HTML {
	return restful.HTML(o.IDString())
}

func (o office) Coords() template.HTML {
	return restful.HTML(officeCoords[o.IDString()])
}
