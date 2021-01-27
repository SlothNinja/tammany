package tammany

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func (g *Game) cancelFinish(c *gin.Context, cu *user.User) (string, game.ActionType, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	err := g.validateCancelFinish(c, cu)
	if err != nil {
		return "tammany/flash_notice", game.None, err
	}

	g.SubPhase = noSubPhase
	return "tammany/flash_notice", game.UndoPop, nil
}

func (g *Game) validateCancelFinish(c *gin.Context, cu *user.User) error {
	switch {
	case !g.IsCurrentPlayer(cu):
		return sn.NewVError("Only the current player can take this action.")
	case !g.inActionPhase():
		return sn.NewVError("Wrong phase for performing this action.")
	default:
		return nil
	}
}
