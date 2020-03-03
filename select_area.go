package tammany

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/sn"
	"github.com/gin-gonic/gin"
)

const (
	wardKey   = "ward"
	officeKey = "office"
)

func wardFrom(c *gin.Context) (w *Ward) {
	w, _ = c.Value(wardKey).(*Ward)
	return
}

func withWard(c *gin.Context, w *Ward) *gin.Context {
	c.Set(wardKey, w)
	return c
}

func officeFrom(c *gin.Context) (o office) {
	o, _ = c.Value(officeKey).(office)
	return
}

func withOffice(c *gin.Context, o office) *gin.Context {
	c.Set(officeKey, o)
	return c
}

func (g *Game) selectArea(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	if g.Phase == placeImmigrant {
		g.getWard(c)
		tmpl, act = "tammany/place_immigrant_dialog", game.None
	} else if w := g.getWard(c); w != nil {
		g.getWard(c)
		tmpl, act = "tammany/place_pieces_dialog", game.None
	} else if o := g.getOffice(c); o != noOffice {
		tmpl, act = "tammany/assign_office_dialog", game.None
	} else {
		tmpl, act, err = "tammany/flash_notice", game.None, sn.NewVError("Invalid area selected.")
	}
	return
}

func (g *Game) getWard(c *gin.Context) *Ward {
	id, ok := toWardID[c.PostForm("area")]
	if !ok {
		return nil
	}
	w := g.wardByID(id)
	withWard(c, w)
	return w
}

func (g *Game) getOffice(c *gin.Context) office {
	o, ok := toOffice[c.PostForm("area")]
	if !ok {
		return noOffice
	}
	withOffice(c, o)
	return o
}
