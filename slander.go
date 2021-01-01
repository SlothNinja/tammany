package tammany

import (
	"encoding/gob"
	"html/template"
	"strconv"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("*game.firstSlanderEntry", new(firstSlanderEntry))
	gob.RegisterName("*game.secondSlanderEntry", new(secondSlanderEntry))
}

// SlanderedPlayer returns the player that was slandered.
func (g *Game) SlanderedPlayer() *Player {
	return g.PlayerByID(g.SlanderedPlayerID)
}

func (g *Game) slander(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		p *Player
		w *Ward
		n nationality
	)

	if p, w, n, err = g.validateSlander(c, cu); err != nil {
		act = game.None
		return
	}

	// Log Placement
	cp := g.CurrentPlayer()

	if g.SlanderNationality == noNationality {
		// First Slander
		cp.Chips[n]--
		cp.SlanderChips[g.Term()] = false
		cp.Slandered++

		// Reusing CurrentWardID to maintain first Slandered Ward for the Action Phase,
		// since CurrentWardID is only used during the Election Phase, there should be no conflict.
		g.CurrentWardID = w.ID

		g.SlanderNationality = n
		g.SlanderedPlayerID = p.ID()
		w.Bosses[p.ID()]--

		// Log First Slander
		e := g.newFirstSlanderEntryFor(cp, w, p, n)
		restful.AddNoticef(c, string(e.HTML(c)))

	} else {
		// Second Slander
		cp.Chips[n] -= 2
		cp.Slandered++
		g.SlanderNationality = n
		g.SlanderedPlayerID = p.ID()
		w.Bosses[p.ID()]--

		// Log Second Slander
		e := g.newSecondSlanderEntryFor(cp, w, p, n)
		restful.AddNoticef(c, string(e.HTML(c)))

	}

	tmpl, act = "tammany/slander_update", game.Cache
	return
}

type firstSlanderEntry struct {
	*Entry
	WardID wardID
	Chip   nationality
}

func (g *Game) newFirstSlanderEntryFor(p *Player, w *Ward, op *Player, n nationality) (e *firstSlanderEntry) {
	e = new(firstSlanderEntry)
	e.Entry = g.newEntryFor(p)
	e.WardID = w.ID
	e.OtherPlayerID = op.ID()
	e.Chip = n
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *firstSlanderEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s used an %s favor to slander %s in ward %d",
		g.NameByPID(e.PlayerID), e.Chip, g.NameByPID(e.OtherPlayerID), e.WardID)
}

type secondSlanderEntry struct {
	*Entry
	WardID wardID
	Chip   nationality
}

func (g *Game) newSecondSlanderEntryFor(p *Player, w *Ward, op *Player, n nationality) (e *secondSlanderEntry) {
	e = new(secondSlanderEntry)
	e.Entry = g.newEntryFor(p)
	e.WardID = w.ID
	e.OtherPlayerID = op.ID()
	e.Chip = n
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *secondSlanderEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s used two %s favors to slander %s in ward %d",
		g.NameByPID(e.PlayerID), e.Chip, g.NameByPID(e.OtherPlayerID), e.WardID)
}

func (g *Game) validateSlander(c *gin.Context, cu *user.User) (p *Player, w *Ward, n nationality, err error) {
	var (
		cp   *Player
		nInt int
	)

	if nInt, err = strconv.Atoi(c.PostForm("slander-nationality")); err != nil {
		return
	}

	switch w, cp, n, p = g.getWard(c), g.CurrentPlayer(), nationality(nInt), g.playerBySID(c.PostForm("slandered-player")); {
	case !g.IsCurrentPlayer(cu):
		err = sn.NewVError("Only the current player can slander another player.")
	case w == nil:
		err = sn.NewVError("You must first select a ward.")
	case w.LockedUp:
		err = sn.NewVError("You can't slander a player in locked ward.")
	case w.Immigrants[n] < 1:
		err = sn.NewVError("You attempted to slander with a %s chip, but there are no %s immigrants in the selected ward.", n, n)
	case cp.placedPieces() == 1:
		err = sn.NewVError("You are in the process of placing pieces (immigrants and/or bosses).  You must use office before or after placing pieces, but not during.")
	case g.Phase != actions:
		err = sn.NewVError("Wrong phase for performing this action.")
	case g.Term() < 2:
		err = sn.NewVError("You can't slander in term %d.", g.Term())
	case g.SlanderNationality == noNationality && cp.Chips[n] < 1:
		err = sn.NewVError("You don't have a %s favor to use for the slander.", n)
	case g.SlanderNationality != noNationality && cp.Chips[n] < 2:
		err = sn.NewVError("You don't have two %s favors to use for the second slander.", n)
	case cp.Equal(p):
		err = sn.NewVError("You can't slander yourself.")
	case g.SlanderedPlayer() != nil && !g.SlanderedPlayer().Equal(p):
		err = sn.NewVError("You attempted to slander %s, but you are in the process or slandering %s.", g.NameFor(p), g.NameFor(g.SlanderedPlayer()))
	case cp.Slandered == 1 && !w.adjacent(g.CurrentWard()):
		err = sn.NewVError("Ward %d is not adjacent to ward %d.", w.ID, g.CurrentWardID)
	case cp.Slandered == 1 && g.SlanderNationality != n:
		err = sn.NewVError("You attempted to slander using %s favors, but you are in the process or slandering using %s favors.", n, g.SlanderNationality)
	case cp.Slandered >= 2:
		err = sn.NewVError("You have already slandered twice this term.")
	case cp.Slandered == 0 && !cp.CanSlanderIn(g.Term()):
		err = sn.NewVError("You have already slandered this term.")
	}
	return
}

// CanSlanderIn returns true if play can slander in the given term.
func (p *Player) CanSlanderIn(term int) bool {
	return p.SlanderChips[term]
}

func (g *Game) endSlander(cu *user.User) {
	cp := g.CurrentPlayer()
	if cp.Slandered == 1 {
		cp.Slandered = 2
	}
}
