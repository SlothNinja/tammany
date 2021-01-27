package tammany

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/codec"
	"github.com/SlothNinja/color"
	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/mlog"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

const (
	gameKey   = "Game"
	homePath  = "/"
	jsonKey   = "JSON"
	statusKey = "Status"
	hParam    = "hid"
	msgEnter  = "Entering"
	msgExit   = "Exiting"
)

var (
	ErrInvalidID = errors.New("invalid identifier")
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

func (g *Game) update(c *gin.Context, cu *user.User) (string, game.ActionType, error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	switch a := c.PostForm("action"); a {
	case "select-area":
		return g.selectArea(c)
	case "assign-office":
		return g.assignOffice(c, cu)
	case "place-pieces":
		return g.placePieces(c, cu)
	case "remove":
		return g.removeImmigrant(c, cu)
	case "move-from":
		return g.moveFrom(c, cu)
	case "move-to":
		return g.moveTo(c, cu)
	case "place-lockup-marker":
		return g.placeLockupMarker(c, cu)
	case "deputy-take-chip":
		return g.deputyTakeChip(c, cu)
	case "take-chip":
		return g.takeChip(c, cu)
	case "slander":
		return g.slander(c, cu)
	case "bid":
		return g.bid(c, cu)
	case "undo":
		return g.undoAction(c, cu)
	case "redo":
		return g.redoAction(c, cu)
	case "reset":
		return g.resetTurn(c, cu)
	case "cancel-finish":
		return g.cancelFinish(c, cu)
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
}

// func isAdmin(u *user.User) bool {
// 	if u == nil {
// 		return false
// 	}
// 	return u.Admin
// }

func (client *Client) show(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		id, err := getID(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			return
		}

		ml, err := client.MLog.Get(c, id)
		if err != nil {
			client.Log.Errorf(err.Error())
			return
		}

		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			return
		}

		c.HTML(http.StatusOK, prefix+"/show", gin.H{
			"Context":    c,
			"VersionID":  sn.VersionID(),
			"CUser":      cu,
			"Game":       gameFrom(c),
			"IsAdmin":    cu.IsAdmin(),
			"Admin":      game.AdminFrom(c),
			"MessageLog": ml,
			"ColorMap":   color.MapFrom(c),
			"Notices":    restful.NoticesFrom(c),
			"Errors":     restful.ErrorsFrom(c),
		})
	}
}

func (client *Client) addMessage(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		id, err := getID(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			return
		}

		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			return
		}

		ml, err := client.MLog.Get(c, id)
		if err != nil {
			client.Log.Errorf(err.Error())
			return
		}

		m := ml.AddMessage(cu, c.PostForm("message"))

		_, err = client.MLog.Put(c, id, ml)
		if err != nil {
			log.Errorf(err.Error())
			return
		}

		c.HTML(http.StatusOK, "shared/message", gin.H{
			"message": m,
			"ctx":     c,
			"map":     gameFrom(c).ColorMapFor(cu),
			"link":    cu.Link(),
		})
	}
}

func (client *Client) update(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}

		g := gameFrom(c)
		if g == nil {
			client.Log.Errorf("Controller#Update Game Not Found")
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
		template, actionType, err := g.update(c, cu)
		switch {
		case err != nil && sn.IsVError(err):
			restful.AddErrorf(c, "%v", err)
			withJSON(c, g)
		case err != nil:
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		case actionType == game.Cache:
			client.Cache.SetDefault(g.UndoKey(cu), g)
		case actionType == game.Save:
			err = client.save(c, g, cu)
			if err != nil {
				client.Log.Errorf(err.Error())
				restful.AddErrorf(c, "Controller#Update Save Error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		case actionType == game.Undo:
			mkey := g.UndoKey(cu)
			client.Cache.Delete(mkey)
		}

		switch jData := jsonFrom(c); {
		case jData != nil && template == "json":
			c.JSON(http.StatusOK, jData)
		case template == "":
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
		default:
			d := gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     cu,
				"Game":      g,
				"Ward":      wardFrom(c),
				"Office":    officeFrom(c),
				"IsAdmin":   cu.IsAdmin(),
				"Notices":   restful.NoticesFrom(c),
				"Errors":    restful.ErrorsFrom(c),
			}
			c.HTML(http.StatusOK, template, d)
		}
	}
}

func (client *Client) save(c *gin.Context, g *Game, cu *user.User) error {
	_, err := client.DS.RunInTransaction(c, func(tx *datastore.Transaction) error {
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

		client.Cache.Delete(g.UndoKey(cu))
		return nil
	})
	return err
}

func (client *Client) saveWith(c *gin.Context, g *Game, cu *user.User, ks []*datastore.Key, es []interface{}) error {
	_, err := client.DS.RunInTransaction(c, func(tx *datastore.Transaction) error {
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

		client.Cache.Delete(g.UndoKey(cu))
		return nil
	})
	return err
}

func (g *Game) encode(c *gin.Context) (err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	var encoded []byte
	if encoded, err = codec.Encode(g.State); err != nil {
		return
	}
	g.SavedState = encoded
	g.updateHeader()

	return
}

// func (g *Game) cache(c *gin.Context) error {
// 	log.Debugf(msgEnter)
// 	defer log.Debugf(msgExit)
//
// 	item := &memcache.Item{
// 		Key:        g.UndoKey(c),
// 		Expiration: time.Minute * 30,
// 	}
// 	v, err := codec.Encode(g)
// 	if err != nil {
// 		return err
// 	}
// 	item.Value = v
// 	return memcache.Set(c, item)
// }

