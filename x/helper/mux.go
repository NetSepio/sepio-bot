package mux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Route struct {
	Pattern     string      // match pattern that should trigger this route handler
	Description string      // short description of this route
	Help        string      // detailed help string for this route
	Run         HandlerFunc // route handler function to call
}
type Context struct {
	Fields          []string
	Content         string
	IsDirected      bool
	IsPrivate       bool
	HasPrefix       bool
	HasMention      bool
	HasMentionFirst bool
}
type HandlerFunc func(*discordgo.Session, *discordgo.Message, *Context)
type Mux struct {
	Routes  []*Route
	Default *Route
	Prefix  string
}

// New returns a new Discord message route mux
func New() *Mux {
	m := &Mux{}
	m.Prefix = "-dg "
	return m
}

// Route allows you to register a route
func (m *Mux) Route(pattern, desc string, cb HandlerFunc) (*Route, error) {

	r := Route{}
	r.Pattern = pattern
	r.Description = desc
	r.Run = cb
	m.Routes = append(m.Routes, &r)

	return &r, nil
}

type AutoGenerated struct {
	siteURL       string `json:"siteURL"`
	siteTag       string `json:"siteTag"`
	siteSafety    string `json:"siteSafety"`
	infoHash      string `json:"infoHash"`
	domainAddress string `json:"domainAddress"`
	id            string `json:"id"`
	metaDataUri   string `json:"metaDataUri"`
}

func fetchQuery(msg string) []byte {

	query := fmt.Sprintf(`            
    {                
         reviews(where: { siteURL: "%s" })
         {                    siteTag                  }            }        `, msg)
	jsonData := map[string]string{"query": query}
	jsonValue, _ := json.Marshal(jsonData)
	return jsonValue
}

func (m *Mux) FuzzyMatch(msg string) string {
	var f interface{}
	// jsonData := map[string]string{
	// 	"query": `
	//         {
	//             reviews(where: { siteURL: "http://ethergift.net/" }){
	// 				siteTag
	// 			  }
	//         }
	//     `,
	// }
	// jsonValue, _ := json.Marshal(jsonData)
	query := fmt.Sprintf(`            
    {                
         reviews(where: { siteURL: "%s" })
         {                    siteTag                  }            }        `, msg)
	jsonData := map[string]string{"query": query}
	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", "https://query.graph.lazarus.network/subgraphs/name/NetSepio", bytes.NewBuffer(jsonValue))
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	defer response.Body.Close()
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	}
	data, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(data, &f)
	// fmt.Println(string(data), "..")
	mine := f.(map[string]interface{})
	foomap := mine["data"]
	v := foomap.(map[string]interface{})

	temp := make(map[string]int)
	for _, val := range v {
		for _, items := range val.([]interface{}) {
			// fmt.Println(reflect.TypeOf(newData).Kind(), "newData")
			for _, _value := range items.(map[string]interface{}) {
				if temp[_value.(string)] == 1 || temp[_value.(string)] > 1 {
					temp[_value.(string)] += 1
				} else {
					temp[_value.(string)] = 1
				}
			}
		}
	}
	// var testVal string;
	var count int = 0
	var recommend string
	for key, data := range temp {
		if data > count {
			count = data
			recommend = key
		}
	}
	// fmt.Println(recommend, count, "rec")
	if len(recommend) == 0 {
		return ""
	} else {
		return "```" + recommend
	}
}

func Extend(tempArray []string, val interface{}) {
	panic("unimplemented")
}

func (m *Mux) OnMessageCreate(ds *discordgo.Session, mc *discordgo.MessageCreate) {
	var err error
	// Ignore all messages created by the Bot account itself
	if mc.Author.ID == ds.State.User.ID {
		return
	}

	// Create Context struct that we can put various infos into
	ctx := &Context{
		Content: strings.TrimSpace(mc.Content),
	}

	// Fetch the channel for this Message
	var c *discordgo.Channel
	c, err = ds.State.Channel(mc.ChannelID)
	if err != nil {
		// Try fetching via REST API
		c, err = ds.Channel(mc.ChannelID)
		if err != nil {
			log.Printf("unable to fetch Channel for Message, %s", err)
		} else {
			// Attempt to add this channel into our State
			err = ds.State.ChannelAdd(c)
			if err != nil {
				log.Printf("error updating State with Channel, %s", err)
			}
		}
	}
	// Add Channel info into Context (if we successfully got the channel)
	if c != nil {
		if c.Type == discordgo.ChannelTypeDM {
			ctx.IsPrivate, ctx.IsDirected = true, true
		}
	}

	// Detect @name or @nick mentions
	if !ctx.IsDirected {

		// Detect if Bot was @mentioned
		for _, v := range mc.Mentions {

			if v.ID == ds.State.User.ID {

				ctx.IsDirected, ctx.HasMention = true, true

				reg := regexp.MustCompile(fmt.Sprintf("<@!?(%s)>", ds.State.User.ID))

				// Was the @mention the first part of the string?
				if reg.FindStringIndex(ctx.Content)[0] == 0 {
					ctx.HasMentionFirst = true
				}

				// strip bot mention tags from content string
				ctx.Content = reg.ReplaceAllString(ctx.Content, "")

				break
			}
		}
	}

	// Detect prefix mention
	if !ctx.IsDirected && len(m.Prefix) > 0 {

		// TODO : Must be changed to support a per-guild user defined prefix
		if strings.HasPrefix(ctx.Content, m.Prefix) {
			ctx.IsDirected, ctx.HasPrefix, ctx.HasMentionFirst = true, true, true
			ctx.Content = strings.TrimPrefix(ctx.Content, m.Prefix)
		}
	}
	// r, fl := m.FuzzyMatch(ctx.Content)
	// if r != nil {
	// 	ctx.Fields = fl
	// 	r.Run(ds, mc.Message, ctx)
	// 	return
	// }
	val := m.FuzzyMatch(ctx.Content)
	if len(val) == 0 {
		return
	} else {
		val += "```\n"
		ds.ChannelMessageSend(mc.ChannelID, val)
	}
	if m.Default != nil && (ctx.HasMentionFirst) {
		m.Default.Run(ds, mc.Message, ctx)
	}

}
