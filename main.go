package battlesnakegameformat

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

// View Structs - these are returned from https://engine.battlesnake.com/games/{id}

type ViewGame struct {
	Game       ViewGameSettings `json:"Game"`
	Frames     []ViewFrame      `json:"Frames"`
	FirstFrame ViewFrame        `json:"FirstFrame"`
	LastTurn   int32            `json:"LastTurn"`
}

type ViewGameSettings struct {
	ID      string      `json:"ID"`
	Ruleset ViewRuleset `json:"Ruleset"`
	Timeout int32       `json:"SnakeTimeout"`
	Status  string      `json:"Status"`
	Width   int32       `json:"Width"`
	Height  int32       `json:"Height"`
}

type ViewRuleset struct {
	FoodSpawnChance int32  `json:"foodSpawnChance,string"`
	MinimumFood     int32  `json:"minimumFood,string"`
	Name            string `json:"name"`
	Map             string `json:"map"`
	MapAuthor       string `json:"map_author"`
	DamagePerTurn   int32  `json:"damagePerTurn,string"`
}

type ViewTurn struct {
	Frames []ViewFrame `json:"Frames"`
	Count  int32       `json:"Count"`
}

type ViewFrame struct {
	Turn    int32       `json:"Turn"`
	Snakes  []ViewSnake `json:"Snakes"`
	Food    []ViewCoord `json:"Food"`
	Hazards []ViewCoord `json:"Hazards"`
}

type ViewSnake struct {
	ID         string      `json:"ID"`
	Name       string      `json:"Name"`
	URL        string      `json:"URL"`
	Body       []ViewCoord `json:"Body"`
	Health     int32       `json:"Health"`
	Color      string      `json:"Color"`
	HeadType   string      `json:"HeadType"`
	TailType   string      `json:"TailType"`
	Latency    string      `json:"Latency"`
	Shout      string      `json:"Shout"`
	Squad      string      `json:"Squad"`
	APIVersion string      `json:"APIVersion"`
	Author     string      `json:"Author"`
	Death      ViewDeath   `json:"Death"`
}

type ViewDeath struct {
	Cause        string `json:"Cause"`
	Turn         int32  `json:"Turn"`
	EliminatedBy string `json:"EliminatedBy"`
}

type ViewCoord struct {
	X int32 `json:"X"`
	Y int32 `json:"Y"`
}

type ViewGameResponse struct {
	Game ViewGameSettings `json:"Game"`
	// ignore LastFrame
}

// Move Structs - these are sent to http://battlesnake-url/move

type MoveGameState struct {
	Game  MoveGame        `json:"game"`
	Turn  int32           `json:"turn"`
	Board MoveBoard       `json:"board"`
	You   MoveBattlesnake `json:"you"`
}

type MoveGame struct {
	ID      string      `json:"id"`
	Ruleset MoveRuleset `json:"ruleset"`
	Timeout int32       `json:"timeout"`
}

type MoveRuleset struct {
	Name     string       `json:"name"`
	Version  string       `json:"version"`
	Settings MoveSettings `json:"settings"`
}

type MoveSettings struct {
	FoodSpawnChance     int32      `json:"foodSpawnChance"`
	MinimumFood         int32      `json:"minimumFood"`
	HazardDamagePerTurn int32      `json:"hazardDamagePerTurn"`
	Royale              MoveRoyale `json:"royale"`
	Squad               MoveSquad  `json:"squad"`
}

type MoveRoyale struct {
	ShrinkEveryNTurns int32 `json:"shrinkEveryNTurns"`
}

type MoveSquad struct {
	AllowBodyCollisions bool `json:"allowBodyCollisions"`
	SharedElimination   bool `json:"sharedElimination"`
	SharedHealth        bool `json:"sharedHealth"`
	SharedLength        bool `json:"sharedLength"`
}

type MoveBoard struct {
	Height  int32             `json:"height"`
	Width   int32             `json:"width"`
	Food    []MoveCoord       `json:"food"`
	Snakes  []MoveBattlesnake `json:"snakes"`
	Hazards []MoveCoord       `json:"hazards"`
}

type MoveBattlesnake struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Health  int32       `json:"health"`
	Body    []MoveCoord `json:"body"`
	Head    MoveCoord   `json:"head"`
	Length  int32       `json:"length"`
	Latency string      `json:"latency"`
	Shout   string      `json:"shout"`
	Squad   string      `json:"squad"`
}

