package tammany

import (
	"encoding/gob"
	"html/template"

	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("*game.assignedOfficeEntry", new(assignedOfficeEntry))
}

func (client *Client) startCityOfficesPhase(c *gin.Context, g *Game) ([]*contest.Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	for _, player := range g.Players() {
		g.beginningOfTurnResetFor(player)
	}
	switch {
	case g.Year() == 16:
		return client.startEndGamePhase(c, g)
	case g.mayor() != nil:
		g.Phase = assignCityOffices
		return nil, nil
	default:
		g.startNextTerm()
		return nil, nil
	}
}

func (g *Game) assignOffice(c *gin.Context, cu *user.User) (string, game.ActionType, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	p, o, err := g.validateAssignOffice(c, cu)
	if err != nil {
		return "tammany/flash_notice", game.None, err
	}

	p.Office = o
	cp := g.CurrentPlayer()
	if g.allPlayersHaveOffice() {
		cp.PerformedAction = true
	}

	// Log Assignment
	e := g.newAssignedOfficeEntryFor(cp, o, p)
	restful.AddNoticef(c, string(e.HTML(c, g, cu)))
	return "tammany/assign_office", game.Cache, nil
}

type assignedOfficeEntry struct {
	*Entry
	Office office
}

func (g *Game) newAssignedOfficeEntryFor(p *Player, o office, op *Player) *assignedOfficeEntry {
	e := new(assignedOfficeEntry)
	e.Entry = g.newEntryFor(p)
	e.Office = o
	e.OtherPlayerID = op.ID()
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *assignedOfficeEntry) HTML(c *gin.Context, g *Game, cu *user.User) template.HTML {
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

func (g *Game) validateAssignOffice(c *gin.Context, cu *user.User) (*Player, office, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	cp := g.CurrentPlayer()

	o, p := g.getOffice(c), g.playerBySID(c.PostForm("pid"))
	switch {
	case !g.IsCurrentPlayer(cu):
		return nil, noOffice, sn.NewVError("Only the current player can select an office.")
	case o == noOffice:
		return nil, noOffice, sn.NewVError("Invalid office assigned.")
	case g.CurrentPlayer().PerformedAction:
		return nil, noOffice, sn.NewVError("You have already performed an action.")
	case g.officeAssigned(o):
		return nil, noOffice, sn.NewVError("%s office has already been assigned.", o)
	case !officeValues.include(o):
		return nil, noOffice, sn.NewVError("Invalid value received for office.", o)
	case p == nil:
		return nil, noOffice, sn.NewVError("Invalid value received for player.")
	case p.Office != noOffice:
		return nil, noOffice, sn.NewVError("%s has already been assigned the office of %s", g.NameFor(p), p.Office)
	case g.Phase == assignDeputyMayor && g.mayor() == nil:
		return nil, noOffice, sn.NewVError("There is no Mayor to appoint a Deputy Mayor.")
	case g.Phase == assignDeputyMayor && !cp.isMayor() && !cu.IsAdmin():
		return nil, noOffice, sn.NewVError("You are not the Mayor and therefore can't assign offices.")
	case g.Phase == assignDeputyMayor && o != deputyMayor:
		return nil, noOffice, sn.NewVError("The mayor must first appoint a Deputy Mayor.")
	case g.Phase == deputyMayorAssignOffice && g.deputyMayor() == nil:
		return nil, noOffice, sn.NewVError("There is no Deputy Mayor to assign offices.")
	case g.Phase == deputyMayorAssignOffice && !cp.isDeputyMayor() && !cu.IsAdmin():
		return nil, noOffice, sn.NewVError("You are not the Deputy Mayor and therefore can't assign offices.")
	case g.Phase == assignCityOffices && g.mayor() == nil:
		return nil, noOffice, sn.NewVError("There is no Mayor to assign offices.")
	case g.Phase == assignCityOffices && !cp.isMayor() && !cu.IsAdmin():
		return nil, noOffice, sn.NewVError("You are not the Mayor and therefore can't assign offices.")
	default:
		return p, o, nil
	}
}
