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

func (client Client) init(c *gin.Context, g *Game) error {
	err := client.Game.AfterLoad(c, g.Header, g)
	if err != nil {
		return err
	}

	for _, player := range g.Players() {
		player.init(g)
	}

	return nil
}

func (client Client) afterCache(c *gin.Context, g *Game) error {
	return client.init(c, g)
}
