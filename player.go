package tammany

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"sort"

	"github.com/SlothNinja/color"
	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("TammanyPlayer", newPlayer())
}

// Player represents a player of the game.
type Player struct {
	*game.Player
	Log              GameLog
	Chips            Chips
	PlayedChips      Chips
	Office           office
	PlacedBosses     int `form:"placed-bosses"`
	PlacedImmigrants int `form:"placed-immigrants"`
	LockedUp         int `form:"lockedup"`
	Slandered        int `form:"slandered"`
	SlanderChips     slanderChips
	Candidate        bool `form:"candidate"`
	HasBid           bool `form:"has-bid"`
	UsedOffice       bool `form:"used-office"`
}

// Game is a back reference to the game in which the player exists.
// TODO: Deprecated
func (p *Player) Game() (g *Game) {
	g = p.Player.Game().(*Game)
	log.Warningf("p.Game is deprecated and will eventually be removed.")
	return
}

func (p *Player) wardElectionReset() {
	p.HasBid = false
	p.Candidate = false
	p.PerformedAction = false
}

func (g *Game) termResetFor(p *Player) {
	g.beginningOfTurnResetFor(p)
	p.HasBid = false
	p.Candidate = false
	p.LockedUp = 0
	p.Slandered = 0
}

func (g *Game) beginningOfTurnResetFor(p *Player) {
	p.PerformedAction = false
	p.PlacedBosses = 0
	p.PlacedImmigrants = 0
	p.Log = make(GameLog, 0)
	p.UsedOffice = false
	g.SubPhase = noSubPhase

	// Reset of shared slander variables
	if g.Phase == actions {
		g.SlanderedPlayerID = noPlayerID
		g.SlanderNationality = noNationality
		g.CurrentWardID = noWard
		g.MoveFromWardID = noWard
	}
}

type slanderChips map[int]bool

func (sc slanderChips) At(i int) bool { return sc[i] }

// Players provides of slice of players that implements the sort.Interface.
type Players []*Player

// Len implments sort.Interface.
func (ps Players) Len() int { return len(ps) }

// Swap implments sort.Interface.
func (ps Players) Swap(i, j int) { ps[i], ps[j] = ps[j], ps[i] }

// ByAll implments sort.Interface for sorting by all compare methods.
type ByAll struct{ Players }

// Less implments sort.Interface for sorting by all compare methods.
func (s ByAll) Less(i, j int) bool {
	return s.Players[i].compare(s.Players[j]) == game.LessThan
}

// ByChipsAndMayor implments sort.Interface for sorting by favor chips and mayor.
type ByChipsAndMayor struct{ Players }

// Less implments sort.Interface for sorting by favor chips and mayor.
func (s ByChipsAndMayor) Less(i, j int) bool {
	return s.Players[i].compareWithoutScore(s.Players[j]) == game.LessThan
}

func (p *Player) compare(p2 *Player) (c game.Comparison) {
	if c = p.CompareByScore(p2.Player); c == game.EqualTo {
		c = p.compareWithoutScore(p2)
	}
	return
}

func (p *Player) compareWithoutScore(p2 *Player) (c game.Comparison) {
	if c = p.compareByTotalChips(p2); c == game.EqualTo {
		if c = p.compareByFavors(p2); c == game.EqualTo {
			c = p.compareByMayor(p2)
		}
	}
	return
}

func (p *Player) compareByTotalChips(p2 *Player) (c game.Comparison) {
	switch {
	case p.Chips.Count() < p2.Chips.Count():
		c = game.LessThan
	case p.Chips.Count() > p2.Chips.Count():
		c = game.GreaterThan
	default:
		c = game.EqualTo
	}
	return
}

func (p *Player) compareByFavors(p2 *Player) game.Comparison {
	for _, n := range nationalities() {
		switch {
		case p.Chips[n] < p2.Chips[n]:
			return game.LessThan
		case p.Chips[n] > p2.Chips[n]:
			return game.GreaterThan
		}
	}
	return game.EqualTo
}

func (p *Player) compareByMayor(p2 *Player) (c game.Comparison) {
	switch m1, m2 := p.isMayor(), p2.isMayor(); {
	case !m1 && m2:
		c = game.LessThan
	case m1 && !m2:
		c = game.GreaterThan
	default:
		c = game.EqualTo
	}
	return
}

