package tammany

import (
	"net/http"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
)

func (client Client) finish(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := gameFrom(c)
		cu, err := client.User.Current(c)
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
			return
		}

		oldCP := g.CurrentPlayer()

		var (
			s  *stats.Stats
			cs contest.Contests
		)

		switch g.Phase {
		case actions:
			s, cs, err = client.actionsPhaseFinishTurn(c, g, cu)
		case placeImmigrant:
			s, cs, err = client.placeImmigrantPhaseFinishTurn(c, g, cu)
		case takeFavorChip:
			s, cs, err = client.takeChipPhaseFinishTurn(c, g, cu)
		case elections:
			s, cs, err = client.electionPhaseFinishTurn(c, g, cu)
		case assignCityOffices:
			s, err = g.assignOfficesPhaseFinishTurn(c, cu)
		default:
			err = sn.NewVError("Improper Phase for finishing turn.")
		}

		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
			return
		}

		// cs != nil then game over
		if cs != nil {
			g.Phase = gameOver
			g.Status = game.Completed
			s = s.GetUpdate(c, g.UpdatedAt)
			ks, es := wrap(s, cs)
			err = client.saveWith(c, g, cu, ks, es)
			if err != nil {
				log.Errorf(err.Error())
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
			err = g.sendEndGameNotifications(c)
			if err != nil {
				log.Warningf(err.Error())
			}
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
			return
		}

		s = s.GetUpdate(c, g.UpdatedAt)
		err = client.saveWith(c, g, cu, []*datastore.Key{s.Key}, []interface{}{s})
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
			return
		}

		newCP := g.CurrentPlayer()
		if newCP != nil && oldCP.ID() != newCP.ID() {
			err = g.SendTurnNotificationsTo(c, newCP)
			if err != nil {
				log.Warningf(err.Error())
			}
		}

		c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
	}
}

func (g *Game) validateFinishTurn(c *gin.Context, cu *user.User) (*stats.Stats, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	cp, s := g.CurrentPlayerFor(cu), stats.Fetched(c)
	switch {
	case s == nil:
		return nil, sn.NewVError("missing stats for player.")
	case cu == nil:
		return nil, sn.NewVError("missing current user.")
	case !cp.IsCurrentUser(cu):
		return nil, sn.NewVError("Only the current player may finish a turn.")
	case !cp.PerformedAction:
		return nil, sn.NewVError("%s has yet to perform an action.", cu.Name)
	case g.ImmigrantInTransit != noNationality:
		return nil, sn.NewVError("You must complete move of %s immigrant before finishing turn.", g.ImmigrantInTransit)
	default:
		return s, nil
	}
}

// ps is an optional parameter.
// If no player is provided, assume current player.
func (g *Game) nextPlayer(ps ...game.Playerer) *Player {
	if nper := g.NextPlayerer(ps...); nper != nil {
		return nper.(*Player)
	}
	return nil
}

func (client Client) actionsPhaseFinishTurn(c *gin.Context, g *Game, cu *user.User) (*stats.Stats, contest.Contests, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	s, err := g.validateActionsPhaseFinishTurn(c, cu)
	if err != nil {
		return nil, nil, err
	}

	cp := g.CurrentPlayer()
	if g.CanUseOffice(cp) && c.PostForm("action") != "confirm-finish" {
		g.SubPhase = officeWarning
		client.Cache.SetDefault(g.UndoKey(c, cu), g)
		return s, nil, nil
	}

	restful.AddNoticef(c, "%s finished turn.", g.NameFor(cp))

	np := g.nextPlayer()
	g.beginningOfTurnResetFor(np)
	g.setCurrentPlayers(np)

	if game.IndexFor(np, g.Playerers) == 0 {
		switch g.Year() {
		case 4, 8, 12, 16:
			cs, err := client.startElections(c, g, cu)
			if err != nil {
				return nil, nil, err
			}
			// if cs != nil then end game
			if cs != nil {
				return s, cs, nil
			}
		default:
			g.setYear(g.Year() + 1)
		}
	}

	if g.Phase == actions {
		g.castleGardenPhase()
	}

	return s, nil, nil
}

func (g *Game) validateActionsPhaseFinishTurn(c *gin.Context, cu *user.User) (*stats.Stats, error) {
	s, err := g.validateFinishTurn(c, cu)
	if err != nil {
		return nil, err
	}
	if g.Phase != actions {
		return nil, sn.NewVError(`Expected "Actions" phase but have %q phase.`, g.Phase)
	}
	return s, nil
}

