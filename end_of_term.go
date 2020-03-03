package tammany

import (
	"bytes"
	"encoding/gob"
	"html/template"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("*game.scoreVPEntry", new(scoreVPEntry))
}

type electionResults struct {
	PlayerResults playerResults
	MayorID       int
}

type playerResults map[int]*playerResult

type playerResult struct {
	WardIDS wardIDS
	Score   int
}

func (g *Game) startScoreVictoryPointsPhase(c *gin.Context) {
	g.Phase = scoreVictoryPoints
	g.scoreVictoryPoints(c)
}

func (g *Game) scoreVictoryPoints(c *gin.Context) {
	results := new(electionResults)
	results.PlayerResults = make(playerResults, len(g.Players()))
	results.MayorID = game.NoPlayerID
	for _, player := range g.Players() {
		results.PlayerResults[player.ID()] = new(playerResult)
	}

	for _, ward := range g.ActiveWards() {
		if player := g.winnerIn(ward); player != nil {
			points := 1
			if ward.ID == 14 {
				points = 2
			}
			presult := results.PlayerResults[player.ID()]
			presult.WardIDS = append(presult.WardIDS, ward.ID)
			presult.Score += points
			player.Score += points
		}
	}

	m := g.newMayor(results)
	if m != nil {
		for _, player := range g.Players() {
			player.Office = noOffice
		}
		m.Score += 3
		m.Office = mayor
		results.MayorID = m.ID()
		g.setCurrentPlayers(m)
	} else {
		g.setCurrentPlayers(g.playerByIndex(0))
	}

	e := g.newScoreVPEntry()
	e.ElectionResults = results
}

type scoreVPEntry struct {
	*Entry
	ElectionResults *electionResults
}

func (g *Game) newScoreVPEntry() (e *scoreVPEntry) {
	e = new(scoreVPEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return
}

//func (e *scoreVPEntry) Mayor() *Player {
//	return e.Game().(*Game).PlayerByID(e.ElectionResults.MayorID)
//}

func (e *scoreVPEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	ts := restful.TemplatesFrom(c)
	buf := new(bytes.Buffer)
	tmpl := ts["tammany/score_vp_entry"]
	if err := tmpl.Execute(buf, gin.H{
		"entry": e,
		"g":     g,
	}); err != nil {
		log.Errorf(err.Error())
		return ""
	}
	return restful.HTML(buf.String())
}

//	s :=
//		`       <div>
//                <ul>
//`
//	for pid, result := range e.ElectionResults.PlayerResults {
//		player := e.Game().PlayererByID(pid)
//		sids := make([]string, len(result.WardIDS))
//		name := player.Name()
//		for i, id := range result.WardIDS {
//			sids[i] = fmt.Sprintf("%d", id)
//		}
//		if len(sids) > 0 {
//			s += fmt.Sprintf(
//				`                       <li>
//                                %s scored %d points for winning wards %s.
//                        </li>
//`, name, result.Score, restful.ToSentence(sids))
//		} else {
//			s += fmt.Sprintf(
//				`                       <li>
//                                %s scored 0 points.
//                        </li>
//`, name)
//		}
//	}
//	s +=
//		`               </ul>
//`
//	if e.Mayor() != nil {
//		name := e.Mayor().Name()
//		s += fmt.Sprintf(
//			`               <div>
//                        %s scored 3 points for becoming mayor.
//                </div>
//        </div>
//`, name)
//	} else {
//		s +=
//			`               <div>
//                        No one became mayor.
//                </div>
//        </div>
//`
//	}
//	return template.HTML(s)
//}

func (g *Game) winnerIn(w *Ward) (player *Player) {
	for _, p := range g.Players() {
		if w.BossesFor(p) > 0 {
			player = p
			break
		}
	}
	return
}
