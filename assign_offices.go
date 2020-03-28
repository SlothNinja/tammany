package tammany

import (
	"encoding/gob"
	"html/template"

	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("*game.assignedOfficeEntry", new(assignedOfficeEntry))
}

func (g *Game) startCityOfficesPhase(c *gin.Context) (cs contest.Contests) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	for _, player := range g.Players() {
		g.beginningOfTurnResetFor(player)
	}
	switch {
	case g.Year() == 16:
		cs = g.startEndGamePhase(c)
	case g.mayor() != nil:
		g.Phase = assignCityOffices
	default:
		g.startNextTerm()
	}
	return
}

func (g *Game) assignOffice(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		p *Player
		o office
	)

	if p, o, err = g.validateAssignOffice(c); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	p.Office = o
	cp := g.CurrentPlayer()
	if g.allPlayersHaveOffice() {
		cp.PerformedAction = true
	}

	// Log Assignment
	e := g.newAssignedOfficeEntryFor(cp, o, p)
	restful.AddNoticef(c, string(e.HTML(c)))
	tmpl, act = "tammany/assign_office", game.Cache
	return
}

type assignedOfficeEntry struct {
	*Entry
	Office office
}

func (g *Game) newAssignedOfficeEntryFor(p *Player, o office, op *Player) (e *assignedOfficeEntry) {
	e = new(assignedOfficeEntry)
	e.Entry = g.newEntryFor(p)
	e.Office = o
	e.OtherPlayerID = op.ID()
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *assignedOfficeEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s assigned %s the office of %s.",
		g.NameByPID(e.PlayerID), g.NameByPID(e.OtherPlayerID), e.Office)
}

func (g *Game) allPlayersHaveOffice() bool {
	for _, p := range g.Players() {
		if !p.hasAnOffice() {
			return false
		}
	}
	return true
}

func (p *Player) hasAnOffice() bool {
	return p.Office != noOffice
}

func (g *Game) validateAssignOffice(c *gin.Context) (p *Player, o office, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	cp := g.CurrentPlayer()

	switch o, p = g.getOffice(c), g.playerBySID(c.PostForm("pid")); {
	case !g.CUserIsCPlayerOrAdmin(c):
		err = sn.NewVError("Only the current player can select an office.")
	case o == noOffice:
		err = sn.NewVError("Invalid office assigned.")
	case g.CurrentPlayer().PerformedAction:
		err = sn.NewVError("You have already performed an action.")
	case g.officeAssigned(o):
		err = sn.NewVError("%s office has already been assigned.", o)
	case !officeValues.include(o):
		err = sn.NewVError("Invalid value received for office.", o)
	case p == nil:
		err = sn.NewVError("Invalid value received for player.")
	case p.Office != noOffice:
		err = sn.NewVError("%s has already been assigned the office of %s", g.NameFor(p), p.Office)
	case g.Phase == assignDeputyMayor:
		switch {
		case g.mayor() == nil:
			err = sn.NewVError("There is no Mayor to appoint a Deputy Mayor.")
		case !cp.isMayor() && !g.IsAdmin():
			err = sn.NewVError("You are not the Mayor and therefore can't assign offices.")
		case o != deputyMayor:
			err = sn.NewVError("The mayor must first appoint a Deputy Mayor.")
		}
	case g.Phase == deputyMayorAssignOffice:
		switch {
		case g.deputyMayor() == nil:
			err = sn.NewVError("There is no Deputy Mayor to assign offices.")
		case !cp.isDeputyMayor() && !g.IsAdmin():
			err = sn.NewVError("You are not the Deputy Mayor and therefore can't assign offices.")
		}
	case g.Phase == assignCityOffices:
		switch {
		case g.mayor() == nil:
			err = sn.NewVError("There is no Mayor to assign offices.")
		case !cp.isMayor() && !g.IsAdmin():
			err = sn.NewVError("You are not the Mayor and therefore can't assign offices.")
		}
	}
	return
}
