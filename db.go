package tammany

import (
	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	gtype "github.com/SlothNinja/type"
	"github.com/gin-gonic/gin"
)

const kind = "Game"

// New creates a new game.
func New(c *gin.Context, id int64) (g *Game) {
	g = new(Game)
	g.Header = game.NewHeader(c, g, id)
	g.State = newState()
	g.Key.Parent = pk(c)
	g.Type = gtype.Tammany
	return
}

func newState() *State {
	return new(State)
}

func pk(c *gin.Context) *datastore.Key {
	return datastore.NameKey(gtype.Tammany.SString(), "root", game.GamesRoot(c))
}

func newKey(c *gin.Context, id int64) *datastore.Key {
	return datastore.IDKey("Game", id, pk(c))
}

func (g *Game) init(c *gin.Context) (err error) {
	if err = g.Header.AfterLoad(g); err != nil {
		return
	}

	for _, player := range g.Players() {
		player.init(g)
	}

	return
}

func (g *Game) afterCache(c *gin.Context) error {
	return g.init(c)
}
