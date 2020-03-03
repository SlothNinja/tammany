package tammany

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

// Register registers Tammany Hall with the server.
func Register(t gtype.Type, r *gin.Engine) {
	gob.Register(new(Game))
	game.Register(t, newGamer, phaseNames, nil)
	AddRoutes(t.Prefix(), r)
}

const noPlayerID = game.NoPlayerID

// Game provides a Tammany Hall game.
type Game struct {
	*game.Header
	*State
}

// State stores the game state of a Tammany Hall game.
type State struct {
	Playerers game.Playerers
	Log       GameLog

	Wards              Wards
	CastleGarden       Nationals
	Bag                Nationals
	CurrentWardID      wardID
	SelectedWardID     wardID
	MoveFromWardID     wardID
	SelectedOffice     office
	ImmigrantInTransit nationality
	SlanderedPlayerID  int
	SlanderNationality nationality

	ConfirmedOffice bool
}

const noWardID wardID = -1

// GetPlayerers implements game.GetPlayerers interface
func (g *Game) GetPlayerers() game.Playerers {
	return g.Playerers
}

// Players returns the players of the game.
func (g *Game) Players() (players Players) {
	ps := g.Playerers
	length := len(ps)
	if length > 0 {
		players = make(Players, length)
		for i, p := range ps {
			players[i] = p.(*Player)
		}
	}
	return
}

// CurrentPlayerLinks provides url links to the current players.
func (g *Game) CurrentPlayerLinks(c *gin.Context) template.HTML {
	cps := g.CurrentPlayers()
	if len(cps) == 0 || g.Status != game.Running {
		return "None"
	}

	var links string
	for _, cp := range cps {
		links += fmt.Sprintf("<div style='margin:3px'><img src=%q height='28px' style='vertical-align:middle' /> <span style='vertical-align:middle'>%s</span></div>", cp.bossImagePath(),
			g.PlayerLinkByID(c, cp.ID()%len(g.Users)))
	}
	return template.HTML(links)
}

func (g *Game) setPlayers(players Players) {
	length := len(players)
	if length > 0 {
		ps := make(game.Playerers, length)
		for i, p := range players {
			ps[i] = p
		}
		g.Playerers = ps
	}
}

// CurrentWard returns the ward currently conducting an election.
func (g *Game) CurrentWard() *Ward {
	return g.wardByID(g.CurrentWardID)
}

func (g *Game) wardByID(wid wardID) *Ward {
	index, ok := wardIndices[wid]
	if !ok {
		return nil
	}
	return g.Wards[index]
}

func (g *Game) setCurrentWard(w *Ward) {
	wid := noWardID
	if w != nil {
		wid = w.ID
	}
	g.CurrentWardID = wid
}

// SelectedWard provides the ward selected by the player in order to perform an action therein.
func (g *Game) SelectedWard() *Ward {
	return g.wardByID(g.SelectedWardID)
}

func (g *Game) setSelectedWard(w *Ward) {
	wid := noWardID
	if w != nil {
		wid = w.ID
	}
	g.SelectedWardID = wid
}

func (g *Game) moveFromWard() *Ward {
	return g.wardByID(g.MoveFromWardID)
}

func (g *Game) setMoveFromWard(w *Ward) {
	wid := noWardID
	if w != nil {
		wid = w.ID
	}
	g.MoveFromWardID = wid
}

// Term provides the current game term.
func (g *Game) Term() int {
	return (g.Round + 3) / 4
}

// Year provides the current game year.
func (g *Game) Year() int {
	return g.Round
}

func (g *Game) setYear(y int) {
	g.Round = y
}

// Games provides a slice of Games.
type Games []*Game

func (g *Game) start(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	g.Status = game.Running
	g.Phase = setup

	for i, u := range g.Users {
		g.addNewPlayer(u, i)
	}

	g.RandomTurnOrder()

	g.setYear(1)
	g.Wards = newWards()
	g.setSelectedWard(nil)
	g.setMoveFromWard(nil)
	g.setCurrentWard(nil)
	g.Bag = defaultBag()
	g.CastleGarden = defaultNationals()
	g.immigration()
	g.castleGardenPhase()
}

func (g *Game) addNewPlayer(u *user.User, id int) {
	p := createPlayer(g, u, id)
	g.Playerers = append(g.Playerers, p)
}

func (g *Game) startNextTerm() {
	g.setYear(g.Year() + 1)

	for _, p := range g.Players() {
		g.termResetFor(p)
	}

	g.unlockWards()

	g.immigration()
	g.castleGardenPhase()
}

func (g *Game) unlockWards() {
	for _, w := range g.ActiveWards() {
		w.LockedUp = false
	}
}