// Equal returns true if players equal, false otherwise.
func (p *Player) Equal(op *Player) bool {
	return p != nil && op != nil && p.Player.Equal(op)
}

// NotEqual returns true if players not equal, false otherwise.
func (p *Player) NotEqual(op *Player) bool {
	return !p.Equal(op)
}

func (client *Client) determine2PPlaces(c *gin.Context, g *Game) ([]contest.ResultsMap, error) {
	// sort players by score
	players := g.Players()
	sort.Sort(ByAll{players})
	g.setPlayers(players)

	// find 'worse' player for each user
	ps := make(Players, len(g.Users))
	for i, u := range g.Users {
		for _, p := range g.Players() {
			if p.User().Equal(u) {
				ps[i] = p
				break
			}
		}
	}
	sort.Sort(Reverse{ByAll{ps}})
	return client.determinePlacesCommon(c, g)
}

func (client *Client) determinePlaces(c *gin.Context, g *Game) ([]contest.ResultsMap, error) {
	// sort players by score
	players := g.Players()
	sort.Sort(Reverse{ByAll{players}})
	g.setPlayers(players)
	return client.determinePlacesCommon(c, g)
}

func (client *Client) determinePlacesCommon(c *gin.Context, g *Game) ([]contest.ResultsMap, error) {
	places := make([]contest.ResultsMap, 0)
	for i, p1 := range g.Players() {
		rmap := make(contest.ResultsMap, 0)
		results := make([]*contest.Result, 0)
		for j, p2 := range g.Players() {
			r, err := client.Rating.For(c, p2.User(), g.Type)
			if err != nil {
				return nil, err
			}
			result := &contest.Result{
				GameID: g.ID(),
				Type:   g.Type,
				R:      r.R,
				RD:     r.RD,
			}
			switch c := p1.compare(p2); {
			case i == j:
			case c == game.GreaterThan:
				result.Outcome = 1
				results = append(results, result)
			case c == game.LessThan:
				result.Outcome = 0
				results = append(results, result)
			case c == game.EqualTo:
				result.Outcome = 0.5
			}
		}
		rmap[p1.User().Key] = results
		places = append(places, rmap)
	}
	return places, nil
}

//func (this *Game) determinePlaces() []Players {
//	// sort players by score
//	players := this.Players()
//	sort.Sort(Reverse{ByAll{players}})
//	this.SetPlayers(players)
//
//	places := make([]Players, 0)
//	player := this.Players()[0]
//	players = Players{player}
//	for _, p := range this.Players()[1:] {
//		if p.compare(player) == game.EqualTo {
//			players = append(players, p)
//		} else {
//			places = append(places, players)
//			players = Players{p}
//		}
//	}
//	return append(places, players)
//}

// Reverse implements sort.Interface for reverse order sorting.
type Reverse struct{ sort.Interface }

// Less implements sort.Interface for reverse order sorting.
func (si Reverse) Less(i, j int) bool { return si.Interface.Less(j, i) }

// Chips stores favor chips by nationality.
type Chips map[nationality]int

// Count provides a total count of favor chips regardless of nationality.
func (cs Chips) Count() (count int) {
	for _, value := range cs {
		count += value
	}
	return
}

// Any returns true if any chips present, otherwise false.
func (cs Chips) Any() bool {
	return cs.Count() > 0
}

// NationalityCount returns the number of nationalities represented in the chips.
func (cs Chips) NationalityCount() (count int) {
	for _, value := range cs {
		if value != 0 {
			count++
		}
	}
	return
}

// RemainingChips provides the chips remaining after a player bid.
func (p *Player) RemainingChips() (cs Chips) {
	cs = make(Chips, len(p.Chips))
	for nationality := range p.Chips {
		cs[nationality] = p.remainingChipsFor(nationality)
	}
	return
}

func (p *Player) remainingChipsFor(n nationality) int {
	return p.Chips[n] - p.PlayedChips[n]
}

// MaxInfluenceIn returns the maximal amount of influence a player has in a ward w if bid all relevant favor chips.
func (p *Player) MaxInfluenceIn(w *Ward) int {
	return w.BossesFor(p) + w.playableChipsFor(p)
}

