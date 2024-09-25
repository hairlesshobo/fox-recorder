package custom

import (
	"fmt"
	// "math"
	"slices"
	"sort"
	"sync"
	"time"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

// LevelMeter indicates the level of an audio signal.
type LevelMeter struct {
	*cview.Box

	// Rune to use when rendering the empty area of the level meter.
	emptyRune rune

	// Color of the empty area of the level meter.
	emptyColor tcell.Color

	// Rune to use when rendering the filled area of the level meter.
	filledRune rune

	// Current level.
	level int

	peakLevel int

	peakHoldTimeMs int

	lastPeakTime int64

	longTermMaxLevel int

	channelNumber string

	// Maximum level passable to the level meter
	maxLevel int

	// Minimum level represented on the level meter
	minLevel int

	// slice containing meter level steps
	meterSteps []int

	// meter level to foreground color map
	colorMap map[int]tcell.Color

	sync.RWMutex
}

// NewLevelMeter returns a new level meter bar.
func NewLevelMeter(meterSteps []int, colorMap map[int]tcell.Color) *LevelMeter {
	p := &LevelMeter{
		Box:              cview.NewBox(),
		emptyRune:        tcell.RuneBlock,
		emptyColor:       cview.Styles.PrimitiveBackgroundColor,
		filledRune:       tcell.RuneBlock,
		maxLevel:         slices.Max(meterSteps),
		minLevel:         slices.Min(meterSteps),
		peakHoldTimeMs:   750,
		peakLevel:        -150,
		level:            -150,
		longTermMaxLevel: -150,
		channelNumber:    "",
		meterSteps:       meterSteps,
		colorMap:         colorMap,
	}
	p.SetBackgroundColor(cview.Styles.PrimitiveBackgroundColor)
	return p
}

func (p *LevelMeter) SetChannelNumber(name string) {
	p.Lock()
	defer p.Unlock()

	p.channelNumber = name
}

// SetEmptyRune sets the rune used for the empty area of the level meter.
func (p *LevelMeter) SetEmptyRune(empty rune) {
	p.Lock()
	defer p.Unlock()

	p.emptyRune = empty
}

// SetEmptyColor sets the color of the empty area of the level meter.
func (p *LevelMeter) SetEmptyColor(empty tcell.Color) {
	p.Lock()
	defer p.Unlock()

	p.emptyColor = empty
}

// SetFilledRune sets the rune used for the filled area of the level meter.
func (p *LevelMeter) SetFilledRune(filled rune) {
	p.Lock()
	defer p.Unlock()

	p.filledRune = filled
}

func (p *LevelMeter) SetLongTermMaxLevel(level int) {
	p.Lock()
	defer p.Unlock()

	p.longTermMaxLevel = level

	if p.longTermMaxLevel < p.minLevel {
		p.longTermMaxLevel = p.minLevel
	}
}

func (p *LevelMeter) GetLongTermMaxLevel() int {
	p.RLock()
	defer p.RUnlock()

	return p.longTermMaxLevel
}

func (p *LevelMeter) SetPeakHoldTime(time int) {
	p.Lock()
	defer p.Unlock()

	p.peakHoldTimeMs = time
}

func (p *LevelMeter) SetMinLevel(level int) {
	p.Lock()
	defer p.Unlock()

	p.minLevel = level
}

// SetLevel sets the current level.
func (p *LevelMeter) SetLevel(level int) {
	p.Lock()
	defer p.Unlock()

	p.level = level

	if p.level < p.minLevel {
		p.level = p.minLevel
	} else if p.level > p.maxLevel {
		p.level = p.maxLevel
	}

	if p.level > p.peakLevel || (time.Now().UnixMilli()-p.lastPeakTime) > int64(p.peakHoldTimeMs) {
		p.peakLevel = p.level
		p.lastPeakTime = time.Now().UnixMilli()
	}
}

// GetLevel gets the current level.
func (p *LevelMeter) GetLevel() int {
	p.RLock()
	defer p.RUnlock()

	return p.level
}

func getLevelColor(colorMap map[int]tcell.Color, currentLevel int) tcell.Color {

	keys := make([]int, 0, len(colorMap))

	for k := range colorMap {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))

	for key := range keys {
		mapLevel := keys[key]
		mapColor := colorMap[mapLevel]
		if currentLevel >= mapLevel {
			return mapColor
		}
	}

	return tcell.ColorPurple
}

// Draw draws this primitive onto the screen.
func (p *LevelMeter) Draw(screen tcell.Screen) {
	if !p.GetVisible() {
		return
	}

	p.Box.Draw(screen)

	p.Lock()
	defer p.Unlock()

	x, y, meterWidth, _ := p.GetInnerRect()
	foundPeak := false

	if len(p.channelNumber) > 0 {
		fmtString := fmt.Sprintf("%%%dv", meterWidth)
		runeArray := []rune(fmt.Sprintf(fmtString, p.channelNumber))
		for w := 0; w < meterWidth; w++ {
			screen.SetContent(x+w, y, runeArray[w], nil, tcell.StyleDefault.Bold(true))
		}
	}

	y += 1

	for step := 0; step < len(p.meterSteps); step++ {
		stepLevel := p.meterSteps[step]
		doDraw := false
		foregroundColor := getLevelColor(p.colorMap, stepLevel)
		style := tcell.StyleDefault.Foreground(foregroundColor).Background(p.GetBackgroundColor())

		if !foundPeak && p.peakLevel >= stepLevel {
			foundPeak = true
			style = tcell.StyleDefault.Bold(true).Foreground(foregroundColor).Background(p.GetBackgroundColor())
			doDraw = true
		} else {
			if p.level >= stepLevel {
				doDraw = true
			}
		}

		if doDraw {
			for w := 0; w < meterWidth; w++ {
				screen.SetContent(x+w, y+(step), p.filledRune, nil, style)
			}
		}
	}

	y += len(p.meterSteps)

	// show max value
	// maxValue := int(math.Abs(float64(p.longTermMaxLevel)))
	fmtString := fmt.Sprintf("%%%dv", meterWidth)
	runeArray := []rune(fmt.Sprintf(fmtString, p.longTermMaxLevel))
	for w := 0; w < meterWidth; w++ {
		screen.SetContent(x+w, y, runeArray[w], nil, tcell.StyleDefault.Bold(true))
	}
}