func (client Client) electionPhaseFinishTurn(c *gin.Context, g *Game, cu *user.User) (*stats.Stats, contest.Contests, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	s, err := g.validateElectionPhaseFinishTurn(c, cu)
	if err != nil {
		return nil, nil, err
	}

	// validateElectionPhaseFinishTurn ensures cu != nil
	restful.AddNoticef(c, "%s finished turn.", cu.Name)

	cs, err := client.continueElections(c, g, cu)
	if err != nil {
		return nil, nil, err
	}

	return s, cs, nil
}

func (client Client) continueElections(c *gin.Context, g *Game, cu *user.User) (contest.Contests, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	// when true election phase is over
	if !g.electionsTillUnresolved(c, cu) {
		return nil, nil
	}

	g.startAwardChipsPhase(c)
	g.startScoreVictoryPointsPhase(c)
	g.newTurnOrder(c)

	return client.startCityOfficesPhase(c, g)
}

func (g *Game) validateElectionPhaseFinishTurn(c *gin.Context, cu *user.User) (*stats.Stats, error) {
	s, err := g.validateFinishTurn(c, cu)
	switch {
	case err != nil:
		return nil, err
	case g.Phase != elections:
		return nil, sn.NewVError(`Expected "Elections" phase but have %q phase.`, g.Phase)
	default:
		return s, nil
	}
}

func (g *Game) electionsTillUnresolved(c *gin.Context, cu *user.User) bool {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	for _, w := range g.ActiveWards() {
		if !w.Resolved {
			if g.CurrentWard() == w {
				if !g.resolve(c, cu, w) {
					return false
				}
			} else {
				if !g.startElectionIn(c, cu, w) {
					return false
				}
			}
		}
	}
	return true
}

func (client Client) placeImmigrantPhaseFinishTurn(c *gin.Context, g *Game, cu *user.User) (*stats.Stats, contest.Contests, error) {
	s, err := g.validatePlaceImmigrantPhaseFinishTurn(c, cu)
	if err != nil {
		return nil, nil, err
	}

	g.Phase = elections
	g.CurrentWard().Resolved = true
	cs, err := client.continueElections(c, g, cu)
	if err != nil {
		return nil, nil, err
	}
	return s, cs, nil
}

func (g *Game) validatePlaceImmigrantPhaseFinishTurn(c *gin.Context, cu *user.User) (*stats.Stats, error) {
	s, err := g.validateFinishTurn(c, cu)
	switch {
	case err != nil:
		return nil, err
	case g.Phase != placeImmigrant:
		return nil, sn.NewVError(`Expected "Place Immigrant" phase but have %q phase.`, g.Phase)
	default:
		return s, nil
	}
}

func (client Client) takeChipPhaseFinishTurn(c *gin.Context, g *Game, cu *user.User) (*stats.Stats, contest.Contests, error) {
	s, err := g.validateTakeChipPhaseFinishTurn(c, cu)
	if err != nil {
		return nil, nil, err
	}
	g.Phase = elections
	g.CurrentWard().Resolved = true
	cs, err := client.continueElections(c, g, cu)
	if err != nil {
		return nil, nil, err
	}
	return s, cs, nil
}

func (g *Game) validateTakeChipPhaseFinishTurn(c *gin.Context, cu *user.User) (*stats.Stats, error) {
	s, err := g.validateFinishTurn(c, cu)
	switch {
	case err != nil:
		return nil, err
	case g.Phase != takeFavorChip:
		return nil, sn.NewVError(`Expected "Take Favor Chip" phase but have %q phase.`, g.Phase)
	default:
		return s, nil
	}
}

func (g *Game) assignOfficesPhaseFinishTurn(c *gin.Context, cu *user.User) (*stats.Stats, error) {
	s, err := g.validateAssignOfficesPhaseFinishTurn(c, cu)
	switch {
	case err != nil:
		return nil, err
	default:
		restful.AddNoticef(c, "%s finished turn.", g.NameFor(g.CurrentPlayer()))

		g.startNextTerm()
		return s, nil
	}
}

func (g *Game) validateAssignOfficesPhaseFinishTurn(c *gin.Context, cu *user.User) (*stats.Stats, error) {
	s, err := g.validateFinishTurn(c, cu)
	switch {
	case err != nil:
		return nil, err
	case g.Phase != assignCityOffices:
		return nil, sn.NewVError(`Expected "Assign City Offices" phase but have %q phase.`, g.Phase)
	case !g.allPlayersHaveOffice():
		return nil, sn.NewVError("You must first assign all players an office")
	default:
		return s, nil
	}
}
