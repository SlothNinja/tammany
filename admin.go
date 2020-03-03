package tammany

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func (g *Game) adminState(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	h := game.NewHeader(c, nil, 0)
	if err := restful.BindWith(c, h, binding.FormPost); err != nil {
		return "", game.None, err
	}

	g.UserIDS = h.UserIDS
	g.Title = h.Title
	g.Phase = h.Phase
	g.Round = h.Round
	g.NumPlayers = h.NumPlayers
	g.Password = h.Password
	g.CreatorID = h.CreatorID
	if !(len(h.CPUserIndices) == 1 && h.CPUserIndices[0] == -1) {
		g.CPUserIndices = h.CPUserIndices
	}
	if !(len(h.WinnerIDS) == 1 && h.WinnerIDS[0] == -1) {
		g.WinnerIDS = h.WinnerIDS
	}
	g.Status = h.Status
	return "", game.Save, nil
}

type chips struct {
	Chips       []int `form:"chips"`
	PlayedChips []int `form:"played-chips"`
}

func newChips() *chips {
	return &chips{
		Chips:       make([]int, 4),
		PlayedChips: make([]int, 4),
	}
}

func (g *Game) adminPlayer(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	p := newPlayer()
	chips := newChips()
	if err := restful.BindWith(c, p.Player, binding.FormPost); err != nil {
		return "", game.None, err
	}

	if err := restful.BindWith(c, p, binding.FormPost); err != nil {
		return "", game.None, err
	}

	if err := restful.BindWith(c, chips, binding.FormPost); err != nil {
		return "", game.None, err
	}

	p2 := g.PlayerByID(p.ID())

	for i, n := range g.Nationalities() {
		p2.Chips[n] = chips.Chips[i]
		p2.PlayedChips[n] = chips.PlayedChips[i]
	}

	p2.Score = p.Score
	p2.PerformedAction = p.PerformedAction
	p2.Candidate = p.Candidate
	p2.UsedOffice = p.UsedOffice
	p2.PlacedBosses = p.PlacedBosses
	p2.PlacedImmigrants = p.PlacedImmigrants
	p2.HasBid = p.HasBid

	return "", game.Save, nil
}

func (g *Game) adminWard(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var w2 struct {
		ID       wardID `form:"ward-id"`
		Irish    int    `form:"Irish"`
		English  int    `form:"English"`
		German   int    `form:"German"`
		Italian  int    `form:"Italian"`
		Bosses   []int  `form:"bosses"`
		Resolved bool   `form:"resolved"`
		LockedUp bool   `form:"lockedup"`
	}

	if err := restful.BindWith(c, &w2, binding.FormPost); err != nil {
		return "", game.None, err
	}

	w1 := g.wardByID(w2.ID)
	w1.Immigrants[irish] = w2.Irish
	w1.Immigrants[german] = w2.German
	w1.Immigrants[italian] = w2.Italian
	w1.Immigrants[english] = w2.English

	for i, p := range g.Players() {
		w1.Bosses[p.ID()] = w2.Bosses[i]
	}

	w1.Resolved = w2.Resolved
	w1.LockedUp = w2.LockedUp

	return "", game.Save, nil
}

func (g *Game) adminCastleGarden(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var cg struct {
		Irish   int `form:"Irish"`
		English int `form:"English"`
		German  int `form:"German"`
		Italian int `form:"Italian"`
	}

	if err := restful.BindWith(c, &cg, binding.FormPost); err != nil {
		return "", game.None, err
	}

	g.CastleGarden[irish] = cg.Irish
	g.CastleGarden[german] = cg.German
	g.CastleGarden[italian] = cg.Italian
	g.CastleGarden[english] = cg.English

	return "", game.Save, nil
}

func (g *Game) adminImmigrantBag(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var cg struct {
		Irish   int `form:"Irish"`
		English int `form:"English"`
		German  int `form:"German"`
		Italian int `form:"Italian"`
	}

	if err := restful.BindWith(c, &cg, binding.FormPost); err != nil {
		return "", game.None, err
	}

	g.Bag[irish] = cg.Irish
	g.Bag[german] = cg.German
	g.Bag[italian] = cg.Italian
	g.Bag[english] = cg.English

	return "", game.Save, nil
}
