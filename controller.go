package tammany

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/codec"
	"github.com/SlothNinja/color"
	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/memcache"
	"github.com/SlothNinja/mlog"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
)

const (
	gameKey   = "Game"
	homePath  = "/"
	jsonKey   = "JSON"
	statusKey = "Status"
	hParam    = "hid"
)

func gameFrom(c *gin.Context) (g *Game) {
	g, _ = c.Value(gameKey).(*Game)
	return
}

func withGame(c *gin.Context, g *Game) (ret *gin.Context) {
	ret = c
	c.Set(gameKey, g)
	return
}

func jsonFrom(c *gin.Context) (g *Game) {
	g, _ = c.Value(jsonKey).(*Game)
	return
}

func withJSON(c *gin.Context, g *Game) (ret *gin.Context) {
	ret = c
	c.Set(jsonKey, g)
	return
}

//type Action func(*restful.Context, *Game, url.Values) (string, game.ActionType, error)
//
//var actionMap = map[string]Action{
//	"select-area":         selectArea,
//	"assign-office":       assignOffice,
//	"place-pieces":        placePieces,
//	"remove":              removeImmigrant,
//	"move-from":           moveFrom,
//	"move-to":             moveTo,
//	"place-lockup-marker": placeLockupMarker,
//	"deputy-take-chip":    deputyTakeChip,
//	"take-chip":           takeChip,
//	"slander":             slander,
//	"bid":                 bid,
//	"undo":                undoAction,
//	"redo":                redoAction,
//	"reset":               resetTurn,
//	"finish":              finishTurn,
//	"game-state":          adminState,
//	"player":              adminPlayer,
//	"ward":                adminWard,
//	"castle-garden":       adminCastleGarden,
//	"immigrant-bag":       adminImmigrantBag,
//	"warn-office":         warnOffice,
//	"confirm-finish":      confirmFinishTurn,
//	"cancel-finish":       cancelFinish,
//}

func (g *Game) update(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	switch a := c.PostForm("action"); a {
	case "select-area":
		return g.selectArea(c)
	case "assign-office":
		return g.assignOffice(c)
	case "place-pieces":
		return g.placePieces(c)
	case "remove":
		return g.removeImmigrant(c)
	case "move-from":
		return g.moveFrom(c)
	case "move-to":
		return g.moveTo(c)
	case "place-lockup-marker":
		return g.placeLockupMarker(c)
	case "deputy-take-chip":
		return g.deputyTakeChip(c)
	case "take-chip":
		return g.takeChip(c)
	case "slander":
		return g.slander(c)
	case "bid":
		return g.bid(c)
	case "undo":
		return g.undoAction(c)
	case "redo":
		return g.redoAction(c)
	case "reset":
		return g.resetTurn(c)
	case "cancel-finish":
		return g.cancelFinish(c)
	case "game-state":
		return g.adminState(c)
	case "player":
		return g.adminPlayer(c)
	case "ward":
		return g.adminWard(c)
	case "castle-garden":
		return g.adminCastleGarden(c)
	case "immigrant-bag":
		return g.adminImmigrantBag(c)
	default:
		return "tammany/flash_notice", game.None, sn.NewVError("%v is not a valid action.", a)
	}
	// return
}

func (client Client) show(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := gameFrom(c)
		cu := user.CurrentFrom(c)
		c.HTML(http.StatusOK, prefix+"/show", gin.H{
			"Context":    c,
			"VersionID":  sn.VersionID(),
			"CUser":      cu,
			"Game":       g,
			"IsAdmin":    user.IsAdmin(c),
			"Admin":      game.AdminFrom(c),
			"MessageLog": mlog.From(c),
			"ColorMap":   color.MapFrom(c),
			"Notices":    restful.NoticesFrom(c),
			"Errors":     restful.ErrorsFrom(c),
		})
	}
}