type MoveCoord struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type MoveBattlesnakeResponse struct {
	Move  string `json:"move"`
	Shout string `json:"shout,omitempty"`
}

// Compressed format - currently using zip until a better format is implemented

// Compress contents using zip archive (stored in buf)
func Encode(game *ViewGame, buf *bytes.Buffer) error {
	contents, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("error marshaling ViewGame to json: %s", err)
	}
	w := zip.NewWriter(buf)
	f, err := w.Create("game.json")
	if err != nil {
		return fmt.Errorf("error adding file to zip archive: %s", err)
	}
	_, err = f.Write([]byte(contents))
	if err != nil {
		return fmt.Errorf("error writing contents to zip archive: %s", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("error closing zip archive: %s", err)
	}
	return nil
}

// Uncompress data for a game
func Decode(data []byte) (*ViewGame, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("error creating new zip reader: %s", err)
	}
	// fmt.Printf("zip archive contains %d files\n", len(r.File))
	if len(r.File) != 1 {
		return nil, fmt.Errorf("expected 1 file in zip archive, found %d", len(r.File))
	}
	rc, err := r.File[0].Open()
	if err != nil {
		return nil, fmt.Errorf("error opening zip archive file: %s", err)
	}
	defer rc.Close()
	unzipped, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("error reading compressed game: %s", err)
	}
	var game ViewGame
	err = json.Unmarshal(unzipped, &game)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling compressed game: %s", err)
	}
	return &game, nil
}

// Translation functions

func (game *ViewGame) ToMove(turn int32, snakeId string) (*MoveGameState, error) {
	frame, err := getFrame(game, turn)
	if err != nil {
		return nil, err
	}
	var you *MoveBattlesnake
	var snakes []MoveBattlesnake = make([]MoveBattlesnake, 0, len(frame.Snakes))
	for _, frameSnake := range frame.Snakes {
		if snakeId == frameSnake.ID {
			you = &MoveBattlesnake{
				ID:      frameSnake.ID,
				Name:    frameSnake.Name,
				Health:  frameSnake.Health,
				Body:    convertCoords(frameSnake.Body),
				Head:    convertCoord(frameSnake.Body[0]),
				Length:  int32(len(frameSnake.Body)),
				Latency: frameSnake.Latency,
				Shout:   frameSnake.Shout,
				Squad:   frameSnake.Squad,
			}
		}
		snakes = append(snakes, MoveBattlesnake{
			ID:      frameSnake.ID,
			Name:    frameSnake.Name,
			Health:  frameSnake.Health,
			Body:    convertCoords(frameSnake.Body),
			Head:    convertCoord(frameSnake.Body[0]),
			Length:  int32(len(frameSnake.Body)),
			Latency: frameSnake.Latency,
			Shout:   frameSnake.Shout,
			Squad:   frameSnake.Squad,
		})
	}
	if you == nil {
		return nil, errors.New("no snake ID found matching " + snakeId)
	}
	return &MoveGameState{
		Game: MoveGame{
			ID: game.Game.ID,
			Ruleset: MoveRuleset{
				Name: game.Game.Ruleset.Name,
				Settings: MoveSettings{
					FoodSpawnChance:     game.Game.Ruleset.FoodSpawnChance,
					MinimumFood:         game.Game.Ruleset.MinimumFood,
					HazardDamagePerTurn: game.Game.Ruleset.DamagePerTurn,
				},
			},
			Timeout: game.Game.Timeout,
		},
		Turn: turn,
		Board: MoveBoard{
			Width:   game.Game.Width,
			Height:  game.Game.Height,
			Snakes:  snakes,
			Food:    convertCoords(frame.Food),
			Hazards: convertCoords(frame.Hazards),
		},
		You: *you,
	}, nil
}

func convertCoord(c ViewCoord) MoveCoord {
	return MoveCoord(c)
}

func convertCoords(coords []ViewCoord) []MoveCoord {
	var result = make([]MoveCoord, 0, len(coords))
	for _, c := range coords {
		result = append(result, MoveCoord(c))
	}
	return result
}

func getFrame(game *ViewGame, turn int32) (*ViewFrame, error) {
	if len(game.Frames) < int(turn)+1 {
		return nil, fmt.Errorf("no frame found for turn %d", turn)
	}
	return &game.Frames[turn], nil
}
