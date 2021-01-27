package tammany

import (
	"fmt"
	"strconv"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func (g *Game) bid(c *gin.Context, cu *user.User) (string, game.ActionType, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	err := g.validateBid(c, cu)
	if err != nil {
		return "tammany/flash_notice", game.None, err
	}

	cp := g.CurrentPlayerFor(cu)
	cp.PerformedAction = true
	cp.HasBid = true

	switch cp.PlayedChips.Count() {
	case 0:
		restful.AddNoticef(c, "You played no chips for the election in ward %d.", g.CurrentWardID)
	default:
		strings := []string{}
		for _, n := range g.Nationalities() {
			if cp.PlayedChips[n] > 0 {
				strings = append(strings, fmt.Sprintf("%d %s chips", cp.PlayedChips[n], n))
			}
		}
		restful.AddNoticef(c, "You played %s for the election in ward %d.", restful.ToSentence(strings), g.CurrentWardID)
	}
	return "tammany/bid_update", game.Cache, nil
}

func (g *Game) validateBid(c *gin.Context, cu *user.User) error {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if !g.IsCurrentPlayer(cu) {
		return sn.NewVError("Only the current player can place a bid.")
	}

	cp := g.CurrentPlayerFor(cu)
	if cp.PerformedAction {
		return sn.NewVError("You have already performed an action.")
	}

	for _, n := range g.Nationalities() {
		v := c.PostForm(fmt.Sprintf("%s-0", n.LString()))
		count, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		cp.PlayedChips[n] = count

		switch {
		case cp.PlayedChips[n] > 0 && g.CurrentWard().Immigrants[n] <= 0:
			return sn.NewVError("You played %s favour chips, but there are no %s immigrants in ward %d",
				n, n, g.CurrentWardID)
		case cp.PlayedChips[n] < 0:
			return sn.NewVError("Invalid value received for played %s chips.", n)
		case cp.PlayedChips[n] > cp.Chips[n]:
			return sn.NewVError("You played more %s chips, than you have.", n)
		}
	}
	return nil
}
