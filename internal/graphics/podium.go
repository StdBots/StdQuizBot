package graphics

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"time"
)

// PlayerResult contains scoring info for the podium banner
type PlayerResult struct {
	Rank      int
	Name      string
	Username  string
	Score     int
	Correct   int
	Total     int
	AvgTime   float64
}

// PodiumData holds all parameters for rendering the victory banner
type PodiumData struct {
	QuizTitle   string
	TotalQ      int
	TotalPlayer int
	TopPlayers  []PlayerResult
	FinishedAt  time.Time
}

// RenderPodiumBanner generates an HD 1000x640 victory podium banner in pure Go
func RenderPodiumBanner(data PodiumData) ([]byte, error) {
	const width = 1000
	const height = 640

	c := NewCanvas(width, height)

	// Gradient Background: Deep Cyber Violet to Obsidian Black
	c.FillVerticalGradient(
		color.RGBA{R: 24, G: 16, B: 52, A: 255}, // #181034
		color.RGBA{R: 8, G: 5, B: 18, A: 255},   // #080512
	)
	c.DrawGridPattern(40, color.RGBA{R: 168, G: 85, B: 247, A: 12}) // Subtle Purple Grid

	// Header Badge
	badgeW := 260
	c.FillRoundedRect(35, 24, badgeW, 26, 6, color.RGBA{R: 126, G: 34, B: 206, A: 220})
	c.DrawText("QUIZ TOURNAMENT COMPLETED", 47, 30, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 1)

	// Quiz Title
	title := TruncateString(data.QuizTitle, 36)
	c.DrawText(title, 35, 58, color.RGBA{R: 250, G: 245, B: 255, A: 255}, 2)

	// Subtitle & Total Players
	sub := fmt.Sprintf("QUESTIONS: %d • TOTAL FIGHTERS: %d • %s",
		data.TotalQ, data.TotalPlayer, data.FinishedAt.Format("02 Jan, 15:04 MST"),
	)
	c.DrawText(sub, 35, 92, color.RGBA{R: 192, G: 132, B: 252, A: 255}, 1)

	// Divider
	c.FillRect(35, 114, width-70, 2, color.RGBA{R: 88, G: 28, B: 135, A: 180})

	// Podium Section (Top 3 Cards)
	// 3 Cards: 1st in Center, 2nd on Left, 3rd on Right (Olympic Podium style) or sequential 1, 2, 3
	cardW := 290
	cardH := 200
	podiumY := 134
	spacing := 25

	podiumOrder := []int{0, 1, 2}
	for i, idx := range podiumOrder {
		if idx >= len(data.TopPlayers) {
			continue
		}

		p := data.TopPlayers[idx]
		x := 35 + i*(cardW+spacing)

		var borderCol, badgeBg, medalCol color.RGBA
		var medalTitle string
		switch idx {
		case 0: // 1st Place
			borderCol = color.RGBA{R: 255, G: 215, B: 0, A: 220} // Gold
			badgeBg = color.RGBA{R: 110, G: 85, B: 0, A: 255}
			medalCol = color.RGBA{R: 255, G: 225, B: 50, A: 255}
			medalTitle = "🥇 CHAMPION (1ST)"
		case 1: // 2nd Place
			borderCol = color.RGBA{R: 203, G: 213, B: 225, A: 200} // Silver
			badgeBg = color.RGBA{R: 71, G: 85, B: 105, A: 255}
			medalCol = color.RGBA{R: 241, G: 245, B: 249, A: 255}
			medalTitle = "🥈 RUNNER-UP (2ND)"
		case 2: // 3rd Place
			borderCol = color.RGBA{R: 245, G: 158, B: 11, A: 200} // Bronze
			badgeBg = color.RGBA{R: 120, G: 53, B: 15, A: 255}
			medalCol = color.RGBA{R: 251, G: 191, B: 36, A: 255}
			medalTitle = "🥉 3RD PLACE"
		}

		// Outer Card
		c.FillRoundedRect(x, podiumY, cardW, cardH, 12, color.RGBA{R: 35, G: 22, B: 65, A: 230})
		c.FillRect(x+15, podiumY+10, cardW-30, 2, borderCol)

		// Medal Title Badge
		c.FillRoundedRect(x+16, podiumY+20, cardW-32, 28, 6, badgeBg)
		titleX := x + 16 + (cardW-32-len(medalTitle)*7)/2
		c.DrawText(medalTitle, titleX, podiumY+27, medalCol, 1)

		// Player Name
		pName := TruncateString(p.Name, 18)
		nameX := x + (cardW-len(pName)*14)/2
		c.DrawText(pName, nameX, podiumY+62, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 2)

		// Score & Stats
		scoreStr := fmt.Sprintf("%d POINTS", p.Score)
		sW := len(scoreStr) * 7
		c.DrawText(scoreStr, x+(cardW-sW)/2, podiumY+104, medalCol, 1)

		accStr := fmt.Sprintf("CORRECT: %d / %d", p.Correct, data.TotalQ)
		c.DrawText(accStr, x+24, podiumY+132, color.RGBA{R: 216, G: 180, B: 254, A: 255}, 1)

		speedStr := fmt.Sprintf("AVG SPEED: %.1fs", p.AvgTime)
		c.DrawText(speedStr, x+24, podiumY+156, color.RGBA{R: 192, G: 132, B: 252, A: 255}, 1)
	}

	// Runners List (Ranks 4 to 8)
	listY := 360
	c.DrawText("HONORABLE FIGHTERS (TOP 4 - 8):", 35, listY, color.RGBA{R: 216, G: 180, B: 254, A: 255}, 1)

	card4W := 445
	card4H := 46
	startY := listY + 22
	rowH := 56

	for r := 3; r < 7 && r < len(data.TopPlayers); r++ {
		p := data.TopPlayers[r]
		col := (r - 3) % 2
		row := (r - 3) / 2

		rx := 35 + col*(card4W+35)
		ry := startY + row*rowH

		c.FillRoundedRect(rx, ry, card4W, card4H, 8, color.RGBA{R: 28, G: 18, B: 52, A: 200})

		rankBadge := fmt.Sprintf("#%d", p.Rank)
		c.DrawText(rankBadge, rx+14, ry+15, color.RGBA{R: 168, G: 85, B: 247, A: 255}, 1)

		pName := TruncateString(p.Name, 16)
		c.DrawText(pName, rx+48, ry+15, color.RGBA{R: 245, G: 243, B: 255, A: 255}, 1)

		statStr := fmt.Sprintf("%d pts  (%.1fs)", p.Score, p.AvgTime)
		stW := len(statStr) * 7
		c.DrawText(statStr, rx+card4W-stW-16, ry+15, color.RGBA{R: 251, G: 191, B: 36, A: 255}, 1)
	}

	// Footer Bar
	footerY := height - 32
	c.FillRect(0, footerY, width, 32, color.RGBA{R: 5, G: 3, B: 12, A: 230})
	creditText := "⚡ POWERED BY @STDBOTS • STD DEEPANSHU • https://deepanshu.in"
	c.DrawText(creditText, 35, footerY+10, color.RGBA{R: 200, G: 200, B: 200, A: 255}, 1)

	botTag := "@StdQuizBot"
	botTagW := len(botTag) * 7
	c.DrawText(botTag, width-botTagW-35, footerY+10, color.RGBA{R: 192, G: 132, B: 252, A: 255}, 1)

	// Encode to PNG buffer
	var buf bytes.Buffer
	if err := png.Encode(&buf, c.Img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
