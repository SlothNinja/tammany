package tammany

import "github.com/SlothNinja/game"

const (
	noPhase game.Phase = iota
	actions
	placeImmigrant
	elections
	takeFavorChip
	endGameScoring
	scoreVictoryPoints
	assignCityOffices
	awardFavorChips
	setup
	castleGarden
	announceWinners
	gameOver
	assignDeputyMayor
	deputyMayorAssignOffice
)

var phaseNames = game.PhaseNameMap{
	noPhase:                 "None",
	actions:                 "Actions",
	placeImmigrant:          "Place Immigrant",
	elections:               "Elections",
	takeFavorChip:           "Take Favor Chip",
	endGameScoring:          "End Game Scoring",
	scoreVictoryPoints:      "Score Victory Points",
	assignCityOffices:       "Assign City Offices",
	awardFavorChips:         "Award Favor Chips",
	setup:                   "Setup",
	castleGarden:            "Castle Garden",
	announceWinners:         "Announce Winners",
	gameOver:                "Game Over",
	assignDeputyMayor:       "Assign Deputy Mayor",
	deputyMayorAssignOffice: "Deputy Mayor Assigns Office",
}

const (
	noSubPhase game.SubPhase = iota
	officeWarning
)
