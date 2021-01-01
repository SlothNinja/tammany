package tammany

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"strconv"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(placedPiecesEntry))
	gob.RegisterName("*game.placedBossesEntry", new(placedBossesEntry))
	gob.RegisterName("*game.placedBossEntry", new(placedBossEntry))
	gob.RegisterName("*game.placedImmigrantEntry", new(placedImmigrantEntry))
	gob.RegisterName("*game.removedImmigrantEntry", new(removedImmigrantEntry))
	gob.RegisterName("*game.placedBossAndImmigrantEntry", new(placedBossAndImmigrantEntry))
	gob.RegisterName("*game.takeChipEntry", new(takeChipEntry))
}

func (g *Game) placePieces(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		b      int
		n, cnt nationality
		w      *Ward
	)

	if b, n, cnt, w, err = g.validatePlacePieces(c, cu); err != nil {
		log.Errorf(err.Error())
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	// Log Placement
	cp := g.CurrentPlayer()
	e := g.newPlacedPiecesEntryFor(cp, b, n, cnt, w)
	restful.AddNoticef(c, string(e.HTML(c)))

	g.endSlander(cu)

	// Place Bosses
	w.Bosses[cp.ID()] += b
	cp.PlacedBosses += b

	// Place Immigrant
	if n != noNationality {
		w.Immigrants[n]++
		cp.PlacedImmigrants++

		// Remove from Castle Garden and Take Favor Chip
		if g.Phase == actions {
			g.CastleGarden[n]--
			cp.Chips[n]++
		}
	}

	// Performed Action
	if g.Phase == actions && cp.placedPieces() >= 2 {
		cp.PerformedAction = true
	} else if g.Phase == placeImmigrant && cp.placedPieces() >= 1 {
		cp.PerformedAction = true
	}

	tmpl, act = "tammany/place_pieces", game.Cache
	return
}

func (g *Game) validatePlacePieces(c *gin.Context, cu *user.User) (b int, n nationality, cb nationality, w *Ward, err error) {
	if b, err = getBosses(c); err != nil {
		log.Errorf(err.Error())
		return
	}

	n = getNationality(c)
	w = g.getWard(c)
	cp := g.CurrentPlayer()

	count := b
	if n != noNationality {
		count++
	}

	switch {
	// General Checks
	case w == nil:
		err = sn.NewVError("You must first select a ward.")
	case w.LockedUp:
		err = sn.NewVError("You can't place pieces into a locked ward.")
	case cp.PerformedAction:
		err = sn.NewVError("You have already performed an action.")
	// Phase Related Checks
	case g.Phase == actions:
		switch {
		case b < 0, b > 2:
			err = sn.NewVError("You cannot place %d bosses.", b)
		case count < 1, count > 2:
			err = sn.NewVError("You cannot place %d pieces.", count)
		case count+cp.placedPieces() > 2:
			err = sn.NewVError("You already placed %d pieces.  You cannot place %d more pieces.", cp.placedPieces(), count)
		case n != noNationality && g.CastleGarden[n] < 1:
			err = sn.NewVError("There is not a %s immigrant in the Castle Garden", n)
		case n != noNationality && cp.PlacedImmigrants >= 1:
			err = sn.NewVError("You already placed %d immigrants.  You cannot place another immigrant.", cp.PlacedImmigrants)
		default:
			cb = n
		}
	case g.Phase == placeImmigrant:
		switch {
		case b != 0:
			err = sn.NewVError("You cannot place a boss.")
		case count != 1:
			err = sn.NewVError("You must place 1 immigrant.")
		case n == noNationality:
			err = sn.NewVError("You selected an invalid nationality.")
		}
	default:
		err = sn.NewVError("Wrong phase for performing this action.")
	}
	log.Errorf("err: %v", err)
	return
}

type placedPiecesEntry struct {
	*Entry
	Bosses    int
	Immigrant nationality
	Chip      nationality
	WardID    wardID
}

