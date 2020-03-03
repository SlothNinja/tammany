package tammany

import (
	"bytes"
	"encoding/gob"
	"html/template"

	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("*game.awardChipsEntry", new(awardChipsEntry))
}

func (g *Game) startAwardChipsPhase(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Entering")

	g.Phase = awardFavorChips
	g.awardChips()
}

func (g *Game) awardChips() {
	nationalities := g.Nationalities()
	awardChips := make(map[int]Chips, len(g.Players()))

	for _, player := range g.Players() {
		awardChips[player.ID()] = Chips{}
	}

	for _, nationality := range nationalities {
		winners := g.awardChipsFor(nationality)
		for _, winner := range winners {
			awardChips[winner.ID()][nationality] = 3
		}
	}
	e := g.newAwardChipsEntry()
	e.ChipWinners = awardChips
}

type awardChipsEntry struct {
	*Entry
	ChipWinners map[int]Chips
}

func (g *Game) newAwardChipsEntry() (e *awardChipsEntry) {
	e = new(awardChipsEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return
}

func (e *awardChipsEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	ts := restful.TemplatesFrom(c)
	buf := new(bytes.Buffer)
	tmpl := ts["tammany/award_chips_entry"]
	if err := tmpl.Execute(buf, gin.H{
		"entry": e,
		"g":     g,
		"ctx":   c,
	}); err != nil {
		return ""
	}
	return restful.HTML(buf.String())
}

func (g *Game) awardChipsFor(n nationality) (winners Players) {
	winners = g.chipWinners(n)
	for _, player := range winners {
		player.Chips[n] += 3
	}
	return
}

func (g *Game) chipWinners(n nationality) (winners Players) {
	var max int
	winners = make(Players, 0)
	for _, player := range g.Players() {
		controlled := g.ControlledBy(player, n)
		switch {
		case controlled > max:
			max = controlled
			winners = Players{player}
		case controlled == max:
			winners = append(winners, player)
		}
	}
	return
}
