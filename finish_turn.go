package tammany

import (
	"net/http"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
)

func (client Client) finish(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := gameFrom(c)
		oldCP := g.CurrentPlayer()

		var (
			s   *stats.Stats
			cs  contest.Contests
			err error
		)

		switch g.Phase {
		case actions:
			s, cs, err = client.actionsPhaseFinishTurn(c, g)
		case placeImmigrant:
			s, cs, err = client.placeImmigrantPhaseFinishTurn(c, g)
		case takeFavorChip:
			s, cs, err = client.takeChipPhaseFinishTurn(c, g)
		case elections:
			s, cs, err = client.electionPhaseFinishTurn(c, g)
		case assignCityOffices:
			s, err = g.assignOfficesPhaseFinishTurn(c)
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
			err = client.saveWith(c, g, ks, es)
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
		err = client.saveWith(c, g, []*datastore.Key{s.Key}, []interface{}{s})
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

func (g *Game) validateFinishTurn(c *gin.Context) (s *stats.Stats, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var cp *Player

	switch cp, s = g.CurrentPlayer(), stats.Fetched(c); {
	case s == nil:
		err = sn.NewVError("missing stats for player.")
	case !g.CUserIsCPlayerOrAdmin(c):
		err = sn.NewVError("Only the current player may finish a turn.")
	case !cp.PerformedAction:
		err = sn.NewVError("%s has yet to perform an action.", g.NameFor(cp))
	case g.ImmigrantInTransit != noNationality:
		err = sn.NewVError("You must complete move of %s immigrant before finishing turn.", g.ImmigrantInTransit)
	}
	return
}

// ps is an optional parameter.
// If no player is provided, assume current player.
func (g *Game) nextPlayer(ps ...game.Playerer) *Player {
	if nper := g.NextPlayerer(ps...); nper != nil {
		return nper.(*Player)
	}
	return nil
}

func (client Client) actionsPhaseFinishTurn(c *gin.Context, g *Game) (*stats.Stats, contest.Contests, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	s, err := g.validateActionsPhaseFinishTurn(c)
	if err != nil {
		return nil, nil, err
	}

	cp := g.CurrentPlayer()
	if g.CanUseOffice(cp) && c.PostForm("action") != "confirm-finish" {
		g.SubPhase = officeWarning
		client.Cache.SetDefault(g.UndoKey(c), g)
		return s, nil, nil
	}

	restful.AddNoticef(c, "%s finished turn.", g.NameFor(cp))

	np := g.nextPlayer()
	g.beginningOfTurnResetFor(np)
	g.setCurrentPlayers(np)

	if game.IndexFor(np, g.Playerers) == 0 {
		switch g.Year() {
		case 4, 8, 12, 16:
			cs, err := client.startElections(c, g)
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

func (g *Game) validateActionsPhaseFinishTurn(c *gin.Context) (*stats.Stats, error) {
	s, err := g.validateFinishTurn(c)
	if err != nil {
		return nil, err
	}
	if g.Phase != actions {
		return nil, sn.NewVError(`Expected "Actions" phase but have %q phase.`, g.Phase)
	}
	return s, nil
}

func (client Client) electionPhaseFinishTurn(c *gin.Context, g *Game) (*stats.Stats, contest.Contests, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	s, err := g.validateElectionPhaseFinishTurn(c)
	if err != nil {
		return nil, nil, err
	}

	restful.AddNoticef(c, "%s finished turn.", g.NameFor(g.CurrentPlayer()))

	cs, err := client.continueElections(c, g)
	if err != nil {
		return nil, nil, err
	}

	return s, cs, nil
}

func (client Client) continueElections(c *gin.Context, g *Game) (contest.Contests, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	// when true election phase is over
	if !g.electionsTillUnresolved(c) {
		return nil, nil
	}

	g.startAwardChipsPhase(c)
	g.startScoreVictoryPointsPhase(c)
	g.newTurnOrder(c)

	return client.startCityOfficesPhase(c, g)
}

func (g *Game) validateElectionPhaseFinishTurn(c *gin.Context) (s *stats.Stats, err error) {
	if s, err = g.validateFinishTurn(c); g.Phase != elections {
		err = sn.NewVError(`Expected "Elections" phase but have %q phase.`, g.Phase)
	}
	return
}

func (g *Game) electionsTillUnresolved(c *gin.Context) (done bool) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	for _, w := range g.ActiveWards() {
		if !w.Resolved {
			if g.CurrentWard() == w {
				if !g.resolve(c, w) {
					return
				}
			} else {
				if !g.startElectionIn(c, w) {
					return
				}
			}
		}
	}
	done = true
	return
}

func (client Client) placeImmigrantPhaseFinishTurn(c *gin.Context, g *Game) (*stats.Stats, contest.Contests, error) {
	s, err := g.validatePlaceImmigrantPhaseFinishTurn(c)
	if err != nil {
		return nil, nil, err
	}

	g.Phase = elections
	g.CurrentWard().Resolved = true
	cs, err := client.continueElections(c, g)
	if err != nil {
		return nil, nil, err
	}
	return s, cs, nil
}

func (g *Game) validatePlaceImmigrantPhaseFinishTurn(c *gin.Context) (s *stats.Stats, err error) {
	if s, err = g.validateFinishTurn(c); g.Phase != placeImmigrant {
		err = sn.NewVError(`Expected "Place Immigrant" phase but have %q phase.`, g.Phase)
	}
	return
}

func (client Client) takeChipPhaseFinishTurn(c *gin.Context, g *Game) (*stats.Stats, contest.Contests, error) {
	s, err := g.validateTakeChipPhaseFinishTurn(c)
	if err != nil {
		return nil, nil, err
	}
	g.Phase = elections
	g.CurrentWard().Resolved = true
	cs, err := client.continueElections(c, g)
	if err != nil {
		return nil, nil, err
	}
	return s, cs, nil
}

func (g *Game) validateTakeChipPhaseFinishTurn(c *gin.Context) (s *stats.Stats, err error) {
	if s, err = g.validateFinishTurn(c); g.Phase != takeFavorChip {
		err = sn.NewVError(`Expected "Take Favor Chip" phase but have %q phase.`, g.Phase)
	}
	return
}

func (g *Game) assignOfficesPhaseFinishTurn(c *gin.Context) (s *stats.Stats, err error) {
	if s, err = g.validateAssignOfficesPhaseFinishTurn(c); err == nil {
		restful.AddNoticef(c, "%s finished turn.", g.NameFor(g.CurrentPlayer()))

		g.startNextTerm()
	}
	return
}

func (g *Game) validateAssignOfficesPhaseFinishTurn(c *gin.Context) (s *stats.Stats, err error) {
	switch s, err = g.validateFinishTurn(c); {
	case err != nil:
	case g.Phase != assignCityOffices:
		err = sn.NewVError(`Expected "Assign City Offices" phase but have %q phase.`, g.Phase)
	case !g.allPlayersHaveOffice():
		err = sn.NewVError("You must first assign all players an office")
	}
	return
}