func (g *Game) newPlacedPiecesEntryFor(p *Player, b int, n, c nationality, w *Ward) (e *placedPiecesEntry) {
	e = &placedPiecesEntry{
		Entry:     g.newEntryFor(p),
		Bosses:    b,
		Immigrant: n,
		Chip:      c,
		WardID:    w.ID,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *placedPiecesEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	var ss []string
	if e.Bosses > 0 {
		ss = append(ss, fmt.Sprintf("placed %d %s in ward %d",
			e.Bosses, restful.Pluralize("boss", e.Bosses), e.WardID))
	}
	if e.Immigrant != noNationality {
		ss = append(ss, fmt.Sprintf("placed 1 %s immigrant in ward %d", e.Immigrant, e.WardID))
	}
	if e.Chip != noNationality {
		ss = append(ss, fmt.Sprintf("received 1 %s favor", e.Chip))
	}
	return restful.HTML("%s %s.", g.NameByPID(e.PlayerID), restful.ToSentence(ss))
}

func getBosses(c *gin.Context) (b int, err error) {
	if v := c.PostForm("bosses"); v != "" {
		b, err = strconv.Atoi(v)
	}
	return
}

func getNationality(c *gin.Context) (n nationality) {
	n, _ = toNationality[c.PostForm("immigrant")]
	return
}

// Legacy Log Entries
type placedBossesEntry struct {
	*Entry
	WardID wardID
}

func (g *Game) newPlacedBossesEntryFor(p *Player, ward *Ward) (e *placedBossesEntry) {
	e = new(placedBossesEntry)
	e.Entry = g.newEntryFor(p)
	e.WardID = ward.ID
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *placedBossesEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s placed two bosses in ward %d.", g.NameByPID(e.PlayerID), e.WardID)
}

func (p *Player) hasPlacedOnePiece() bool {
	return p.placedPieces() == 1
}

type placedBossEntry struct {
	*Entry
	WardID wardID
}

func (g *Game) newPlacedBossEntryFor(p *Player, w *Ward) (e *placedBossEntry) {
	e = new(placedBossEntry)
	e.Entry = g.newEntryFor(p)
	e.WardID = w.ID
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *placedBossEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s placed a boss in ward %d.", g.NameByPID(e.PlayerID), e.WardID)
}

func (g *Game) removeImmigrant(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	var (
		w *Ward
		n nationality
	)

	if w, n, err = g.validateRemoveImmigrant(c, cu); err != nil {
		log.Errorf(err.Error())
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	// Log Placement
	cp := g.CurrentPlayer()
	e := g.newRemovedImmigrantEntryFor(cp, w, n)
	restful.AddNoticef(c, string(e.HTML(c)))

	// Remove Immigrant
	g.endSlander(cu)
	w.Immigrants[n]--
	g.Bag[n]++
	cp.UsedOffice = true

	tmpl, act = "tammany/place_pieces", game.Cache
	log.Errorf("err: %v", err)
	return
}

type removedImmigrantEntry struct {
	*Entry
	WardID    wardID
	Immigrant nationality
}

func (g *Game) newRemovedImmigrantEntryFor(p *Player, w *Ward, n nationality) (e *removedImmigrantEntry) {
	e = new(removedImmigrantEntry)
	e.Entry = g.newEntryFor(p)
	e.WardID = w.ID
	e.Immigrant = n
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *removedImmigrantEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s removed a %s immigrant from ward %d.", g.NameByPID(e.PlayerID), e.Immigrant, e.WardID)
}

func (g *Game) validateRemoveImmigrant(c *gin.Context, cu *user.User) (*Ward, nationality, error) {
	n := getNationality(c)
	w := g.getWard(c)
	cp := g.CurrentPlayer()
	chief := g.chiefOfPolice()

	switch {
	case !g.IsCurrentPlayer(cu):
		return nil, noNationality, sn.NewVError("Only the current player can remove an immigrant from a ward.")
	case w == nil:
		return nil, noNationality, sn.NewVError("You must first select a ward.")
	case w.LockedUp:
		return nil, noNationality, sn.NewVError("You can't remove an immigrant from a locked ward.")
	case cp.placedPieces() == 1:
		return nil, noNationality, sn.NewVError("You must remove an immigrant before or after the placing pieces action, not during.")
	case !g.inActionPhase():
		return nil, noNationality, sn.NewVError("You can't remove an immigrant during the %s phase.", g.Phase)
	case w.hasOneImmigrant():
		return nil, noNationality, sn.NewVError("You can't remove the last immigrant from the ward.")
	case cp.NotEqual(chief):
		return nil, noNationality, sn.NewVError("You are the %s.  Only the Chief of Police can remove an immigrant from the ward.", cp.Office)
	}
	return w, n, nil
}

func (w *Ward) hasImmigrants() bool {
	return w.Immigrants.count() > 0
}

type placedImmigrantEntry struct {
	*Entry
	WardID    wardID
	Immigrant nationality
}

func (g *Game) newPlacedImmigrantEntryFor(p *Player, w *Ward, n nationality) (e *placedImmigrantEntry) {
	e = new(placedImmigrantEntry)
	e.Entry = g.newEntryFor(p)
	e.WardID = w.ID
	e.Immigrant = n
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *placedImmigrantEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s placed a %s immigrant in ward %d.", g.NameByPID(e.PlayerID), e.Immigrant, e.WardID)
}

func (p *Player) hasPlacedImmigrants() bool {
	return p.PlacedImmigrants > 0
}

type placedBossAndImmigrantEntry struct {
	*Entry
	WardID    wardID
	Immigrant nationality
}

func (g *Game) newPlacedBossAndImmigrantEntryFor(p *Player, w *Ward, n nationality) (e *placedBossAndImmigrantEntry) {
	e = new(placedBossAndImmigrantEntry)
	e.Entry = g.newEntryFor(p)
	e.WardID = w.ID
	e.Immigrant = n
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *placedBossAndImmigrantEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s placed a boss and a %s immigrant in ward %d.", g.NameByPID(e.PlayerID), e.Immigrant, e.WardID)
}

func (g *Game) deputyTakeChip(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	var n nationality
	if n, err = g.validateDeputyTakeChip(c, cu); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()

	// Take Favor Chip
	g.endSlander(cu)
	cp.Chips[n]++
	cp.UsedOffice = true

	// Log Placement
	e := g.newTakeChipEntryFor(cp, n)
	restful.AddNoticef(c, string(e.HTML(c)))

	tmpl, act = "tammany/place_pieces", game.Cache
	return
}

func (g *Game) validateDeputyTakeChip(c *gin.Context, cu *user.User) (n nationality, err error) {
	var ok bool
	if n, ok = toNationality[c.PostForm("chip")]; !ok {
		err = sn.NewVError("Invalid value received for chip nationatlity.")
		return
	}

	cp := g.CurrentPlayer()
	deputy := g.deputyMayor()

	switch {
	case !g.IsCurrentPlayer(cu):
		err = sn.NewVError("Only the current player can take a chip.")
	case cp.UsedOffice:
		err = sn.NewVError("You have already taken a favor chip.")
	case !g.inActionPhase():
		err = sn.NewVError("You can't take a favour chip in phase %q.", g.PhaseName())
	case cp.NotEqual(deputy):
		err = sn.NewVError("You are the %s.  Only the Deputy Mayor may take a favor chip.", cp.Office)
	}
	return
}

func (g *Game) takeChip(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	var n nationality
	if n, err = g.validateTakeChip(c, cu); err != nil {
		tmpl, act = "tammany/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()

	// Take Favor Chip
	g.endSlander(cu)
	cp.Chips[n]++
	cp.PerformedAction = true

	// Log Placement
	e := g.newTakeChipEntryFor(cp, n)
	restful.AddNoticef(c, string(e.HTML(c)))
	tmpl, act = "tammany/take_chip_update", game.Cache
	return
}

type takeChipEntry struct {
	*Entry
	Chip nationality
}

func (g *Game) newTakeChipEntryFor(p *Player, n nationality) (e *takeChipEntry) {
	e = new(takeChipEntry)
	e.Entry = g.newEntryFor(p)
	e.Chip = n
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return
}

func (e *takeChipEntry) HTML(c *gin.Context) template.HTML {
	g := gameFrom(c)
	return restful.HTML("%s took a %s favor chip.", g.NameByPID(e.PlayerID), e.Chip)
}

func (g *Game) validateTakeChip(c *gin.Context, cu *user.User) (n nationality, err error) {
	var ok bool
	if n, ok = toNationality[c.PostForm("chip")]; !ok {
		err = sn.NewVError("Invalid value received for chip nationatlity.")
		return
	}

	cp := g.CurrentPlayer()

	switch {
	case !g.IsCurrentPlayer(cu):
		err = sn.NewVError("Only the current player can take a chip.")
	case cp.PerformedAction:
		err = sn.NewVError("You have already performed an action.")
	case g.Phase != takeFavorChip:
		err = sn.NewVError("You can't take a favour chip in phase %q.", g.PhaseName())
	}
	return
}
