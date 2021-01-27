package tammany

import (
	"time"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/gin-gonic/gin"
)

func (g *Game) adminState(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	h := struct {
		Title         string           `form:"title"`
		Turn          int              `form:"turn"" binding:"min=0"`
		Phase         game.Phase       `form:"phase" binding:"min=0"`
		SubPhase      game.SubPhase    `form:"sub-phase" binding:"min=0"`
		Round         int              `form:"round" binding:"min=0"`
		NumPlayers    int              `form:"num-players" binding"min=0,max=5"`
		Password      string           `form:"password"`
		CreatorID     int64            `form:"creator-id"`
		CreatorSID    string           `form:"creator-sid"`
		CreatorName   string           `form:"creator-name"`
		UserIDS       []int64          `form:"user-ids"`
		UserSIDS      []string         `form:"user-sids"`
		UserNames     []string         `form:"user-names"`
		UserEmails    []string         `form:"user-emails"`
		OrderIDS      game.UserIndices `form:"order-ids"`
		CPUserIndices game.UserIndices `form:"cp-user-indices"`
		WinnerIDS     game.UserIndices `form:"winner-ids"`
		Status        game.Status      `form:"status"`
		Progress      string           `form:"progress"`
		Options       []string         `form:"options"`
		OptString     string           `form:"opt-string"`
		CreatedAt     time.Time        `form:"created-at"`
		UpdatedAt     time.Time        `form:"updated-at"`
	}{}

	err := c.ShouldBind(&h)
	if err != nil {
		return "", game.None, err
	}

	// h := game.NewHeader(c, nil, 0)
	// if err := restful.BindWith(c, h, binding.FormPost); err != nil {
	// 	return "", game.None, err
	// }

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
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)
	obj := struct {
		PlacedBosses     int   `form:"placed-bosses"`
		PlacedImmigrants int   `form:"placed-immigrants"`
		LockedUp         int   `form:"lockedup"`
		Slandered        int   `form:"slandered"`
		Candidate        bool  `form:"candidate"`
		HasBid           bool  `form:"has-bid"`
		UsedOffice       bool  `form:"used-office"`
		IDF              int   `form:"idf"`
		PerformedAction  bool  `form:"performed-action"`
		Score            int   `form:"score"`
		Passed           bool  `form:"passed"`
		Chips            []int `form:"chips"`
		PlayedChips      []int `form:"played-chips"`
	}{}

	err := c.ShouldBind(&obj)
	if err != nil {
		return "", game.None, err
	}

	p2 := g.PlayerByID(obj.IDF)

	for i, n := range g.Nationalities() {
		p2.Chips[n] = obj.Chips[i]
		p2.PlayedChips[n] = obj.PlayedChips[i]
	}

	p2.Score = obj.Score
	p2.PerformedAction = obj.PerformedAction
	p2.Candidate = obj.Candidate
	p2.UsedOffice = obj.UsedOffice
	p2.PlacedBosses = obj.PlacedBosses
	p2.PlacedImmigrants = obj.PlacedImmigrants
	p2.HasBid = obj.HasBid

	return "", game.Save, nil
}

func (g *Game) adminWard(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	w2 := struct {
		ID       wardID `form:"ward-id"`
		Irish    int    `form:"Irish"`
		English  int    `form:"English"`
		German   int    `form:"German"`
		Italian  int    `form:"Italian"`
		Bosses   []int  `form:"bosses"`
		Resolved bool   `form:"resolved"`
		LockedUp bool   `form:"lockedup"`
	}{}

	err := c.ShouldBind(&w2)
	if err != nil {
		return "", game.None, err
	}

	// if err := restful.BindWith(c, &w2, binding.FormPost); err != nil {
	// 	return "", game.None, err
	// }

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
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	cg := struct {
		Irish   int `form:"Irish"`
		English int `form:"English"`
		German  int `form:"German"`
		Italian int `form:"Italian"`
	}{}

	err := c.ShouldBind(&cg)
	if err != nil {
		return "", game.None, err
	}

	// if err := restful.BindWith(c, &cg, binding.FormPost); err != nil {
	// 	return "", game.None, err
	// }

	g.CastleGarden[irish] = cg.Irish
	g.CastleGarden[german] = cg.German
	g.CastleGarden[italian] = cg.Italian
	g.CastleGarden[english] = cg.English

	return "", game.Save, nil
}

func (g *Game) adminImmigrantBag(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	cg := struct {
		Irish   int `form:"Irish"`
		English int `form:"English"`
		German  int `form:"German"`
		Italian int `form:"Italian"`
	}{}

	err := c.ShouldBind(&cg)
	if err != nil {
		return "", game.None, err
	}

	// if err := restful.BindWith(c, &cg, binding.FormPost); err != nil {
	// 	return "", game.None, err
	// }

	g.Bag[irish] = cg.Irish
	g.Bag[german] = cg.German
	g.Bag[italian] = cg.Italian
	g.Bag[english] = cg.English

	return "", game.Save, nil
}
