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

func (g *Game) bid(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validateBid(c); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	cu := user.CurrentFrom(c)
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
	tmpl, act = "tammany/bid_update", game.Cache
	return
}

func (g *Game) validateBid(c *gin.Context) (err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		count int
		v     string
	)

	if !g.CUserIsCPlayerOrAdmin(c) {
		err = sn.NewVError("Only the current player can place a bid.")
		return
	}

	cu := user.CurrentFrom(c)
	cp := g.CurrentPlayerFor(cu)
	if cp.PerformedAction {
		err = sn.NewVError("You have already performed an action.")
		return
	}

	for _, n := range g.Nationalities() {
		v = c.PostForm(fmt.Sprintf("%s-0", n.LString()))
		if count, err = strconv.Atoi(v); err != nil {
			return
		}
		cp.PlayedChips[n] = count

		switch {
		case cp.PlayedChips[n] > 0 && g.CurrentWard().Immigrants[n] <= 0:
			err = sn.NewVError("You played %s favour chips, but there are no %s immigrants in ward %d",
				n, n, g.CurrentWardID)
			return
		case cp.PlayedChips[n] < 0:
			err = sn.NewVError("Invalid value received for played %s chips.", n)
			return
		case cp.PlayedChips[n] > cp.Chips[n]:
			err = sn.NewVError("You played more %s chips, than you have.", n)
			return
		}
	}
	return
}
