package tammany

import (
	"bytes"
	"encoding/gob"
	"html/template"

	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("*game.resolvedElectionEntry", new(resolvedElectionEntry))
	gob.RegisterName("*game.wonWardEntry", new(wonWardEntry))
}

func (g *Game) resolve(c *gin.Context, cu *user.User, w *Ward) (resolved bool) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		cnt    int
		winner *Player
	)

	cp := g.CurrentPlayerFor(cu)
	g.RemoveCurrentPlayers(cp)

	cds := g.candidates()
	for _, cd := range cds {
		if !cd.HasBid {
			return
		}
		switch ecnt := cd.electionCountIn(w); {
		case ecnt == cnt:
			winner = nil
		case ecnt > cnt:
			winner = cd
			cnt = ecnt
		}
	}

	// Create ActionLog Entry
	e := g.newResolvedElectionEntry(winner)
	e.WardID = w.ID

	// Make copy of bosses in ward for logs
	bosses := make(BossesMap, len(w.Bosses))
	for key, value := range w.Bosses {
		bosses[key] = value
	}
	e.Bosses = bosses

	// Remove Bosses
	for pid := range w.Bosses {
		if winner != nil && winner.ID() == pid {
			w.Bosses[pid] = 1
		} else {
			w.Bosses[pid] = 0
		}
	}

	playedChips := make(map[int]Chips)
	for _, player := range g.Players() {
		cs := Chips{}
		for nationality, cnt := range player.PlayedChips {
			cs[nationality] = player.PlayedChips[nationality]
			player.PlayedChips[nationality] = 0
			player.Chips[nationality] -= cnt
		}
		playedChips[player.ID()] = cs
	}

	e.PlayedChips = playedChips

	var contested bool
	if len(cds) > 1 {
		contested = true
	}
	e.Contested = contested

	// Proceed to Next Ward Election
	if winner != nil {
		e := g.newWonWardEntry(winner)
		e.WardID = w.ID

		switch w.ID {
		case 1, 2:
			g.Phase = placeImmigrant
			winner.wardElectionReset()
			g.setCurrentPlayers(winner)
		case 4, 7:
			g.Phase = takeFavorChip
			winner.wardElectionReset()
			g.setCurrentPlayers(winner)
		default:
			resolved, w.Resolved = true, true
		}
	} else {
		resolved, w.Resolved = true, true
	}
	return
}

type resolvedElectionEntry struct {
	*Entry
	WardID      wardID
	Bosses      BossesMap
	PlayedChips map[int]Chips
	Contested   bool
}

func (g *Game) newResolvedElectionEntry(p *Player) (e *resolvedElectionEntry) {
	e = new(resolvedElectionEntry)
	if p == nil {
		e.Entry = g.newEntry()
		e.PlayerID = noPlayerID
	} else {
		e.Entry = g.newEntryFor(p)
	}
	g.Log = append(g.Log, e)
	return
}

type wonWardEntry struct {
	*Entry
	WardID wardID
}

func (g *Game) newWonWardEntry(p *Player) (e *wonWardEntry) {
	e = new(wonWardEntry)
	e.Entry = g.newEntryFor(p)
	p.Log = append(p.Log, e)
	return
}

func (e *wonWardEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s won the election in ward %d.", g.NameByPID(e.PlayerID), e.WardID)
}

func (g *Game) startElectionIn(c *gin.Context, cu *user.User, w *Ward) (resolved bool) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	g.setCurrentWard(w)
	g.setCurrentPlayers()
	var bs, cds Players
	switch bs, cds = g.initCandidatesFor(w); {
	case len(bs) == 0:
	case len(cds) == 0:
	case len(cds) == 1:
		cds[0].HasBid = true
	case len(bs) == 1:
		b := bs[0]
		cMax := nonBidderMaxIn(w, b, cds)

		// All other candidates have fixed/known influence in ward.
		// Check to see if player must play chips to win or force tie in ward.
		if w.BossesFor(b) <= cMax && b.MaxInfluenceIn(w) >= cMax {
			g.setCurrentPlayers(b)
		} else {
			b.HasBid = true
		}
	default:
		g.setCurrentPlayers(bs...)
	}

	if len(g.CurrentPlayers()) == 0 {
		if resolved = g.resolve(c, cu, w); resolved {
			g.setCurrentWard(nil)
		}
	}
	return
}

func nonBidderMaxIn(w *Ward, b *Player, cds Players) (cMax int) {
	for _, cd := range cds {
		if !cd.Equal(b) {
			if cBosses := w.BossesFor(cd); cBosses > cMax {
				cMax = cBosses
			}
		}
	}
	return
}

func (g *Game) initCandidatesFor(w *Ward) (bs, cds Players) {
	for _, p := range g.Players() {
		p.wardElectionReset()
		if w.BossesFor(p) > 0 {
			cds = append(cds, p)
			p.Candidate = true
			if w.playableChipsFor(p) > 0 {
				bs = append(bs, p)
			} else {
				p.HasBid = true
			}
		}
	}
	return
}

func (w *Ward) playableChipsFor(player *Player) (chips int) {
	for nationality, count := range w.Immigrants {
		if count > 0 {
			chips += player.ChipsFor(nationality)
		}
	}
	return
}

func (e *resolvedElectionEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	ts := restful.TemplatesFrom(c)
	buf := new(bytes.Buffer)
	tmpl := ts["tammany/resolved_election_entry"]
	if err := tmpl.Execute(buf, gin.H{
		"entry": e,
		"g":     g,
		"ctx":   c,
	}); err != nil {
		return ""
	}
	return restful.HTML(buf.String())
}
