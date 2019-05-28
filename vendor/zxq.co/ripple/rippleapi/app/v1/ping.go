package v1

import (
	"math/rand"
	"time"

	"git.zxq.co/ripple/rippleapi/common"
)

var rn = rand.New(rand.NewSource(time.Now().UnixNano()))

var kaomojis = [...]string{
	"Σ(ノ°▽°)ノ",
	"( ƅ°ਉ°)ƅ",
	"ヽ(　･∀･)ﾉ",
	"˭̡̞(◞⁎˃ᆺ˂)◞*✰",
	"(p^-^)p",
	"(ﾉ^∇^)ﾉﾟ",
	"ヽ(〃･ω･)ﾉ",
	"(۶* ‘ꆚ’)۶”",
	"（。＞ω＜）。",
	"（ﾉ｡≧◇≦）ﾉ",
	"ヾ(｡･ω･)ｼ",
	"(ﾉ･д･)ﾉ",
	".+:｡(ﾉ･ω･)ﾉﾞ",
	"Σ(*ﾉ´>ω<｡`)ﾉ",
	"ヾ（〃＾∇＾）ﾉ♪",
	"＼（＠￣∇￣＠）／",
	"＼(^▽^＠)ノ",
	"ヾ(@^▽^@)ノ",
	"(((＼（＠v＠）／)))",
	"＼(*T▽T*)／",
	"＼（＾▽＾）／",
	"＼（Ｔ∇Ｔ）／",
	"ヽ( ★ω★)ノ",
	"ヽ(；▽；)ノ",
	"ヾ(。◕ฺ∀◕ฺ)ノ",
	"ヾ(＠† ▽ †＠）ノ",
	"ヾ(＠^∇^＠)ノ",
	"ヾ(＠^▽^＠)ﾉ",
	"ヾ（＠＾▽＾＠）ノ",
	"ヾ(＠゜▽゜＠）ノ",
	"(.=^・ェ・^=)",
	"((≡^⚲͜^≡))",
	"(^･o･^)ﾉ”",
	"(^._.^)ﾉ",
	"(^人^)",
	"(=；ェ；=)",
	"(=｀ω´=)",
	"(=｀ェ´=)",
	"（=´∇｀=）",
	"(=^･^=)",
	"(=^･ｪ･^=)",
	"(=^‥^=)",
	"(=ＴェＴ=)",
	"(=ｘェｘ=)",
	"＼(=^‥^)/’`",
	"~(=^‥^)/",
	"└(=^‥^=)┐",
	"ヾ(=ﾟ･ﾟ=)ﾉ",
	"ヽ(=^･ω･^=)丿",
	"d(=^･ω･^=)b",
	"o(^・x・^)o",
	"V(=^･ω･^=)v",
	"(⁎˃ᆺ˂)",
	"(,,^・⋏・^,,)",
}

var randomSentences = [...]string{
	"Proudly sponsored by Kirotuso!",
	"The brace is on fire!",
	"deverupa ga daisuki!",
	"It works!!!!",
	"Feelin' groovy!",
	"sudo rm -rf /",
	"Hi! I'm Flowey! Flowey the flower!",
	"Ripple devs are actually cats",
	"Support Howl's fund for buying a power supply for his SSD!",
	"Superman dies",
	"PP when?",
	"RWC hype",
}

func surpriseMe() string {
	return randomSentences[rn.Intn(len(randomSentences))] + " " + kaomojis[rn.Intn(len(kaomojis))]
}

type pingResponse struct {
	common.ResponseBase
	ID         int `json:"user_id"`
	Privileges int `json:"privileges"`
}

// PingGET is a message to check with the API that we are logged in, and know what are our privileges.
func PingGET(md common.MethodData) common.CodeMessager {
	var r pingResponse
	r.Code = 200

	if md.ID() == 0 {
		r.Message = "You have not given us a token, so we don't know who you are! But you can still login with /api/v1/tokens/new " + kaomojis[rn.Intn(len(kaomojis))]
	} else {
		r.Message = surpriseMe()
	}

	r.ID = md.ID()
	r.Privileges = int(md.User.Privileges)

	return r
}

type surpriseMeResponse struct {
	common.ResponseBase
	Cats [100]string `json:"cats"`
}

// SurpriseMeGET generates cute cats.
//
// ... Yes.
func SurpriseMeGET(md common.MethodData) common.CodeMessager {
	var r surpriseMeResponse
	r.Code = 200
	for i := 0; i < 100; i++ {
		r.Cats[i] = surpriseMe()
	}
	return r
}
