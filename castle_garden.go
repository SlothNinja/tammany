package tammany

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("*game.castleGardenEntry", new(castleGardenEntry))
}

func defaultBag() Nationals {
	return Nationals{irish: 19, english: 20, german: 21, italian: 25}
}

func (g *Game) fillGardenFor(n int) (filled bool) {
	if g.CastleGarden.empty() {
		for i := 0; i < n+2; i++ {
			g.CastleGarden[g.Bag.draw()]++
		}
		filled = true
	}
	return
}

func (g *Game) emptyGarden() {
	for n, count := range g.CastleGarden {
		g.CastleGarden[n] = 0
		g.Bag[n] += count
	}
}

func (g *Game) castleGardenPhase() {
	g.Phase = castleGarden
	cp := g.CurrentPlayer()
	entry := g.newCastleGardenEntry(cp)
	if g.fillGardenFor(g.NumPlayers) {
		entry.Filled = true
		for nationality, count := range g.CastleGarden {
			entry.Immigrants[nationality] = count
		}
	}
	g.actionsPhase()
}

type castleGardenEntry struct {
	*Entry
	Filled     bool
	Immigrants Nationals
}

func (g *Game) newCastleGardenEntry(p *Player) *castleGardenEntry {
	e := new(castleGardenEntry)
	e.Entry = g.newEntryFor(p)
	e.Immigrants = defaultNationals()
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *castleGardenEntry) HTML(c *gin.Context, g *Game, cu *user.User) template.HTML {
	n := g.NameByPID(e.PlayerID)
	if !e.Filled {
		return restful.HTML("%s placed no immigrants in the Castle Garden.", n)
	}
	segments := []string{}
	for nationality := range defaultNationals() {
		count := e.Immigrants[nationality]
		if count != 0 {
			segments = append(segments, fmt.Sprintf("%d %s", count, nationality))
		}
	}
	return restful.HTML("%s placed %s immigrants in the Castle Garden.", n, restful.ToSentence(segments))
}
