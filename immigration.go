package tammany

func defaultZone1Immigrants() Nationals {
	return Nationals{irish: 2, english: 2, german: 2}
}

func defaultZone2Immigrants() Nationals {
	return Nationals{irish: 2, english: 2, german: 1}
}

func defaultZone3Immigrants() Nationals {
	return Nationals{irish: 2, english: 1, german: 1}
}

func (g *Game) immigration() {
	switch g.Year() {
	case 1:
		switch g.NumPlayers {
		case 5:
			g.zone1Immigration()
			g.zone2Immigration()
			g.zone3Immigration()
		case 4:
			g.zone1Immigration()
			g.zone2Immigration()
		case 3:
			g.zone1Immigration()
		}
	case 5:
		switch g.NumPlayers {
		case 4:
			g.zone3Immigration()
		case 3:
			g.zone2Immigration()
		}
	case 9:
		switch g.NumPlayers {
		case 3:
			g.zone3Immigration()
		}
	}
}

func (g *Game) zone1Immigration() {
	immigrants := defaultZone1Immigrants()
	immigrants[irish]--
	for _, ward := range g.Zone1Wards() {
		switch ward.ID {
		case 14:
			ward.Immigrants[irish]++
		default:
			ward.Immigrants[immigrants.draw()]++
		}
	}
}

func (g *Game) zone2Immigration() {
	immigrants := defaultZone2Immigrants()
	for _, ward := range g.Zone2Wards() {
		ward.Immigrants[immigrants.draw()]++
	}
}

func (g *Game) zone3Immigration() {
	immigrants := defaultZone3Immigrants()
	for _, ward := range g.Zone3Wards() {
		ward.Immigrants[immigrants.draw()]++
	}
}