func (p *Player) electionCountIn(w *Ward) int {
	return p.PlayedChips.Count() + w.BossesFor(p)
}

// ChipsFor returns the number of favor chips a player has for a given nationality.
func (p *Player) ChipsFor(n nationality) int {
	return p.Chips[n]
}

func (p *Player) init(gr game.Gamer) {
	p.SetGame(gr)
}

func newPlayer() (p *Player) {
	p = new(Player)
	p.Player = game.NewPlayer()
	return
}

func createPlayer(g *Game, id int) (p *Player) {
	p = newPlayer()
	p.SetID(id)
	p.SetGame(g)

	colorMap := g.DefaultColorMap()
	p.SetColorMap(make(color.Colors, g.NumPlayers))

	for i := 0; i < g.NumPlayers; i++ {
		index := (i - p.ID()) % g.NumPlayers
		if index < 0 {
			index += g.NumPlayers
		}
		color := colorMap[index]
		p.ColorMap()[i] = color
	}

	p.Chips = Chips{irish: 0, english: 0, german: 0, italian: 0}
	p.PlayedChips = Chips{irish: 0, english: 0, german: 0, italian: 0}
	p.SlanderChips = slanderChips{2: true, 3: true, 4: true}
	return
}

func (p *Player) bossImagePath() string {
	return fmt.Sprintf("/images/tammany/%s-ward-boss.png", p.Color())
}

// BossImage provides html img for a boss of the player.
func (p *Player) BossImage() template.HTML {
	return template.HTML(fmt.Sprintf(`<img alt="%s-ward-boss" src="%s" />`, p.Color(), p.bossImagePath()))
}

func (g *Game) bossImagePath(p *Player, cu *user.User) string {
	return fmt.Sprintf("/images/tammany/%s-ward-boss.png", g.Color(p, cu))
}

// BossImage provides html img for a boss of the player.
func (g *Game) BossImage(p *Player, cu *user.User) template.HTML {
	return template.HTML(fmt.Sprintf(`<img alt="%s-ward-boss" src="%s" />`, g.Color(p, cu), g.bossImagePath(p, cu)))
}

// Controlled provides a count of the immigrants of the given nationality in the wards having a boss of the player.
func (g *Game) ControlledBy(p *Player, n nationality) (cnt int) {
	for _, w := range g.ActiveWards() {
		if w.BossesFor(p) > 0 {
			cnt += w.Immigrants[n]
		}
	}
	return
}

func (p *Player) placedPieces() int {
	return p.PlacedBosses + p.PlacedImmigrants
}

// CanPlacePiecesIn returns true if the player can legally place pieces in the ward.
func (g *Game) CanPlacePiecesIn(u *user.User, w *Ward) bool {
	return !w.LockedUp && (g.canPlaceBoss(u) || g.canPlaceImmigrant(u))
}

func (g *Game) canPlaceBoss(u *user.User) bool {
	return g.IsCurrentPlayerOrAdmin(u) && g.CurrentPlayerFor(u).placedPieces() < 2 && g.ImmigrantInTransit == noNationality
}

// canPlaceImmigrant returns true if the player can legally place an immigrant.
func (g *Game) canPlaceImmigrant(u *user.User) bool {
	p := g.CurrentPlayerFor(u)
	return g.IsCurrentPlayerOrAdmin(u) && p.PlacedImmigrants == 0 &&
		p.placedPieces() < 2 && g.ImmigrantInTransit == noNationality
}

// CanUseOffice returns true if the player can legally use his office.
func (g *Game) CanUseOffice(p *Player) bool {
	return g.inActionPhase() && p.HasOffice() && !p.UsedOffice && !p.isMayor() &&
		(!p.isCouncilPresident() || p.LockedUp < 2)
}

// CanUseOfficeIn returns true if the user can legally use his office in the ward.
func (g *Game) CanUseOfficeIn(u *user.User, w *Ward) (result bool) {
	p := g.CurrentPlayerFor(u)
	return (g.CanLockup(u, w) || g.CanTakeChip(u) || g.CanRemoveImmigrant(u, w) ||
		g.CanMoveImmigrantFrom(u, w) || g.CanMoveImmigrantTo(u, w)) && !p.UsedOffice
}