func (client Client) update(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := gameFrom(c)
		if g == nil {
			log.Errorf("Controller#Update Game Not Found")
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
		template, actionType, err := g.update(c)
		log.Debugf("err: %v", err)
		switch {
		case err != nil && sn.IsVError(err):
			restful.AddErrorf(c, "%v", err)
			withJSON(c, g)
		case err != nil:
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		case actionType == game.Cache:
			if err := g.cache(c); err != nil {
				restful.AddErrorf(c, "%v", err)
			}
			//mkey := g.UndoKey(c)
			//item := memcache.NewItem(ctx, mkey).SetExpiration(time.Minute * 30)
			//v, err := codec.Encode(g)
			//if err != nil {
			//	log.Errorf(c, "Controller#Update Cache Error: %s", err)
			//	c.Redirect(http.StatusSeeOther, showPath(c, prefix))
			//	return
			//}
			//item.SetValue(v)
			//if err := memcache.Set(ctx, item); err != nil {
			//	log.Errorf(c, "Controller#Update Cache Error: %s", err)
			//	c.Redirect(http.StatusSeeOther, showPath(c, prefix))
			//	return
			//}
			//		case actionType == game.SaveAndStatUpdate:
			//			if err := g.saveAndUpdateStats(c); err != nil {
			//				log.Errorf(c, "%s", err)
			//				restful.AddErrorf(c, "Controller#Update SaveAndStatUpdate Error: %s", err)
			//				c.Redirect(http.StatusSeeOther, showPath(c, prefix))
			//				return
			//			}
		case actionType == game.Save:
			if err := client.save(c, g); err != nil {
				log.Errorf("%s", err)
				restful.AddErrorf(c, "Controller#Update Save Error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		case actionType == game.Undo:
			mkey := g.UndoKey(c)
			if err := memcache.Delete(c, mkey); err != nil && err != memcache.ErrCacheMiss {
				log.Errorf("memcache.Delete error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		}

		switch jData := jsonFrom(c); {
		case jData != nil && template == "json":
			log.Debugf("jData: %v", jData)
			log.Debugf("template: %v", template)
			c.JSON(http.StatusOK, jData)
		case template == "":
			log.Debugf("template: %v", template)
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
		default:
			log.Debugf("template: %v", template)
			cu := user.CurrentFrom(c)

			d := gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     cu,
				"Game":      g,
				"Ward":      wardFrom(c),
				"Office":    officeFrom(c),
				"IsAdmin":   user.IsAdmin(c),
				"Notices":   restful.NoticesFrom(c),
				"Errors":    restful.ErrorsFrom(c),
			}
			log.Debugf("d: %#v", d)
			c.HTML(http.StatusOK, template, d)
		}
	}
}

func (client Client) save(c *gin.Context, g *Game) error {
	_, err := client.RunInTransaction(c, func(tx *datastore.Transaction) error {
		oldG := New(c, g.ID())
		err := tx.Get(oldG.Key, oldG.Header)
		if err != nil {
			return err
		}

		if oldG.UpdatedAt != g.UpdatedAt {
			return fmt.Errorf("Game state changed unexpectantly.  Try again.")
		}

		err = g.encode(c)
		if err != nil {
			return err
		}

		_, err = tx.Put(g.Key, g.Header)
		if err != nil {
			return err
		}

		err = memcache.Delete(c, g.UndoKey(c))
		if err == memcache.ErrCacheMiss {
			return nil
		}
		return err
	})
	return err
}

func (client Client) saveWith(c *gin.Context, g *Game, ks []*datastore.Key, es []interface{}) error {
	_, err := client.RunInTransaction(c, func(tx *datastore.Transaction) error {
		oldG := New(c, g.ID())
		err := tx.Get(oldG.Key, oldG.Header)
		if err != nil {
			return err
		}

		if oldG.UpdatedAt != g.UpdatedAt {
			return fmt.Errorf("Game state changed unexpectantly.  Try again.")
		}

		err = g.encode(c)
		if err != nil {
			return err
		}

		ks = append(ks, g.Key)
		es = append(es, g.Header)

		_, err = tx.PutMulti(ks, es)
		if err != nil {
			return err
		}

		err = memcache.Delete(c, g.UndoKey(c))
		if err == memcache.ErrCacheMiss {
			return nil
		}
		return err
	})
	return err
}

func (g *Game) encode(c *gin.Context) (err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var encoded []byte
	if encoded, err = codec.Encode(g.State); err != nil {
		return
	}
	g.SavedState = encoded
	g.updateHeader()

	return
}

func (g *Game) cache(c *gin.Context) error {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	item := &memcache.Item{
		Key:        g.UndoKey(c),
		Expiration: time.Minute * 30,
	}
	v, err := codec.Encode(g)
	if err != nil {
		return err
	}
	item.Value = v
	return memcache.Set(c, item)
}

func wrap(s *stats.Stats, cs contest.Contests) ([]*datastore.Key, []interface{}) {
	l := len(cs) + 1
	es := make([]interface{}, l)
	ks := make([]*datastore.Key, l)
	es[0] = s
	ks[0] = s.Key
	for i, c := range cs {
		es[i+1] = c
		ks[i+1] = c.Key
	}
	return ks, es
}

func showPath(prefix, hid string) string {
	return fmt.Sprintf("/%s/game/show/%s", prefix, hid)
}

func recruitingPath(prefix string) string {
	return fmt.Sprintf("/%s/games/recruiting", prefix)
}

func newPath(prefix string) string {
	return fmt.Sprintf("/%s/game/new", prefix)
}

func newGamer(c *gin.Context) game.Gamer {
	return New(c, 0)
}

func (client Client) undo(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")
		c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))

		g := gameFrom(c)
		if g == nil {
			log.Errorf("Controller#Update Game Not Found")
			return
		}
		mkey := g.UndoKey(c)
		if err := memcache.Delete(c, mkey); err != nil && err != memcache.ErrCacheMiss {
			log.Errorf("Controller#Undo Error: %s", err)
		}
	}
}

func (client Client) index(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		gs := game.GamersFrom(c)
		switch status := game.StatusFrom(c); status {
		case game.Recruiting:
			c.HTML(http.StatusOK, "shared/invitation_index", gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     user.CurrentFrom(c),
				"Games":     gs,
				"Type":      gtype.Indonesia.String(),
			})
		default:
			c.HTML(http.StatusOK, "shared/games_index", gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     user.CurrentFrom(c),
				"Games":     gs,
				"Type":      gtype.Indonesia.String(),
				"Status":    status,
			})
		}
	}
}

