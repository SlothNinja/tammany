package tammany

import (
	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/mlog"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
)

type server struct {
	*datastore.Client
}

func NewClient(dsClient *datastore.Client) server {
	return server{Client: dsClient}
}

// AddRoutes addes routing for game.
func (srv server) addRoutes(prefix string, engine *gin.Engine) *gin.Engine {
	// Game Group
	g := engine.Group(prefix + "/game")

	// New
	g.GET("/new",
		user.RequireCurrentUser(),
		gtype.SetTypes(),
		srv.newAction(prefix),
	)

	// Create
	g.POST("",
		user.RequireCurrentUser(),
		srv.create(prefix),
	)

	// Show
	g.GET("/show/:hid",
		srv.fetch,
		mlog.Get,
		game.SetAdmin(false),
		srv.show(prefix),
	)

	// Undo
	g.POST("/undo/:hid",
		srv.fetch,
		srv.undo(prefix),
	)

	// Finish
	g.POST("/finish/:hid",
		srv.fetch,
		stats.Fetch(user.CurrentFrom),
		srv.finish(prefix),
	)

	// Drop
	g.POST("/drop/:hid",
		user.RequireCurrentUser(),
		srv.fetch,
		srv.drop(prefix),
	)

	// Accept
	g.POST("/accept/:hid",
		user.RequireCurrentUser(),
		srv.fetch,
		srv.accept(prefix),
	)

	// Update
	g.PUT("/show/:hid",
		user.RequireCurrentUser(),
		srv.fetch,
		game.RequireCurrentPlayerOrAdmin(),
		game.SetAdmin(false),
		srv.update(prefix),
	)

	// Games Group
	gs := engine.Group(prefix + "/games")

	// Index
	gs.GET("/:status",
		gtype.SetTypes(),
		srv.index(prefix),
	)

	// JSON Data for Index
	gs.POST("/:status/json",
		gtype.SetTypes(),
		game.GetFiltered(gtype.Tammany),
		srv.jsonIndexAction(prefix),
	)

	// Add Message
	gs.PUT("/show/:hid/addmessage",
		user.RequireCurrentUser(),
		mlog.Get,
		mlog.AddMessage(prefix),
	)

	// Admin Group
	admin := g.Group("/admin", user.RequireAdmin)

	// Admin Get
	admin.GET("/:hid",
		srv.fetch,
		mlog.Get,
		game.SetAdmin(true),
		srv.show(prefix),
	)

	// Admin Update
	admin.POST("/hid",
		srv.fetch,
		game.SetAdmin(true),
		srv.update(prefix),
	)

	admin.PUT("/:hid",
		srv.fetch,
		game.SetAdmin(true),
		srv.update(prefix),
	)

	return engine
}
