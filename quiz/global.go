package quiz

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kesuaheli/twitchgo"
	"github.com/spf13/viper"
)

var (
	lastFetch time.Time

	TwitchIRC      *twitchgo.Session
	joinedChannels map[string]*Connection = make(map[string]*Connection)
	channelMapMu   sync.RWMutex
)

func FetchQuestions() (err error) {
	const timeout = 30 * time.Second
	if time.Now().Add(timeout).Before(lastFetch) {
		return nil
	}
	lastFetch = time.Now()

	log.Println("Getting Quiz from Google Spreadsheet...")
	Categories, err = ParseFromGoogleSheets(viper.GetString("google.spreadsheetID"))
	if err != nil {
		return err
	}

	var questionCount, answerCountCorrect, answerCountWrong int
	for _, cat := range Categories {
		questionCount += len(cat.Pool)
		for _, q := range cat.Pool {
			answerCountCorrect += len(q.Correct)
			answerCountWrong += len(q.Wrong)
		}
		data, err := json.MarshalIndent(cat, "", "	")
		if err != nil {
			log.Printf("Error marshaling category '%s': %v", cat.Title, err)
			continue
		}
		err = os.WriteFile("sheets/"+cat.Title+".json", data, 0644)
		if err != nil {
			log.Printf("Error writing json file of category '%s': %v", cat.Title, err)
		}
	}

	log.Printf("Got %d quiz categories with a total of %d questions and %d correct and %d wrong answers (%d total) (%.3f%% correct)", len(Categories), questionCount, answerCountCorrect, answerCountWrong, answerCountCorrect+answerCountWrong, float64(answerCountCorrect)/float64(answerCountCorrect+answerCountWrong)*100)

	return nil
}

// MsgToVote checks if msg is a valid vote for an answer. It returns the number of the voted answer
// (indexed 1). If msg isn't valid for a vote MsgToVote returns 0.
//
// If g is nil or the current round g is pointing to is nil all possible votes are valid. With g
// containing a non-nil round MsgToVote will get the maximum vote and invalidates all votes above,
// e.g., the current question has 2 answers but msg is a valid vote for answer 3, MsgToVote will
// return 0, because 3 is not a valid choice at this point.
func MsgToVote(msg string, g *Game) int {
	var vote int
	switch msg {
	case "1", "a", "A":
		vote = 1
	case "2", "b", "B":
		vote = 2
	case "3", "c", "C":
		vote = 3
	case "4", "d", "D":
		vote = 4
	default:
		return 0
	}

	if g == nil {
		return vote
	}
	if g.Current == 0 || g.Current > len(g.Rounds) {
		return 0
	}
	if g.Rounds[g.Current-1] == nil {
		return vote
	}
	if vote > len(g.Rounds[g.Current-1].Answers) {
		return 0
	}
	return vote
}

func (c *Connection) JoinTwitchChannel(channel string) {
	channelMapMu.Lock()
	defer channelMapMu.Unlock()
	if old, ok := joinedChannels[channel]; ok {
		log.Printf("Tried to join a connection (uID: %s) with a already connected Twitch channel (%s). Overriding it with uID: %s", old.userID, channel, c.userID)
	}

	TwitchIRC.JoinChannel(channel)
	joinedChannels[channel] = c
}

// LeaveTwitchChannel leaves the twitch channel for the corresponding connection.
func (c *Connection) LeaveTwitchChannel() {
	for channel, connection := range joinedChannels {
		if connection == c {
			delete(joinedChannels, channel)
		}
	}
}

func OnTwitchChannelMessage(t *twitchgo.Session, channel string, source *twitchgo.IRCUser, msg string) {
	channelMapMu.RLock()
	defer channelMapMu.RUnlock()
	channel, _ = strings.CutPrefix(channel, "#")

	c, ok := joinedChannels[channel]
	if !ok {
		TwitchIRC.LeaveChannel(channel)
		return
	}
	c.OnTwitchChannelMessage(t, source, msg)
}