// CanLockup returns true if the player can legally lock up the ward.
func (g *Game) CanLockup(u *user.User, w *Ward) bool {
	p := g.CurrentPlayerFor(u)
	return g.IsCurrentPlayerOrAdmin(u) && p.isCouncilPresident() && p.LockedUp < 2 &&
		!w.LockedUp && !p.UsedOffice && g.ImmigrantInTransit == noNationality
}

// CanTakeChip returns true if the user can legally take a favor chip.
func (g *Game) CanTakeChip(u *user.User) bool {
	p := g.CurrentPlayerFor(u)
	return g.IsCurrentPlayerOrAdmin(u) && p.isDeputyMayor() && !p.UsedOffice &&
		g.ImmigrantInTransit == noNationality
}

// CanRemoveImmigrant returns true if the user can legally remove an immigrant from the ward.
func (g *Game) CanRemoveImmigrant(u *user.User, w *Ward) bool {
	p := g.CurrentPlayerFor(u)
	return g.IsCurrentPlayerOrAdmin(u) && p.isChiefOfPolice() && !w.LockedUp && w.Immigrants.count() > 0 &&
		!p.UsedOffice && g.ImmigrantInTransit == noNationality
}

// CanMoveImmigrantFrom returns true if the user can legally move an immigrant from the ward.
func (g *Game) CanMoveImmigrantFrom(u *user.User, w *Ward) bool {
	p := g.CurrentPlayerFor(u)
	fromWard := g.moveFromWard()
	return g.IsCurrentPlayerOrAdmin(u) && p.isPrecinctChairman() && !w.LockedUp &&
		w.Immigrants.count() > 0 && fromWard == nil && !p.UsedOffice && g.ImmigrantInTransit == noNationality
}

// CanMoveImmigrantTo returns true if the user can legally move an immigrant to the ward.
func (g *Game) CanMoveImmigrantTo(u *user.User, w *Ward) bool {
	p := g.CurrentPlayerFor(u)
	fromWard := g.moveFromWard()
	return g.IsCurrentPlayerOrAdmin(u) && p.isPrecinctChairman() && !w.LockedUp &&
		w.Immigrants.count() > 0 && fromWard != nil && !p.UsedOffice && g.ImmigrantInTransit != noNationality
}

// CanSlander returns true if the user can legally slander in the ward.
func (g *Game) CanSlander(u *user.User, w *Ward) bool {
	p := g.CurrentPlayerFor(u)
	return g.IsCurrentPlayerOrAdmin(u) && w.BossesFor(p) > 0 && len(w.OtherBosses(p)) > 0 &&
		p.placedPieces() != 1 && w.playableChipsFor(p) > 0 && p.Slandered < 2 &&
		g.ImmigrantInTransit == noNationality
}

// HasOffice returns true if the player has an office.
func (p *Player) HasOffice() bool {
	return p.Office != noOffice
}

// CanSelectWard returns true if the user can legally select the ward.
func (g *Game) CanSelectWard(u *user.User, w *Ward) bool {
	return g.IsCurrentPlayerOrAdmin(u) && (g.Phase == actions || g.Phase == placeImmigrant) &&
		w != nil && g.activeWard(w.ID) && !w.LockedUp
}

// CanSelectOffice returns true if the user can legally select the office.
func (g *Game) CanSelectOffice(u *user.User, o office) bool {
	return g.IsCurrentPlayerOrAdmin(u) && g.Phase == assignCityOffices && !g.officeAssigned(o) &&
		!g.allPlayersHaveOffice()
}

func (g *Game) Color(p *Player, cu *user.User) color.Color {
	uid := g.UserIDS[p.ID()]
	cm := g.ColorMapFor(cu)
	return cm[int(uid)]
}

func (g *Game) GravatarFor(p *Player, cu *user.User) template.HTML {
	return template.HTML(fmt.Sprintf(`<a href=%q ><img src=%q alt="Gravatar" class="%s-border" /> </a>`,
		g.UserPathFor(p), user.GravatarURL(g.EmailFor(p), "80", g.GravTypeFor(p)), g.Color(p, cu)))
}
