package handlers

import (
	"math/rand"
	"net/http"
	"time"
)

type meowResponse struct {
	Cat   string `json:"cat"`   // "Bincho" | "Kanpachi"
	Breed string `json:"breed"` // "siamese" | "bengal"
	Sound string `json:"sound"`
	Mood  string `json:"mood"`
}

// Bincho (シャム猫): vocal, expressive, conversational. Famous for talking back.
var binchoSounds = []meowResponse{
	{Cat: "Bincho", Breed: "siamese", Sound: "にゃーん", Mood: "ご機嫌"},
	{Cat: "Bincho", Breed: "siamese", Sound: "んにゃー", Mood: "甘えたい"},
	{Cat: "Bincho", Breed: "siamese", Sound: "なーお", Mood: "呼んでる"},
	{Cat: "Bincho", Breed: "siamese", Sound: "ぁぉーん", Mood: "切実"},
	{Cat: "Bincho", Breed: "siamese", Sound: "みゃう", Mood: "返事"},
	{Cat: "Bincho", Breed: "siamese", Sound: "ぴゅい", Mood: "甘ったれ"},
	{Cat: "Bincho", Breed: "siamese", Sound: "なぁ〜お", Mood: "おしゃべり"},
	{Cat: "Bincho", Breed: "siamese", Sound: "ふみゃ", Mood: "寝起き"},
	{Cat: "Bincho", Breed: "siamese", Sound: "ねぇ、聞いてる？", Mood: "構ってちゃん"},
	{Cat: "Bincho", Breed: "siamese", Sound: "ご飯まだ？", Mood: "催促"},
	{Cat: "Bincho", Breed: "siamese", Sound: "おはよう", Mood: "挨拶"},
	{Cat: "Bincho", Breed: "siamese", Sound: "ふみふみ…", Mood: "うっとり"},
	{Cat: "Bincho", Breed: "siamese", Sound: "(尻尾ゆらゆら)", Mood: "観察中"},
	{Cat: "Bincho", Breed: "siamese", Sound: "にゃーにゃー", Mood: "話したい"},
	{Cat: "Bincho", Breed: "siamese", Sound: "んむぁ", Mood: "あくび"},
}

// Kanpachi (ベンガル): wild voice, chirps, growls, hunting sounds.
var kanpachiSounds = []meowResponse{
	{Cat: "Kanpachi", Breed: "bengal", Sound: "ガオー", Mood: "野性全開"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "ぐるるる", Mood: "獲物を狙う"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "シャー！", Mood: "警戒"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "クルルッ", Mood: "鳥を見つけた"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "プルル", Mood: "嬉しい"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "ニャッ！", Mood: "鋭い"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "ケケケッ", Mood: "獲物に夢中"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "ぐるにゃ", Mood: "甘え混じり"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "(ダッシュ)", Mood: "全力疾走"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "(獲物発見)", Mood: "ハンター"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "ゴロゴロ…", Mood: "満足"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "うにゃ！", Mood: "突然"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "(高所から見下ろす)", Mood: "支配者"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "ハッ！", Mood: "発見"},
	{Cat: "Kanpachi", Breed: "bengal", Sound: "うぅ〜にゃっ", Mood: "うなり"},
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Meow returns a random sound from Bincho or Kanpachi (50/50 by cat).
// Adds a duet variation occasionally where both cats answer.
func Meow(w http.ResponseWriter, r *http.Request) {
	pool := binchoSounds
	if rng.Intn(2) == 0 {
		pool = kanpachiSounds
	}
	pick := pool[rng.Intn(len(pool))]

	// 1 in 8 chance of a duet
	if rng.Intn(8) == 0 {
		other := kanpachiSounds[rng.Intn(len(kanpachiSounds))]
		if pick.Cat == "Kanpachi" {
			other = binchoSounds[rng.Intn(len(binchoSounds))]
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"duet": []meowResponse{pick, other},
		})
		return
	}
	writeJSON(w, http.StatusOK, pick)
}
