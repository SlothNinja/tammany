package tammany

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn"
	"github.com/gin-gonic/gin"
)

func (g *Game) cancelFinish(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validateCancelFinish(c); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
	} else {
		g.SubPhase = noSubPhase
		tmpl, act = "tammany/flash_notice", game.UndoPop
	}
	return
}

func (g *Game) validateCancelFinish(c *gin.Context) (err error) {
	switch {
	case !g.CUserIsCPlayerOrAdmin(c):
		err = sn.NewVError("Only the current player can take this action.")
	case !g.inActionPhase():
		err = sn.NewVError("Wrong phase for performing this action.")
	}
	return
}