func wrap(s *user.Stats, cs []*contest.Contest) ([]*datastore.Key, []interface{}) {
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

func (client *Client) undo(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
			return
		}

		g := gameFrom(c)
		if g == nil {
			client.Log.Errorf("Controller#Update Game Not Found")
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
			return
		}

		mkey := g.UndoKey(cu)
		client.Cache.Delete(mkey)
		c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
	}
}

func (client *Client) index(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		gs := game.GamersFrom(c)
		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Debugf(err.Error())
		}
		switch status := game.StatusFrom(c); status {
		case game.Recruiting:
			c.HTML(http.StatusOK, "shared/invitation_index", gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     cu,
				"Games":     gs,
				"Type":      gtype.Indonesia.String(),
			})
		default:
			c.HTML(http.StatusOK, "shared/games_index", gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     cu,
				"Games":     gs,
				"Type":      gtype.Indonesia.String(),
				"Status":    status,
			})
		}
	}
}

func (client *Client) newAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		g := New(c, 0)
		withGame(c, g)

		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		err = g.FromParams(c, cu, gtype.GOT)
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		c.HTML(http.StatusOK, prefix+"/new", gin.H{
			"Context":   c,
			"VersionID": sn.VersionID(),
			"CUser":     cu,
			"Game":      g,
		})
	}
}

func (client *Client) create(prefix string) (f gin.HandlerFunc) {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		g := New(c, 0)
		withGame(c, g)

		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		err = g.FromForm(c, cu, g.Type)
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		err = g.encode(c)
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		ks, err := client.DS.AllocateIDs(c, []*datastore.Key{g.Key})
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		k := ks[0]

		_, err = client.DS.RunInTransaction(c, func(tx *datastore.Transaction) error {
			m := mlog.New(k.ID)
			ks := []*datastore.Key{m.Key, k}
			es := []interface{}{m, g.Header}
			_, err := tx.PutMulti(ks, es)
			return err
		})
		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}
		restful.AddNoticef(c, "<div>%s created.</div>", g.Title)
		c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
	}
}

func (client *Client) accept(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		g := gameFrom(c)
		if g == nil {
			client.Log.Errorf("game not found")
			restful.AddErrorf(c, "game not found")
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Errorf(err.Error())
		}
		start, err := g.Accept(c, cu)
		if err != nil {
			client.Log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		if start {
			g.start(c)
		}

		err = client.save(c, g, cu)
		if err != nil {
			client.Log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		if start {
			err = g.SendTurnNotificationsTo(c, g.CurrentPlayerers()...)
			if err != nil {
				client.Log.Warningf(err.Error())
			}
		}
		c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
	}
}

func (client *Client) drop(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

		g := gameFrom(c)
		if g == nil {
			client.Log.Errorf("game not found")
			restful.AddErrorf(c, "game not found")
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Debugf(err.Error())
		}
		err = g.Drop(cu)
		if err != nil {
			client.Log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		err = client.save(c, g, cu)
		if err != nil {
			client.Log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
		}

		c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
	}
}

func (client *Client) fetch(c *gin.Context) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	// create Gamer
	id, err := strconv.ParseInt(c.Param("hid"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	g := New(c, id)

	switch action := c.PostForm("action"); {
	case action == "reset":
		// pull from cache/datastore
		// same as undo & !MultiUndo
		fallthrough
	case action == "undo":
		// pull from cache/datastore
		err := client.dsGet(c, g)
		if err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
		}
	default:
		cu, err := client.User.Current(c)
		if err != nil {
			client.Log.Debugf(err.Error())
		}
		if cu != nil {
			// pull from cache and return if successful; otherwise pull from datastore
			err := client.mcGet(c, g)
			if err == nil {
				return
			}
		}

		err = client.dsGet(c, g)
		if err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
		}
	}
}

// pull temporary game state from cache.  Note may be different from value stored in datastore.
func (client *Client) mcGet(c *gin.Context, g *Game) error {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	cu, err := client.User.Current(c)
	if err != nil {
		return err
	}

	mkey := g.GetHeader().UndoKey(cu)

	item, found := client.Cache.Get(mkey)
	if !found {
		return fmt.Errorf("game not found")
	}

	g2, ok := item.(*Game)
	if !ok {
		return fmt.Errorf("item not a *Game")
	}
	g2.SetCTX(c)

	g = g2
	color.WithMap(withGame(c, g), g.ColorMapFor(cu))
	return nil
}

// pull game state from cache/datastore.  returned memcache should be same as datastore.
func (client *Client) dsGet(c *gin.Context, g *Game) error {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	err := client.DS.Get(c, g.Key, g.Header)
	if err != nil {
		client.Log.Debugf("err: %v", err)
		restful.AddErrorf(c, err.Error())
		return err
	}

	if g == nil {
		return fmt.Errorf("Unable to get game for id: %v", g.ID)
	}

	s := newState()
	err = codec.Decode(&s, g.SavedState)
	if err != nil {
		client.Log.Debugf("err: %v", err)
		restful.AddErrorf(c, err.Error())
		return err
	}
	g.State = s

	err = client.init(c, g)
	if err != nil {
		client.Log.Debugf("err: %v", err)
		restful.AddErrorf(c, err.Error())
		return err
	}

	cu, err := client.User.Current(c)
	if err != nil {
		client.Log.Debugf(err.Error())
	}
	cm := g.ColorMapFor(cu)
	color.WithMap(withGame(c, g), cm)
	return nil
}

func (client *Client) jsonIndexAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf(msgEnter)
		defer client.Log.Debugf(msgExit)

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

func getID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("hid"), 10, 64)
	if err != nil {
		return -1, ErrInvalidID
	}
	return id, nil
}