func (g *Game) actionsPhase() {
	g.Phase = actions
}

func (g *Game) startElections(c *gin.Context) contest.Contests {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	g.Phase = elections
	g.SubPhase = noSubPhase
	g.emptyGarden()
	for _, p := range g.Players() {
		g.beginningOfTurnResetFor(p)
	}
	for _, w := range g.ActiveWards() {
		w.Resolved = false
	}

	return g.continueElections(c)
}

func (g *Game) inActionPhase() bool {
	return g.Phase == actions
}

// InTakeChipPhase returns whether the game is in the take favor chip phase.
func (g *Game) InTakeChipPhase() bool {
	return g.Phase == takeFavorChip
}

// InElectionsPhase returns whether the game is in the election phase.
func (g *Game) InElectionsPhase() bool {
	return g.Phase == elections
}

// InOfficeWarningSubPhase returns whether the game is in the office warning subphase.
func (g *Game) InOfficeWarningSubPhase() bool {
	return g.SubPhase == officeWarning
}

func (g *Game) setCurrentPlayers(ps ...*Player) {
	var pers game.Playerers

	switch l := len(ps); {
	case l == 0:
		pers = nil
	case l == 1:
		pers = game.Playerers{ps[0]}
	default:
		pers = make(game.Playerers, l)
		for i, p := range ps {
			pers[i] = p
		}
	}
	g.SetCurrentPlayerers(pers...)
}

// PlayerByID returns the player having the id.
func (g *Game) PlayerByID(id int) (p *Player) {
	if per := game.PlayererByID(g.Playerers, id); per != nil {
		p = per.(*Player)
	}
	return
}

func (g *Game) playerBySID(sid string) (p *Player) {
	if per := game.PlayerBySID(g.Playerers, sid); per != nil {
		p = per.(*Player)
	}
	return
}

func (g *Game) playerByUserID(id int64) (p *Player) {
	if per := game.PlayererByUserID(g.Playerers, id); per != nil {
		p = per.(*Player)
	}
	return
}

func (g *Game) playerByIndex(i int) (p *Player) {
	if per := game.PlayererByIndex(g.Playerers, i); per != nil {
		p = per.(*Player)
	}
	return
}

func (g *Game) undoAction(c *gin.Context) (string, game.ActionType, error) {
	cp := g.CurrentPlayer()
	if !g.CUserIsCPlayerOrAdmin(c) {
		return "", game.None, sn.NewVError("Only the current player may perform this action.")
	}

	restful.AddNoticef(c, "%s undid action.", g.NameFor(cp))
	return "", game.Undo, nil
}

func (g *Game) resetTurn(c *gin.Context) (string, game.ActionType, error) {
	cp := g.CurrentPlayer()
	if !g.CUserIsCPlayerOrAdmin(c) {
		return "", game.None, sn.NewVError("Only the current player may perform this action.")
	}

	restful.AddNoticef(c, "%s reset turn.", g.NameFor(cp))
	return "", game.Reset, nil
}

func (g *Game) redoAction(c *gin.Context) (string, game.ActionType, error) {
	cp := g.CurrentPlayer()
	if !g.CUserIsCPlayerOrAdmin(c) {
		return "", game.None, sn.NewVError("Only the current player may perform this action.")
	}

	restful.AddNoticef(c, "%s redid action.", g.NameFor(cp))
	return "", game.Redo, nil
}

// CurrentPlayer returns the current player.
func (g *Game) CurrentPlayer() (p *Player) {
	if per := g.CurrentPlayerer(); per != nil {
		p = per.(*Player)
	}
	return
}

func (g *Game) candidates() (cs Players) {
	for _, p := range g.Players() {
		if p.Candidate {
			cs = append(cs, p)
		}
	}
	return
}

// CurrentPlayerFor provides the current player associated with user u.
// Returns nil if no current player is associate with user u.
func (g *Game) CurrentPlayerFor(u *user.User) (p *Player) {
	if per := g.Header.CurrentPlayerFor(g.Playerers, u); per != nil {
		p = per.(*Player)
	}
	return
}

// CurrentPlayers provides the current players of the game.
func (g *Game) CurrentPlayers() (ps Players) {
	for _, p := range g.CurrentPlayersFrom(g.Playerers) {
		ps = append(ps, p.(*Player))
	}
	return
}

func (g *Game) newTurnOrder(c *gin.Context) {
	if g.mayor() != nil {
		index := game.IndexFor(g.mayor(), g.Playerers)
		playersTwice := append(g.Players(), g.Players()...)
		newOrder := playersTwice[index : index+g.NumPlayers]
		g.setPlayers(newOrder)
	}
}
