package tammany

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/mlog"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
)

// AddRoutes addes routing for game.
func AddRoutes(prefix string, engine *gin.Engine) {
	// New
	g1 := engine.Group(prefix)
	g1.GET("/game/new",
		user.RequireCurrentUser(),
		gtype.SetTypes(),
		newAction(prefix),
	)

	// Create
	g1.POST("/game",
		user.RequireCurrentUser(),
		create(prefix),
	)

	// Show
	g1.GET("/game/show/:hid",
		//game.FetchHeader(GamesRoot),
		fetch,
		mlog.Get,
		game.SetAdmin(false),
		show(prefix),
	)

	// Admin
	g1.GET("/game/admin/:hid",
		//game.FetchHeader(GamesRoot),
		fetch,
		mlog.Get,
		game.SetAdmin(true),
		show(prefix),
	)

	// Undo
	g1.POST("/game/undo/:hid",
		//game.FetchHeader(GamesRoot),
		//UndoUpdate(),
		fetch,
		undo(prefix),
	)

	//	// Redo
	//	g1.POST("/game/redo/:hid",
	//		//game.FetchHeader(GamesRoot),
	//		RedoUpdate(),
	//		Redo(prefix),
	//	)
	//
	//	// Reset
	//	g1.POST("/game/reset/:hid",
	//		//game.FetchHeader(GamesRoot),
	//		ResetUpdate(),
	//		Reset(prefix),
	//	)

	// Finish
	g1.POST("/game/finish/:hid",
		//game.FetchHeader(GamesRoot),
		fetch,
		stats.Fetch(user.CurrentFrom),
		finish(prefix),
	)

	// Drop
	g1.POST("/game/drop/:hid",
		user.RequireCurrentUser(),
		//game.FetchHeader(GamesRoot),
		fetch,
		drop(prefix),
	)

	// Accept
	g1.POST("/game/accept/:hid",
		user.RequireCurrentUser(),
		//game.FetchHeader(GamesRoot),
		fetch,
		accept(prefix),
	)

	// Update
	g1.PUT("/game/show/:hid",
		user.RequireCurrentUser(),
		//game.FetchHeader(GamesRoot),
		fetch,
		game.RequireCurrentPlayerOrAdmin(),
		game.SetAdmin(false),
		update(prefix),
	)

	// Admin Update
	g1.POST("/game/admin/:hid",
		user.RequireCurrentUser(),
		//game.FetchHeader(GamesRoot),
		fetch,
		game.RequireCurrentPlayerOrAdmin(),
		game.SetAdmin(true),
		update(prefix),
	)

	g1.PUT("/game/admin/:hid",
		user.RequireCurrentUser(),
		//game.FetchHeader(GamesRoot),
		fetch,
		game.RequireCurrentPlayerOrAdmin(),
		game.SetAdmin(true),
		update(prefix),
	)

	// Index
	g1.GET("/games/:status",
		gtype.SetTypes(),
		index(prefix),
	)

	// JSON Data for Index
	g1.POST("games/:status/json",
		gtype.SetTypes(),
		game.GetFiltered(gtype.Tammany),
		jsonIndexAction(prefix),
	)

	// Add Message
	g1.PUT("/game/show/:hid/addmessage",
		user.RequireCurrentUser(),
		mlog.Get,
		mlog.AddMessage(prefix),
	)
}