func (client Client) newAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := New(c, 0)
		withGame(c, g)
		if err := g.FromParams(c, gtype.GOT); err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		c.HTML(http.StatusOK, prefix+"/new", gin.H{
			"Context":   c,
			"VersionID": sn.VersionID(),
			"CUser":     user.CurrentFrom(c),
			"Game":      g,
		})
	}
}

func (client Client) create(prefix string) (f gin.HandlerFunc) {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := New(c, 0)
		withGame(c, g)

		err := g.FromForm(c, g.Type)
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		err = g.encode(c)
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		ks, err := client.AllocateIDs(c, []*datastore.Key{g.Key})
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		k := ks[0]

		_, err = client.RunInTransaction(c, func(tx *datastore.Transaction) error {
			m := mlog.New(k.ID)
			ks := []*datastore.Key{m.Key, k}
			es := []interface{}{m, g.Header}
			_, err := tx.PutMulti(ks, es)
			return err
		})
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}
		restful.AddNoticef(c, "<div>%s created.</div>", g.Title)
		c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
	}
}

func (client Client) accept(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := gameFrom(c)
		if g == nil {
			log.Errorf("game not found")
			restful.AddErrorf(c, "game not found")
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		u := user.CurrentFrom(c)
		start, err := g.Accept(c, u)
		if err != nil {
			log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		if start {
			g.start(c)
		}

		err = client.save(c, g)
		if err != nil {
			log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		if start {
			err = g.SendTurnNotificationsTo(c, g.CurrentPlayer())
			if err != nil {
				log.Warningf(err.Error())
			}
		}
		c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
	}
}

func (client Client) drop(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := gameFrom(c)
		if g == nil {
			log.Errorf("game not found")
			restful.AddErrorf(c, "game not found")
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		u := user.CurrentFrom(c)
		err := g.Drop(u)
		if err != nil {
			log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		err = client.save(c, g)
		if err != nil {
			log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
		}

		c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
	}
}

func (client Client) fetch(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	// create Gamer
	id, err := strconv.ParseInt(c.Param("hid"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	g := New(c, id)

	switch action := c.PostForm("action"); {
	case action == "reset":
		// pull from memcache/datastore
		// same as undo & !MultiUndo
		fallthrough
	case action == "undo":
		// pull from memcache/datastore
		err := client.dsGet(c, g)
		if err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
		}
	default:
		if user.CurrentFrom(c) != nil {
			// pull from memcache and return if successful; otherwise pull from datastore
			err := client.mcGet(c, g)
			if err == nil {
				return
			}
		}

		err := client.dsGet(c, g)
		if err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
		}
	}
}

// pull temporary game state from memcache.  Note may be different from value stored in datastore.
func (client Client) mcGet(c *gin.Context, g *Game) error {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	mkey := g.GetHeader().UndoKey(c)

	item, err := memcache.Get(c, mkey)
	if err != nil {
		log.Debugf("err: %v", err)
		return err
	}

	err = codec.Decode(g, item.Value)
	if err != nil {
		log.Debugf("err: %v", err)
		return err
	}

	err = client.afterCache(c, g)
	if err != nil {
		log.Debugf("err: %v", err)
		return err
	}
	color.WithMap(withGame(c, g), g.ColorMapFor(user.CurrentFrom(c)))
	return nil
}

// pull game state from memcache/datastore.  returned memcache should be same as datastore.
func (client Client) dsGet(c *gin.Context, g *Game) error {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	err := client.Get(c, g.Key, g.Header)
	if err != nil {
		log.Debugf("err: %v", err)
		restful.AddErrorf(c, err.Error())
		return err
	}

	if g == nil {
		return fmt.Errorf("Unable to get game for id: %v", g.ID)
	}

	s := newState()
	err = codec.Decode(&s, g.SavedState)
	if err != nil {
		log.Debugf("err: %v", err)
		restful.AddErrorf(c, err.Error())
		return err
	}
	g.State = s

	err = client.init(c, g)
	if err != nil {
		log.Debugf("err: %v", err)
		restful.AddErrorf(c, err.Error())
		return err
	}
	cm := g.ColorMapFor(user.CurrentFrom(c))
	color.WithMap(withGame(c, g), cm)
	return nil
}

func (client Client) jsonIndexAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		client.Game.JSONIndexAction(c)
	}
}

func (g *Game) updateHeader() {
	switch g.Phase {
	case gameOver:
		g.Progress = g.PhaseName()
	default:
		g.Progress = fmt.Sprintf("<div>Year: %d</div><div>Phase: %s</div>", g.Year(), g.PhaseName())
	}

	if u := g.Creator; u != nil {
		g.CreatorSID = user.GenID(u.GoogleID)
		g.CreatorName = u.Name
	}

	if l := len(g.Users); l > 0 {
		g.UserSIDS = make([]string, l)
		g.UserNames = make([]string, l)
		g.UserEmails = make([]string, l)
		for i, u := range g.Users {
			g.UserSIDS[i] = user.GenID(u.GoogleID)
			g.UserNames[i] = u.Name
			g.UserEmails[i] = u.Email
		}
	}
}
